package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
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
	s.logger.Info("Received request for tool description")
	return &pb.ToolDescription{
		Name:        "log_notifier",
		Description: "Writes a notification message to the standard output with a timestamp and level. Use to report task completion or important events.",
		Parameters: &pb.ToolParameters{
			Type: "object",
			Properties: map[string]*pb.ToolParameter{
				"message": {
					Type:        "string",
					Description: "The notification message text.",
				},
				"level": {
					Type:        "string",
					Description: "The importance level (e.g., INFO, WARNING, ERROR). Defaults to INFO.",
				},
			},
			Required: []string{"message"},
		},
	}, nil
}

// Run executes the log_notifier tool.
func (s *server) Run(ctx context.Context, in *pb.ToolRunRequest) (*pb.ToolRunResponse, error) {
	s.logger.Info("Received request to run log_notifier", "args", in.Arguments)

	message, ok := in.Arguments.Fields["message"].AsInterface().(string)
	if !ok {
		s.logger.Error("Invalid or missing 'message' argument")
		return &pb.ToolRunResponse{Error: "Invalid or missing 'message' argument"}, nil
	}

	levelStr := "INFO"
	if l, ok := in.Arguments.Fields["level"].AsInterface().(string); ok {
		levelStr = strings.ToUpper(l)
	}

	var level slog.Level
	switch levelStr {
	case "INFO":
		level = slog.LevelInfo
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		s.logger.Warn("Invalid log level provided, defaulting to INFO", "provided_level", levelStr)
		level = slog.LevelInfo
	}

	s.logger.Log(ctx, level, message)

	successMsg := "Notification successfully logged."
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

	logger.Info("Log_Notifier gRPC tool listening", "address", address)
	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
		os.Exit(1)
	}
}