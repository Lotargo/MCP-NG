// File: MCP-NG/server/cmd/server/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	pb "mcp-ng/server/pkg/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

var logger *slog.Logger

// serverConfig holds the port configuration for the servers.
type serverConfig struct {
	GrpcPort int `json:"grpc_port"`
	HttpPort int `json:"http_port"`
}

// toolClient holds the client connection and description for a tool.
type toolClient struct {
	client       pb.ToolClient
	healthClient grpc_health_v1.HealthClient
	description  *pb.ToolDescription
	status       grpc_health_v1.HealthCheckResponse_ServingStatus
}

// server is used to implement the mcp.MCPServer interface.
type server struct {
	pb.UnimplementedMCPServer
	mu          sync.RWMutex
	tools       map[string]*toolClient
	toolCmds    []*exec.Cmd
	humanInputs map[string]*pb.GetHumanInputResponse // In-memory store for human responses
	shutdown    chan struct{}
}

// newServer creates a new server instance. It accepts the project's root path
// to reliably locate tool directories, regardless of where the binary is run from.
func newServer(projectRoot string) *server {
	s := &server{
		tools:       make(map[string]*toolClient),
		toolCmds:    make([]*exec.Cmd, 0),
		humanInputs: make(map[string]*pb.GetHumanInputResponse),
		shutdown:    make(chan struct{}),
	}
	s.discoverAndRunTools(projectRoot) // Pass the root path down
	s.startHealthChecks()
	return s
}

// loadConfig loads the server configuration from a file.
func loadConfig() *serverConfig {
	// Defaults
	config := &serverConfig{
		GrpcPort: 8090,
		HttpPort: 8002,
	}

	configFile, err := os.ReadFile("config.json")
	if err != nil {
		logger.Warn("config.json not found, using default ports", "grpc_port", config.GrpcPort, "http_port", config.HttpPort)
		return config
	}

	if err := json.Unmarshal(configFile, &config); err != nil {
		logger.Error("Failed to parse config.json, using default ports", "error", err)
		// Reset to defaults in case of partial unmarshalling
		config.GrpcPort = 8090
		config.HttpPort = 8002
	}

	logger.Info("Loaded server configuration", "grpc_port", config.GrpcPort, "http_port", config.HttpPort)
	return config
}

// toolConfig defines the structure of the config.json file for each tool.
type toolConfig struct {
	Port    int      `json:"port"`
	Command []string `json:"command"`
}

// discoverAndRunTools scans the filesystem for tools, launches them, and connects.
// It uses the provided projectRoot to build absolute paths for reliability.
func (s *server) discoverAndRunTools(projectRoot string) {
	// Build tool directory paths relative to the provided project root for robustness.
	toolDirs := []string{
		filepath.Join(projectRoot, "MCP-NG/tools/go"),
		filepath.Join(projectRoot, "MCP-NG/tools/python"),
	}
	logger.Info("Starting automatic tool discovery and launch...", "search_paths", toolDirs)

	skipTools := map[string]bool{
		"hybrid_search":     true,
		"keyword_extractor": true,
		"text_summarizer":   true,
		"text_generator":    true,
	}

	for _, dir := range toolDirs {
		// filepath.Walk will handle cases where a directory doesn't exist gracefully.
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// This can happen if a tool dir doesn't exist, which is fine.
				logger.Warn("Cannot access tool directory, skipping", "path", dir, "error", err)
				return filepath.SkipDir // Skip this directory and continue.
			}
			if info.IsDir() && filepath.Dir(path) == dir {
				toolName := filepath.Base(path)
				if skipTools[toolName] {
					logger.Warn("Skipping resource-intensive ML tool by default", "tool", toolName)
					return nil
				}
				configPath := filepath.Join(path, "config.json")
				if _, err := os.Stat(configPath); os.IsNotExist(err) {
					logger.Warn("config.json not found for tool, skipping.", "tool", toolName)
					return nil
				}
				configFile, err := os.ReadFile(configPath)
				if err != nil {
					logger.Warn("Failed to read config.json for tool", "tool", toolName, "error", err)
					return nil
				}
				var config toolConfig
				if err := json.Unmarshal(configFile, &config); err != nil {
					logger.Warn("Failed to parse config.json for tool", "tool", toolName, "error", err)
					return nil
				}
				if len(config.Command) > 0 {
					cmd := exec.Command(config.Command[0], config.Command[1:]...)
					cmd.Dir = path
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					if err := cmd.Start(); err != nil {
						logger.Error("Failed to start tool", "tool", toolName, "error", err)
						return nil
					}
					s.toolCmds = append(s.toolCmds, cmd)
					logger.Info("Started tool", "tool", toolName, "pid", cmd.Process.Pid)
				}
				addr := fmt.Sprintf("127.0.0.1:%d", config.Port)
				var conn *grpc.ClientConn
				var connErr error
				var desc *pb.ToolDescription
				var client pb.ToolClient
				var healthClient grpc_health_v1.HealthClient

				// Increased retries to handle slow tool startup, especially for tools compiled on the fly with 'go run'.
				const maxRetries = 15
				const retryDelay = 1 * time.Second

				for i := 0; i < maxRetries; i++ {
					time.Sleep(retryDelay)
					conn, connErr = grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
					if connErr != nil {
						logger.Warn("Failed to create gRPC client for tool, retrying...", "tool", toolName, "attempt", i+1, "error", connErr)
						continue
					}
					client = pb.NewToolClient(conn)
					healthClient = grpc_health_v1.NewHealthClient(conn)
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					desc, connErr = client.GetDescription(ctx, &pb.GetDescriptionRequest{})
					cancel()
					if connErr == nil {
						break
					}
					logger.Warn("Failed to get description for tool, retrying...", "tool", toolName, "attempt", i+1, "error", connErr)
					conn.Close()
				}
				if connErr != nil {
					logger.Error("Failed to connect to tool after multiple retries, giving up.", "tool", toolName, "error", connErr)
					return nil
				}
				registeredName := desc.Name
				if registeredName == "" {
					logger.Warn("Tool provided an empty name, skipping.", "tool", toolName)
					return nil
				}
				initialStatus := grpc_health_v1.HealthCheckResponse_NOT_SERVING
				resp, err := healthClient.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{Service: "mcp.Tool"})
				if err != nil {
					logger.Warn("Initial health check failed for tool", "tool", registeredName, "error", err)
				} else {
					initialStatus = resp.Status
					logger.Info("Initial health check successful for tool", "tool", registeredName, "status", initialStatus)
				}
				s.mu.Lock()
				s.tools[registeredName] = &toolClient{
					client:       client,
					healthClient: healthClient,
					description:  desc,
					status:       initialStatus,
				}
				s.mu.Unlock()
				logger.Info("Successfully registered tool", "tool", registeredName)
			}
			return nil
		})
		if err != nil && err != filepath.SkipDir {
			logger.Error("Failed to walk tool directory", "directory", dir, "error", err)
		}
	}
}

