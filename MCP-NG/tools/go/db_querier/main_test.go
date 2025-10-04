package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	pb "mcp-ng/server/pkg/mcp"

	_ "github.com/mattn/go-sqlite3"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

// setupTestDB creates and populates a temporary SQLite database for testing.
// It returns the full path to the database file.
func setupTestDB(t *testing.T) string {
	// t.TempDir() automatically creates a temporary directory for the test
	// and cleans it up when the test is finished. This is the modern, safe way.
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	// Create a table and insert data
	sqlStmt := `
	CREATE TABLE users (id INTEGER NOT NULL PRIMARY KEY, name TEXT);
	DELETE FROM users;
	INSERT INTO users (id, name) VALUES (1, 'Alice');
	INSERT INTO users (id, name) VALUES (2, 'Bob');
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		t.Fatalf("%q: %s\n", err, sqlStmt)
	}

	return dbPath
}

// startTestServer starts the db_querier gRPC server on a random available port.
func startTestServer(t *testing.T) (string, func()) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	// We pass a logger to avoid nil pointer panics if logging is used.
	pb.RegisterToolServer(s, &server{logger: slog.New(slog.NewTextHandler(os.Stdout, nil))})
	addr := lis.Addr().String()

	go func() {
		if err := s.Serve(lis); err != nil {
			// Don't log expected errors during graceful shutdown.
			if err != grpc.ErrServerStopped {
				t.Logf("gRPC server exited with error: %v", err)
			}
		}
	}()

	return addr, func() {
		s.GracefulStop()
	}
}

func TestRunDBQuerier(t *testing.T) {
	// 1. Setup the test database
	dbPath := setupTestDB(t)
	// No need to defer os.Remove(), t.TempDir() handles cleanup.

	// 2. Start the server for testing
	addr, stopServer := startTestServer(t)
	defer stopServer()

	// 3. Connect to the server
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewToolClient(conn)

	// 4. Prepare and call the Run method. We pass the full, absolute path.
	args, err := structpb.NewStruct(map[string]interface{}{
		"db_path": dbPath,
		"query":   "SELECT id, name FROM users ORDER BY id ASC",
	})
	if err != nil {
		t.Fatalf("failed to create args struct: %v", err)
	}

	req := &pb.ToolRunRequest{Arguments: args}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := client.Run(ctx, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res.Error != "" {
		t.Fatalf("Run returned an error: %v", res.Error)
	}

	// 5. Check the result
	// Note: The sqlite driver returns int64, which gets converted to float64
	// by the structpb package when there's no explicit type handling.
	expected := []interface{}{
		map[string]interface{}{"id": float64(1), "name": "Alice"},
		map[string]interface{}{"id": float64(2), "name": "Bob"},
	}

	// We need to handle potential nil pointers safely.
	if res.Result == nil || res.Result.GetListValue() == nil {
		t.Fatalf("Result is nil or not a list")
	}
	got := res.Result.GetListValue().AsSlice()

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("unexpected result:\ngot:  %#v\nwant: %#v", got, expected)
	}

	t.Logf("Successfully tested db_querier tool")
}
