package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"

	pb "mcp-ng/server/pkg/mcp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// server implements the Tool service.
type server struct {
	pb.UnimplementedToolServer
	logger *slog.Logger
}

// GetDescription returns the tool's description.
func (s *server) GetDescription(ctx context.Context, in *pb.GetDescriptionRequest) (*pb.ToolDescription, error) {
	s.logger.Info("Received request for list_directory description")
	return &pb.ToolDescription{
		Name:        "list_directory",
		Description: "Lists the contents of a specified directory.",
		Parameters: &pb.ToolParameters{
			Type: "object",
			Properties: map[string]*pb.ToolParameter{
				"path": {
					Type:        "string",
					Description: "The path to the directory to list. Defaults to the current directory.",
				},
			},
			Required: []string{}, // Path is optional
		},
	}, nil
}

// Run executes the list_directory tool.
func (s *server) Run(ctx context.Context, in *pb.ToolRunRequest) (*pb.ToolRunResponse, error) {
	s.logger.Info("Received request to run list_directory", "args", in.Arguments)

	pathArg := "." // Default to current directory
	if in.Arguments != nil && in.Arguments.Fields["path"] != nil {
		pathArg = in.Arguments.Fields["path"].GetStringValue()
	}

	// Security measure
	securePath := filepath.Clean(pathArg)
	if strings.HasPrefix(securePath, "..") {
		s.logger.Error("Access denied: path cannot be outside the project directory", "path", pathArg)
		return &pb.ToolRunResponse{Error: "Access denied: path cannot be outside the project directory"}, nil
	}
	absPath := filepath.Join("/app", securePath)

	files, err := ioutil.ReadDir(absPath)
	if err != nil {
		s.logger.Error("Error listing directory", "path", absPath, "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Error listing directory: %v", err)}, nil
	}

	var entries []interface{}
	for _, file := range files {
		entry := file.Name()
		if file.IsDir() {
			entry += "/"
		}
		entries = append(entries, entry)
	}

	resultValue, err := structpb.NewValue(entries)
	if err != nil {
		s.logger.Error("Error creating result value", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Error creating result value: %v", err)}, nil
	}

	return &pb.ToolRunResponse{Result: resultValue}, nil
}

type Config struct {
	Port int `json:"port"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Read configuration
	configFile, err := os.ReadFile("config.json")
	if err != nil {
		logger.Error("failed to read config file", "error", err)
		os.Exit(1)
	}

	var config Config
	if err := json.Unmarshal(configFile, &config); err != nil {
		logger.Error("failed to parse config file", "error", err)
		os.Exit(1)
	}

	address := fmt.Sprintf(":%d", config.Port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		logger.Error("failed to listen", "address", address, "error", err)
		os.Exit(1)
	}

	s := grpc.NewServer()
	pb.RegisterToolServer(s, &server{logger: logger})

	// Register the health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("mcp.Tool", grpc_health_v1.HealthCheckResponse_SERVING)

	logger.Info("List_directory gRPC tool listening", "address", address)
	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
		os.Exit(1)
	}
}