// startHealthChecks starts a goroutine to periodically check the health of all registered tools.
func (s *server) startHealthChecks() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.mu.Lock()
				for name, tool := range s.tools {
					resp, err := tool.healthClient.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{Service: "mcp.Tool"})
					if err != nil {
						logger.Warn("Health check failed for tool", "tool", name, "error", err)
						tool.status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
					} else {
						if tool.status != resp.Status {
							logger.Info("Health status changed for tool", "tool", name, "status", resp.Status)
						}
						tool.status = resp.Status
					}
				}
				s.mu.Unlock()
			case <-s.shutdown:
				logger.Info("Stopping health checks.")
				return
			}
		}
	}()
}

// cleanup kills all the tool subprocesses during a graceful shutdown.
func (s *server) cleanup() {
	logger.Info("Cleaning up tool subprocesses...")
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, cmd := range s.toolCmds {
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				logger.Error("Failed to kill process", "pid", cmd.Process.Pid, "error", err)
			} else {
				logger.Info("Killed process", "pid", cmd.Process.Pid)
			}
		}
	}
}

// ListTools returns a list of available and healthy tools.
func (s *server) ListTools(ctx context.Context, in *pb.ListToolsRequest) (*pb.ListToolsResponse, error) {
	logger.Info("Received request to list tools")
	s.mu.RLock()
	defer s.mu.RUnlock()

	var toolDescriptions []*pb.ToolDescription
	for name, t := range s.tools {
		if t.status == grpc_health_v1.HealthCheckResponse_SERVING {
			toolDescriptions = append(toolDescriptions, t.description)
		} else {
			logger.Warn("Excluding unhealthy tool from list", "tool", name, "status", t.status)
		}
	}

	return &pb.ListToolsResponse{Tools: toolDescriptions}, nil
}

