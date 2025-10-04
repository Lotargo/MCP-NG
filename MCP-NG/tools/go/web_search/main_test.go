package main

import (
	"context"
	"log"
	"net"
	"testing"
	"time"

	pb "mcp-ng/server/pkg/mcp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

// startTestServer starts the web_search gRPC server on a random available port.
func startTestServer(apiKey string) (string, func()) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterToolServer(s, &server{apiKey: apiKey})
	addr := lis.Addr().String()

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("server exited with error: %v", err)
		}
	}()

	return addr, func() {
		s.GracefulStop()
	}
}

func TestRunWebSearch(t *testing.T) {
	apiKey := "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"
	if apiKey == "" {
		t.Skip("TAVILY_API_KEY not found in env, skipping integration test.")
	}

	// 1. Start the server for testing
	addr, stopServer := startTestServer(apiKey)
	defer stopServer()

	// 2. Connect to the gRPC server
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewToolClient(conn)

	// 3. Prepare and call the Run method
	args, err := structpb.NewStruct(map[string]interface{}{
		"query":       "What is the capital of France?",
		"max_results": 2,
	})
	if err != nil {
		t.Fatalf("failed to create args struct: %v", err)
	}

	req := &pb.ToolRunRequest{Arguments: args}
	// Use a longer timeout for real network requests
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	res, err := client.Run(ctx, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res.Error != "" {
		t.Fatalf("Run returned an error: %v", res.Error)
	}

	// 4. Check the result
	results := res.Result.GetListValue().AsSlice()
	if len(results) == 0 {
		t.Errorf("expected at least one result, but got 0")
	}

	// Check if the first result has a title and url
	firstResult, ok := results[0].(map[string]interface{})
	if !ok {
		t.Fatalf("first result is not a map")
	}
	if _, ok := firstResult["title"]; !ok {
		t.Errorf("first result is missing 'title' field")
	}
	if _, ok := firstResult["url"]; !ok {
		t.Errorf("first result is missing 'url' field")
	}

	t.Logf("Successfully received %d results from Tavily API", len(results))
}