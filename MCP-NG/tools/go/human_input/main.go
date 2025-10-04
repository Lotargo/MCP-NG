package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/google/uuid"
	"mcp-ng/human_input-tool/broker"
	pb "mcp-ng/server/pkg/mcp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// server implements the Tool service.
type server struct {
	pb.UnimplementedToolServer
	brokerType    string
	brokerAddress string
	logger        *slog.Logger
}

// GetDescription returns the tool's description.
func (s *server) GetDescription(ctx context.Context, in *pb.GetDescriptionRequest) (*pb.ToolDescription, error) {
	s.logger.Info("Received request for tool description")
	return &pb.ToolDescription{
		Name:        "human_input",
		Description: "Sends a prompt to a human operator and waits for an asynchronous response. Use for critical or irreversible actions.",
		Parameters: &pb.ToolParameters{
			Type: "object",
			Properties: map[string]*pb.ToolParameter{
				"prompt": {
					Type:        "string",
					Description: "The question or prompt to show to the human operator.",
				},
			},
			Required: []string{"prompt"},
		},
	}, nil
}

// Run executes the human_input tool.
func (s *server) Run(ctx context.Context, in *pb.ToolRunRequest) (*pb.ToolRunResponse, error) {
	s.logger.Info("Received request to run human_input", "args", in.Arguments)

	prompt, ok := in.Arguments.Fields["prompt"].AsInterface().(string)
	if !ok {
		s.logger.Error("Invalid or missing 'prompt' argument")
		return &pb.ToolRunResponse{Error: "Invalid or missing 'prompt' argument"}, nil
	}

	// 1. Create a publisher based on config
	var pub broker.Publisher
	var err error
	switch s.brokerType {
	case "websocket":
		pub, err = broker.NewWebSocketPublisher(s.brokerAddress)
		if err != nil {
			s.logger.Error("Failed to connect to WebSocket broker", "error", err)
			return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to connect to WebSocket broker: %v", err)}, nil
		}
	default:
		s.logger.Error("Unsupported message broker type", "type", s.brokerType)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Unsupported message broker type: %s", s.brokerType)}, nil
	}
	defer pub.Close()

	// 2. Generate task ID and message
	taskID := uuid.New().String()
	msg := broker.Message{
		TaskID: taskID,
		Prompt: prompt,
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		s.logger.Error("Failed to create message", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to create message: %v", err)}, nil
	}

	// 3. Publish the message
	s.logger.Info("Publishing prompt", "task_id", taskID)
	err = pub.Publish("human_intervention_required", msgBytes)
	if err != nil {
		s.logger.Error("Failed to publish message", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to publish message: %v", err)}, nil
	}

	// 4. Return immediately with the task_id
	result, err := structpb.NewStruct(map[string]interface{}{
		"status":  "waiting_for_human",
		"task_id": taskID,
	})
	if err != nil {
		s.logger.Error("Failed to create response", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to create response: %v", err)}, nil
	}

	return &pb.ToolRunResponse{
		Result: &structpb.Value{
			Kind: &structpb.Value_StructValue{StructValue: result},
		},
	}, nil
}

type BrokerConfig struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type Config struct {
	Port   int          `json:"port"`
	Broker BrokerConfig `json:"broker"`
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
	pb.RegisterToolServer(s, &server{
		brokerType:    config.Broker.Type,
		brokerAddress: config.Broker.Address,
		logger:        logger,
	})

	// Register the health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("mcp.Tool", grpc_health_v1.HealthCheckResponse_SERVING)

	logger.Info("Human_Input gRPC tool listening", "address", address, "broker_type", config.Broker.Type, "broker_address", config.Broker.Address)
	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
		os.Exit(1)
	}
}