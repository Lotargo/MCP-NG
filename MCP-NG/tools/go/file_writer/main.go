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
	s.logger.Info("Received request for file_writer description")
	return &pb.ToolDescription{
		Name:        "file_writer",
		Description: "Writes specified content to a file. Overwrites the file if it already exists.",
		Parameters: &pb.ToolParameters{
			Type: "object",
			Properties: map[string]*pb.ToolParameter{
				"filepath": {
					Type:        "string",
					Description: "The path to the file to be written.",
				},
				"content": {
					Type:        "string",
					Description: "The content to write to the file.",
				},
			},
			Required: []string{"filepath", "content"},
		},
	}, nil
}

// Run executes the file_writer tool.
func (s *server) Run(ctx context.Context, in *pb.ToolRunRequest) (*pb.ToolRunResponse, error) {
	s.logger.Info("Received request to run file_writer", "args", in.Arguments)

	pathArg, ok := in.Arguments.Fields["filepath"].AsInterface().(string)
	if !ok {
		s.logger.Error("Invalid or missing 'filepath' argument")
		return &pb.ToolRunResponse{Error: "Invalid or missing 'filepath' argument"}, nil
	}
	contentArg, ok := in.Arguments.Fields["content"].AsInterface().(string)
	if !ok {
		s.logger.Error("Invalid or missing 'content' argument")
		return &pb.ToolRunResponse{Error: "Invalid or missing 'content' argument"}, nil
	}

	// Security measure
	securePath := filepath.Clean(pathArg)
	if strings.HasPrefix(securePath, "..") {
		s.logger.Error("Access denied: filepath cannot be outside the project directory", "path", pathArg)
		return &pb.ToolRunResponse{Error: "Access denied: filepath cannot be outside the project directory"}, nil
	}
	absPath := filepath.Join("/app", securePath)

	err := ioutil.WriteFile(absPath, []byte(contentArg), 0644)
	if err != nil {
		s.logger.Error("Error writing to file", "path", absPath, "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Error writing to file: %v", err)}, nil
	}

	successMsg := fmt.Sprintf("Successfully wrote %d bytes to %s", len(contentArg), securePath)
	resultValue, err := structpb.NewValue(successMsg)
	if err != nil {
		s.logger.Error("Error creating success message", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Error creating success message: %v", err)}, nil
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

	logger.Info("File_writer gRPC tool listening", "address", address)
	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
		os.Exit(1)
	}
}