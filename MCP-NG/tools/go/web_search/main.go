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

const tavilyAPIURL = "https://api.tavily.com/search"

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
		Name:        "web_search",
		Description: "Performs a web search using the Tavily AI search engine. Use this for up-to-date information, facts, or news.",
		Parameters: &pb.ToolParameters{
			Type: "object",
			Properties: map[string]*pb.ToolParameter{
				"query": {
					Type:        "string",
					Description: "The search query.",
				},
				"max_results": {
					Type:        "number",
					Description: "The maximum number of results to return. Defaults to 5.",
				},
			},
			Required: []string{"query"},
		},
	}, nil
}

// Run executes the web_search tool.
func (s *server) Run(ctx context.Context, in *pb.ToolRunRequest) (*pb.ToolRunResponse, error) {
	s.logger.Info("Received request to run web_search", "args", in.Arguments)

	if s.apiKey == "" {
		s.logger.Error("TAVILY_API_KEY is not set in the config")
		return &pb.ToolRunResponse{Error: "TAVILY_API_KEY is not set in the config"}, nil
	}

	query, ok := in.Arguments.Fields["query"].AsInterface().(string)
	if !ok {
		s.logger.Error("Invalid or missing 'query' argument")
		return &pb.ToolRunResponse{Error: "Invalid or missing 'query' argument"}, nil
	}

	maxResults := 5.0 // Default value
	if val, ok := in.Arguments.Fields["max_results"]; ok {
		maxResults = val.GetNumberValue()
	}

	// Prepare request body for Tavily API
	requestBody := map[string]interface{}{
		"api_key":      s.apiKey,
		"query":        query,
		"search_depth": "advanced",
		"max_results":  int(maxResults),
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		s.logger.Error("Failed to marshal request body", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to marshal request body: %v", err)}, nil
	}

	// Create and execute HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", tavilyAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		s.logger.Error("Failed to create request", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to create request: %v", err)}, nil
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("Failed to execute request", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to execute request: %v", err)}, nil
	}
	defer resp.Body.Close()

	// Handle non-200 status codes
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		s.logger.Error("Tavily API error", "status", resp.StatusCode, "body", string(bodyBytes))
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Tavily API error (status %d): %s", resp.StatusCode, string(bodyBytes))}, nil
	}

	// Parse the response
	var tavilyResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tavilyResponse); err != nil {
		s.logger.Error("Failed to decode response", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to decode response: %v", err)}, nil
	}

	// Extract the 'results' field
	results, ok := tavilyResponse["results"]
	if !ok {
		results = []interface{}{} // Return empty list if no results
	}

	resultValue, err := structpb.NewValue(results)
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
	Port       int       `json:"port"`
	TavilyAPI  APIConfig `json:"tavily_api"`
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

	if config.TavilyAPI.APIKey == "" {
		logger.Warn("TAVILY_API_KEY not set in config.json")
	}

	address := fmt.Sprintf(":%d", config.Port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		logger.Error("failed to listen", "address", address, "error", err)
		os.Exit(1)
	}

	s := grpc.NewServer()
	pb.RegisterToolServer(s, &server{
		apiKey: config.TavilyAPI.APIKey,
		logger: logger,
	})

	// Register the health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("mcp.Tool", grpc_health_v1.HealthCheckResponse_SERVING)

	logger.Info("Web_Search gRPC tool listening", "address", address)
	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
		os.Exit(1)
	}
}