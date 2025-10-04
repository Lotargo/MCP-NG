package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/Knetic/govaluate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/structpb"

	pb "mcp-ng/server/pkg/mcp"
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
		Name:        "calculator",
		Description: "A tool that evaluates mathematical expressions. Supports basic arithmetic (+, -, *, /) and parentheses.",
		Parameters: &pb.ToolParameters{
			Type: "object",
			Properties: map[string]*pb.ToolParameter{
				"expression": {
					Type:        "string",
					Description: "The mathematical expression to evaluate, e.g., '(2 + 2) * 4'.",
				},
			},
			Required: []string{"expression"},
		},
	}, nil
}

// Run executes the calculator tool.
func (s *server) Run(ctx context.Context, in *pb.ToolRunRequest) (*pb.ToolRunResponse, error) {
	s.logger.Info("Received request to run calculator", "args", in.Arguments)

	expressionStr, ok := in.Arguments.Fields["expression"].AsInterface().(string)
	if !ok {
		s.logger.Error("Invalid or missing 'expression' argument")
		return &pb.ToolRunResponse{Error: "Invalid or missing 'expression' argument"}, nil
	}

	expression, err := govaluate.NewEvaluableExpression(expressionStr)
	if err != nil {
		s.logger.Error("Invalid expression format", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Invalid expression format: %v", err)}, nil
	}

	result, err := expression.Evaluate(nil)
	if err != nil {
		s.logger.Error("Error evaluating expression", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Error evaluating expression: %v", err)}, nil
	}

	resultValue, err := structpb.NewValue(result)
	if err != nil {
		s.logger.Error("Error converting result to protobuf value", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Error converting result to protobuf value: %v", err)}, nil
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


	healthServer := health.NewServer()

	// 1. Set the status for the specific service that the main server will query.
	healthServer.SetServingStatus("mcp.Tool", grpc_health_v1.HealthCheckResponse_SERVING)

	// 2. Register the health server with the main gRPC server.
	grpc_health_v1.RegisterHealthServer(s, healthServer)


	logger.Info("Calculator gRPC tool listening", "address", address)
	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
		os.Exit(1)
	}
}