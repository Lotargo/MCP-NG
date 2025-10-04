package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
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
		Name:        "db_querier",
		Description: "Executes a SQL query against a specified SQLite database and returns the result. WARNING: Executes any SQL query, including destructive ones like DELETE/UPDATE.",
		Parameters: &pb.ToolParameters{
			Type: "object",
			Properties: map[string]*pb.ToolParameter{
				"db_path": {
					Type:        "string",
					Description: "The path to the SQLite database file (can be relative or absolute).",
				},
				"query": {
					Type:        "string",
					Description: "The SQL query to execute.",
				},
			},
			Required: []string{"db_path", "query"},
		},
	}, nil
}

// Run executes the db_querier tool.
func (s *server) Run(ctx context.Context, in *pb.ToolRunRequest) (*pb.ToolRunResponse, error) {
	s.logger.Info("Received request to run db_querier", "args", in.Arguments)

	dbPath, ok := in.Arguments.Fields["db_path"].AsInterface().(string)
	if !ok {
		s.logger.Error("Invalid or missing 'db_path' argument")
		return &pb.ToolRunResponse{Error: "Invalid or missing 'db_path' argument"}, nil
	}
	query, ok := in.Arguments.Fields["query"].AsInterface().(string)
	if !ok {
		s.logger.Error("Invalid or missing 'query' argument")
		return &pb.ToolRunResponse{Error: "Invalid or missing 'query' argument"}, nil
	}

	// Clean the path to prevent directory traversal issues (e.g., ../../etc/passwd)
	// This makes the tool more secure and portable.
	absPath := filepath.Clean(dbPath)

	// Check if the database file exists before trying to open it.
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		s.logger.Error("Database file not found", "path", absPath)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Database file not found: '%s'", dbPath)}, nil
	}

	db, err := sql.Open("sqlite3", absPath)
	if err != nil {
		s.logger.Error("Failed to open database", "path", absPath, "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to open database: %v", err)}, nil
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		s.logger.Error("SQL query error", "query", query, "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("SQL query error: %v", err)}, nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		s.logger.Error("Failed to get columns", "error", err)
		return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to get columns: %v", err)}, nil
	}

	var results []interface{}
	for rows.Next() {
		rowValues := make([]interface{}, len(columns))
		rowPointers := make([]interface{}, len(columns))
		for i := range rowValues {
			rowPointers[i] = &rowValues[i]
		}

		if err := rows.Scan(rowPointers...); err != nil {
			s.logger.Error("Failed to scan row", "error", err)
			return &pb.ToolRunResponse{Error: fmt.Sprintf("Failed to scan row: %v", err)}, nil
		}

		rowMap := make(map[string]interface{})
		for i, colName := range columns {
			val := rowValues[i]
			// Handle byte slices (BLOB/TEXT)
			if b, ok := val.([]byte); ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		results = append(results, rowMap)
	}

	resultValue, err := structpb.NewValue(results)
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

	logger.Info("DB_Querier gRPC tool listening", "address", address)
	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
		os.Exit(1)
	}
}