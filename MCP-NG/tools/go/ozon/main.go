package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	pb "mcp-ng/server/pkg/mcp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/structpb"
)

const baseURL = "http://localhost:8004" // Using the same mock server URL as the Python tool

// server implements the Tool service.
type server struct {
	pb.UnimplementedToolServer
	clientID string
	apiKey   string
	logger   *slog.Logger
}

// GetDescription returns the tool's description.
func (s *server) GetDescription(ctx context.Context, in *pb.GetDescriptionRequest) (*pb.ToolDescription, error) {
	s.logger.Info("Received request for tool description")
	return &pb.ToolDescription{
		Name:        "ozon",
		Description: "Performs a POST request to the Ozon Seller API using pre-configured credentials.",
		Parameters: &pb.ToolParameters{
			Type: "object",
			Properties: map[string]*pb.ToolParameter{
				"endpoint": {
					Type:        "string",
					Description: "The API endpoint, e.g., '/v2/product/list'.",
				},
				"payload": {
					Type:        "object",
					Description: "The JSON request body.",
				},
			},
			Required: []string{"endpoint", "payload"},
		},
	}, nil
}

// Run executes the ozon tool.
func (s *server) Run(ctx context.Context, in *pb.ToolRunRequest) (*pb.ToolRunResponse, error) {
	s.logger.Info("Received request to run ozon tool", "args", in.Arguments)

	if s.clientID == "" || s.apiKey == "" {
		s.logger.Error("Ozon API keys are not configured on the server environment")
		return &pb.ToolRunResponse{Error: "Ozon API keys are not configured on the server environment"}, nil
	}

	endpoint, ok := in.Arguments.Fields["endpoint"].AsInterface().(string)
	if !ok {
		s.logger.Error("Invalid or missing 'endpoint' argument")
		return &pb.ToolRunResponse{Error: "Invalid or missing 'endpoint' argument"}, nil
	}
	payload, ok := in.Arguments.Fields["payload"].AsInterface().(map[string]interface{})
	if !ok {
		s.logger.Error("Invalid or missing 'payload' argument")
		return &pb.ToolRunResponse{Error: "Invalid or missing 'payload' argument"}, nil
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("Failed to marshal payload", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to marshal payload: %v", err)}, nil
	}

	fullURL := baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		s.logger.Error("Failed to create request", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to create request: %v", err)}, nil
	}

	req.Header.Set("Client-Id", s.clientID)
	req.Header.Set("Api-Key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("Request Exception", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Request Exception: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		s.logger.Error("HTTP Error", "status_code", resp.StatusCode, "body", string(bodyBytes))
		return &pb.ToolRunResponse{Error: fmt.Sprintf("HTTP Error %d: %s", resp.StatusCode, string(bodyBytes))}, nil
	}

	var resultData interface{}
	if err := json.NewDecoder(resp.Body).Decode(&resultData); err != nil {
		s.logger.Error("Failed to decode response", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to decode response: %v", err)}, nil
	}

	resultValue, err := structpb.NewValue(resultData)
	if err != nil {
		s.logger.Error("Error creating result value", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Error creating result value: %v", err)}, nil
	}

	return &pb.ToolRunResponse{Result: resultValue}, nil
}

type APIConfig struct {
	ClientID string `json:"client_id"`
	APIKey   string `json:"api_key"`
}

type Config struct {
	Port     int       `json:"port"`
	OzonAPI  APIConfig `json:"ozon_api"`
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

	if config.OzonAPI.ClientID == "" || config.OzonAPI.APIKey == "" {
		logger.Warn("Ozon API credentials not set in config.json")
	}

	address := fmt.Sprintf(":%d", config.Port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		logger.Error("failed to listen", "address", address, "error", err)
		os.Exit(1)
	}

	s := grpc.NewServer()
	pb.RegisterToolServer(s, &server{
		clientID: config.OzonAPI.ClientID,
		apiKey:   config.OzonAPI.APIKey,
		logger:   logger,
	})

	// Register the health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("mcp.Tool", grpc_health_v1.HealthCheckResponse_SERVING)

	logger.Info("Ozon gRPC tool listening", "address", address)
	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
		os.Exit(1)
	}
}