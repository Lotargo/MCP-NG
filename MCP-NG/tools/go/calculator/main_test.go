package main

import (
	"context"
	"io"
	"log"
	"log/slog"
	"net"
	"testing"
	"time"

	pb "mcp-ng/server/pkg/mcp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

// startTestServer starts the calculator gRPC server on a random available port.
// It returns the address of the server and a function to stop it.
func startTestServer() (string, func()) {
	lis, err := net.Listen("tcp", ":0") // :0 means random available port
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Create a logger that discards output for clean test logs
	nullLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	s := grpc.NewServer()
	pb.RegisterToolServer(s, &server{logger: nullLogger})

	addr := lis.Addr().String()

	// Run server in a goroutine
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("server exited with error: %v", err)
		}
	}()

	// Return the address and a function to stop the server
	return addr, func() {
		s.GracefulStop()
	}
}

func TestRunCalculator(t *testing.T) {
	// Start the server for testing
	addr, stopServer := startTestServer()
	defer stopServer()

	// Connect to the server
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewToolClient(conn)

	// Prepare the test case
	expression := "(2 + 3) * 4"
	expectedResult := float64(20)

	args, err := structpb.NewStruct(map[string]interface{}{
		"expression": expression,
	})
	if err != nil {
		t.Fatalf("failed to create args struct: %v", err)
	}

	req := &pb.ToolRunRequest{
		Name:      "calculator",
		Arguments: args,
	}

	// Call the Run method
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := client.Run(ctx, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Check for errors in the response
	if res.Error != "" {
		t.Fatalf("Run returned an error: %v", res.Error)
	}

	// Check the result
	result := res.Result.AsInterface()
	if result != expectedResult {
		t.Errorf("unexpected result: got %v, want %v", result, expectedResult)
	}

	t.Logf("Successfully tested calculator tool with expression '%s', got %v", expression, result)
}