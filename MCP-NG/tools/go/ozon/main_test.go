package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"reflect"
	"testing"
	"time"

	pb "mcp-ng/server/pkg/mcp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

// startMockOzonServer creates a mock HTTP server that mimics the Ozon API for testing.
func startMockOzonServer(t *testing.T, expectedClientID, expectedAPIKey string) *http.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers
		if r.Header.Get("Client-Id") != expectedClientID {
			http.Error(w, "Missing or incorrect Client-Id header", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Api-Key") != expectedAPIKey {
			http.Error(w, "Missing or incorrect Api-Key header", http.StatusUnauthorized)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Expected POST method", http.StatusMethodNotAllowed)
			return
		}

		// Check body
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
		if _, ok := body["page"]; !ok {
			http.Error(w, "Missing 'page' in payload", http.StatusBadRequest)
			return
		}

		// Send mock response
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"result": {"items": ["item1", "item2"], "total": 2}}`)
	})

	// Use the fixed port from the original python script for the mock server
	server := &http.Server{Addr: ":8004", Handler: handler}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			t.Logf("Mock Ozon server ListenAndServe error: %v", err)
		}
	}()

	return server
}

// startTestGrpcServer starts the ozon gRPC server on a random available port.
func startTestGrpcServer(clientID, apiKey string) (string, func()) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterToolServer(s, &server{clientID: clientID, apiKey: apiKey})
	addr := lis.Addr().String()

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("gRPC server exited with error: %v", err)
		}
	}()

	return addr, func() {
		s.GracefulStop()
	}
}

func TestRunOzon(t *testing.T) {
	// 1. Setup mock credentials
	testClientID := "test-client-id"
	testAPIKey := "test-api-key"

	// 2. Start the mock Ozon HTTP server
	mockServer := startMockOzonServer(t, testClientID, testAPIKey)
	defer mockServer.Shutdown(context.Background()) // Cleanly shut down the server

	// 3. Start the gRPC server for testing
	grpcAddr, stopGrpcServer := startTestGrpcServer(testClientID, testAPIKey)
	defer stopGrpcServer()

	// 4. Connect to the gRPC server
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewToolClient(conn)

	// 5. Prepare and call the Run method
	args, _ := structpb.NewStruct(map[string]interface{}{
		"endpoint": "/v2/product/list",
		"payload":  map[string]interface{}{"page": 1, "page_size": 10},
	})

	req := &pb.ToolRunRequest{Arguments: args}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := client.Run(ctx, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res.Error != "" {
		t.Fatalf("Run returned an error: %v", res.Error)
	}

	// 6. Check the result
	expected := map[string]interface{}{
		"result": map[string]interface{}{
			"items": []interface{}{"item1", "item2"},
			"total": float64(2),
		},
	}

	got := res.Result.GetStructValue().AsMap()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("unexpected result:\ngot:  %v\nwant: %v", got, expected)
	}

	t.Logf("Successfully tested ozon tool")
}