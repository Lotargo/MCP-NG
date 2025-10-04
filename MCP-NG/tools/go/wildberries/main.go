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
	"net/url"
	"os"
	"strings"
	"time"

	pb "mcp-ng/server/pkg/mcp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/structpb"
)

const baseURL = "http://localhost:8003" // Using the same mock server URL as the Python tool

// server implements the Tool service.
type server struct {
	pb.UnimplementedToolServer
	apiKey string
	logger *slog.Logger
}

// GetDescription returns the tool's description.
func (s *server) GetDescription(ctx context.Context, in *pb.GetDescriptionRequest) (*pb.ToolDescription, error) {
	s.logger.Info("Received request for tool description")
	return &pb.ToolDescription{
		Name:        "wildberries",
		Description: "Performs a request to the Wildberries API using a pre-configured authentication key.",
		Parameters: &pb.ToolParameters{
			Type: "object",
			Properties: map[string]*pb.ToolParameter{
				"method": {
					Type:        "string",
					Description: "HTTP method (GET, POST, PUT, DELETE, PATCH).",
				},
				"endpoint": {
					Type:        "string",
					Description: "The API endpoint, e.g., '/api/v3/orders'.",
				},
				"query_params": {
					Type:        "object",
					Description: "Optional dictionary of URL query parameters for GET requests.",
				},
				"json_body": {
					Type:        "object", // Note: Protobuf Struct can represent arrays too
					Description: "Optional JSON object or array for the request body (for POST/PUT/PATCH).",
				},
			},
			Required: []string{"method", "endpoint"},
		},
	}, nil
}

// Run executes the wildberries tool.
func (s *server) Run(ctx context.Context, in *pb.ToolRunRequest) (*pb.ToolRunResponse, error) {
	s.logger.Info("Received request to run wildberries tool", "args", in.Arguments)

	if s.apiKey == "" {
		s.logger.Error("Wildberries API key is not configured on the server environment")
		return &pb.ToolRunResponse{Error: "Wildberries API key is not configured on the server environment"}, nil
	}

	// Extract arguments
	method, ok := in.Arguments.Fields["method"].AsInterface().(string)
	if !ok {
		s.logger.Error("Invalid or missing 'method' argument")
		return &pb.ToolRunResponse{Error: "Invalid or missing 'method' argument"}, nil
	}
	endpoint, ok := in.Arguments.Fields["endpoint"].AsInterface().(string)
	if !ok || !strings.HasPrefix(endpoint, "/") {
		s.logger.Error("Invalid or missing 'endpoint' argument, must start with '/'")
		return &pb.ToolRunResponse{Error: "Invalid or missing 'endpoint' argument, must start with '/'"}, nil
	}

	var reqBody []byte
	var err error
	if jsonBody, ok := in.Arguments.Fields["json_body"]; ok {
		// The body can be an object or an array. Marshal whatever is provided.
		reqBody, err = json.Marshal(jsonBody.AsInterface())
		if err != nil {
			s.logger.Error("Failed to marshal json_body", "error", err)
			return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to marshal json_body: %v", err)}, nil
		}
	}

	fullURL := baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), fullURL, bytes.NewBuffer(reqBody))
	if err != nil {
		s.logger.Error("Failed to create request", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to create request: %v", err)}, nil
	}

	// Add query parameters
	if queryParams, ok := in.Arguments.Fields["query_params"].AsInterface().(map[string]interface{}); ok {
		params := url.Values{}
		for k, v := range queryParams {
			params.Add(k, fmt.Sprintf("%v", v))
		}
		req.URL.RawQuery = params.Encode()
	}

	// Add headers
	req.Header.Set("Authorization", s.apiKey)
	if len(reqBody) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("Request Exception", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Request Exception: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		s.logger.Error("HTTP Error", "status_code", resp.StatusCode, "body", string(bodyBytes))
		return &pb.ToolRunResponse{Error: fmt.Sprintf("HTTP Error %d: %s", resp.StatusCode, string(bodyBytes))}, nil
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Failed to read response body", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to read response body: %v", err)}, nil
	}
	if len(bodyBytes) == 0 {
		return &pb.ToolRunResponse{Result: &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "Success with no content"}}}, nil
	}

	var resultData interface{}
	if err := json.Unmarshal(bodyBytes, &resultData); err != nil {
		// If JSON unmarshal fails, return as plain text
		resultData = string(bodyBytes)
	}

	resultValue, err := structpb.NewValue(resultData)
	if err != nil {
		s.logger.Error("Error creating result value", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Error creating result value: %v", err)}, nil
	}

	return &pb.ToolRunResponse{Result: resultValue}, nil
}

type APIConfig struct {
	APIKey string `json:"api_key"`
}

type Config struct {
	Port           int       `json:"port"`
	WildberriesAPI APIConfig `json:"wildberries_api"`
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

	if config.WildberriesAPI.APIKey == "" {
		logger.Warn("Wildberries API key not set in config.json")
	}

	address := fmt.Sprintf(":%d", config.Port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		logger.Error("failed to listen", "address", address, "error", err)
		os.Exit(1)
	}

	s := grpc.NewServer()
	pb.RegisterToolServer(s, &server{
		apiKey: config.WildberriesAPI.APIKey,
		logger: logger,
	})

	// Register the health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("mcp.Tool", grpc_health_v1.HealthCheckResponse_SERVING)

	logger.Info("Wildberries gRPC tool listening", "address", address)
	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
		os.Exit(1)
	}
}