// ExecuteTool runs a specific tool as part of a task.
func (s *server) ExecuteTool(ctx context.Context, in *pb.ExecuteToolRequest) (*pb.ExecuteToolResponse, error) {
	logger.Info("Received request to execute tool", "tool", in.ToolName, "task_id", in.TaskId)

	s.mu.RLock()
	tool, ok := s.tools[in.ToolName]
	s.mu.RUnlock()

	if !ok || tool.status != grpc_health_v1.HealthCheckResponse_SERVING {
		logger.Error("Attempt to run unavailable tool", "tool", in.ToolName, "task_id", in.TaskId)
		return nil, status.Errorf(codes.NotFound, "Tool '%s' not found or is not healthy.", in.ToolName)
	}

	// Call the tool's internal Run method to perform the work.
	runResp, err := tool.client.Run(ctx, &pb.ToolRunRequest{
		Name:      in.ToolName,
		Arguments: in.Arguments,
	})

	if err != nil {
		logger.Error("gRPC call to tool failed", "tool", in.ToolName, "task_id", in.TaskId, "error", err)
		return nil, status.Errorf(codes.Internal, "gRPC call to tool '%s' failed: %v", in.ToolName, err)
	}

	if runResp.Error != "" {
		logger.Error("Tool returned an error", "tool", in.ToolName, "task_id", in.TaskId, "error", runResp.Error)
		return nil, status.Errorf(codes.Aborted, "Tool '%s' returned an error: %s", in.ToolName, runResp.Error)
	}

	// Handle the result format: convert the tool's `Value` output to a `Struct` for the API response.
	resultStruct, ok := runResp.Result.GetKind().(*structpb.Value_StructValue)
	if !ok {
		// If the tool returns a non-object (e.g., a string), wrap it in an object for API compatibility.
		resultStruct = &structpb.Value_StructValue{StructValue: &structpb.Struct{
			Fields: map[string]*structpb.Value{"result": runResp.Result},
		}}
	}

	return &pb.ExecuteToolResponse{
		TaskId: in.TaskId,
		Result: resultStruct.StructValue,
	}, nil
}

// ProvideHumanInput stores the response from a human for a given task.
func (s *server) ProvideHumanInput(ctx context.Context, in *pb.ProvideHumanInputRequest) (*pb.ProvideHumanInputResponse, error) {
	logger.Info("Received human input", "task_id", in.TaskId)
	s.mu.Lock()
	defer s.mu.Unlock()

	if in.TaskId == "" {
		logger.Error("Received ProvideHumanInput request with empty task_id")
		return nil, status.Error(codes.InvalidArgument, "task_id cannot be empty")
	}

	s.humanInputs[in.TaskId] = &pb.GetHumanInputResponse{
		Status:   "completed",
		Response: in.Response,
	}

	return &pb.ProvideHumanInputResponse{Status: "received"}, nil
}

// GetHumanInput retrieves the response for a task that was waiting for human input.
func (s *server) GetHumanInput(ctx context.Context, in *pb.GetHumanInputRequest) (*pb.GetHumanInputResponse, error) {
	logger.Info("Checking for human input", "task_id", in.TaskId)
	s.mu.RLock()
	defer s.mu.RUnlock()

	if in.TaskId == "" {
		logger.Error("Received GetHumanInput request with empty task_id")
		return nil, status.Error(codes.InvalidArgument, "task_id cannot be empty")
	}

	if resp, ok := s.humanInputs[in.TaskId]; ok {
		logger.Info("Found completed response for task", "task_id", in.TaskId)
		return resp, nil
	}

	logger.Info("No response yet for task", "task_id", in.TaskId)
	return &pb.GetHumanInputResponse{Status: "pending"}, nil
}

func main() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	config := loadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// When running the server directly, the project root is the current working directory (".").
	mcpServer := newServer(".")

	var wg sync.WaitGroup

	// --- Start gRPC Server ---
	grpcAddr := fmt.Sprintf(":%d", config.GrpcPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Error("Failed to listen for gRPC", "address", grpcAddr, "error", err)
		os.Exit(1)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterMCPServer(grpcServer, mcpServer)
	reflection.Register(grpcServer)

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("gRPC server listening", "address", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			logger.Error("Failed to serve gRPC", "error", err)
		}
	}()

	// --- Start gRPC-Gateway (HTTP Server) ---
	httpAddr := fmt.Sprintf(":%d", config.HttpPort)
	grpcGatewayMux := runtime.NewServeMux()

	conn, err := grpc.DialContext(
		ctx,
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error("Failed to dial gRPC server for gateway", "address", grpcAddr, "error", err)
		os.Exit(1)
	}

	// --- THIS IS THE CORRECTED LINE ---
	if err := pb.RegisterMCPHandler(ctx, grpcGatewayMux, conn); err != nil {
		logger.Error("Failed to register gRPC-Gateway handler", "error", err)
		os.Exit(1)
	}

	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
	}).Handler(grpcGatewayMux)

	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: corsHandler,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("HTTP/REST Gateway listening", "address", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP/REST Gateway failed", "error", err)
		}
	}()

	// --- Graceful Shutdown ---
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan

	logger.Info("Shutdown signal received, gracefully shutting down servers...")
	close(mcpServer.shutdown)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	grpcServer.GracefulStop()
	mcpServer.cleanup()

	wg.Wait()
	logger.Info("All servers stopped. Exiting.")
}