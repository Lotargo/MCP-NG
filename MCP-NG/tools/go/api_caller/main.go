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
	"strings"
	"time"

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
		Name:        "api_caller",
		Description: "Performs an HTTP request to a specified URL (API). Use for interacting with external services via REST APIs.",
		Parameters: &pb.ToolParameters{
			Type: "object",
			Properties: map[string]*pb.ToolParameter{
				"url": {
					Type:        "string",
					Description: "The full URL of the endpoint to call.",
				},
				"method": {
					Type:        "string",
					Description: "The HTTP method (GET, POST, PUT, DELETE). Defaults to 'GET'.",
				},
				"headers": {
					Type:        "object",
					Description: "Optional dictionary of headers (e.g., for authorization).",
				},
				"json_body": {
					Type:        "object",
					Description: "Optional dictionary for the JSON request body (for POST/PUT).",
				},
			},
			Required: []string{"url"},
		},
	}, nil
}

// Run executes the api_caller tool.
func (s *server) Run(ctx context.Context, in *pb.ToolRunRequest) (*pb.ToolRunResponse, error) {
	s.logger.Info("Received request to run api_caller", "args", in.Arguments)

	// Extract arguments
	url, ok := in.Arguments.Fields["url"].AsInterface().(string)
	if !ok {
		s.logger.Error("Invalid or missing 'url' argument")
		return &pb.ToolRunResponse{Error: "Invalid or missing 'url' argument"}, nil
	}

	method := "GET"
	if m, ok := in.Arguments.Fields["method"].AsInterface().(string); ok {
		method = strings.ToUpper(m)
	}

	var reqBody []byte
	if jsonBody, ok := in.Arguments.Fields["json_body"].AsInterface().(map[string]interface{}); ok {
		var err error
		reqBody, err = json.Marshal(jsonBody)
		if err != nil {
			s.logger.Error("Failed to marshal json_body", "error", err)
			return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to marshal json_body: %v", err)}, nil
		}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		s.logger.Error("Failed to create request", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to create request: %v", err)}, nil
	}

	// Add headers
	if headers, ok := in.Arguments.Fields["headers"].AsInterface().(map[string]interface{}); ok {
		for k, v := range headers {
			if vStr, ok := v.(string); ok {
				req.Header.Set(k, vStr)
			}
		}
	}
	if len(reqBody) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("Network or HTTP error", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Network or HTTP error: %v", err)}, nil
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		s.logger.Error("HTTP Error", "status_code", resp.StatusCode, "body", string(bodyBytes))
		return &pb.ToolRunResponse{Error: fmt.Sprintf("HTTP Error %d: %s", resp.StatusCode, string(bodyBytes))}, nil
	}

	// Process response
	var resultData interface{}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Failed to read response body", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to read response body: %v", err)}, nil
	}

	// Try to unmarshal as JSON, fall back to string
	if err := json.Unmarshal(bodyBytes, &resultData); err != nil {
		resultData = string(bodyBytes)
	}

	resultValue, err := structpb.NewValue(resultData)
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

	logger.Info("API_Caller gRPC tool listening", "address", address)
	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
		os.Exit(1)
	}
}