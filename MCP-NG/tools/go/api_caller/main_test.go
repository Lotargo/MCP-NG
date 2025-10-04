package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	pb "mcp-ng/server/pkg/mcp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

// startTestServer starts the api_caller gRPC server on a random available port.
func startTestServer() (string, func()) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterToolServer(s, &server{})
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

func TestRunAPICaller(t *testing.T) {
	// 1. Setup a mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Test POST request
		if r.Method == http.MethodPost {
			if r.Header.Get("X-Test-Header") != "TestValue" {
				http.Error(w, "Missing expected header", http.StatusBadRequest)
				return
			}
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "Invalid JSON body", http.StatusBadRequest)
				return
			}
			if body["message"] != "hello" {
				http.Error(w, "Unexpected JSON body", http.StatusBadRequest)
				return
			}
			fmt.Fprintln(w, `{"status": "ok", "received": "hello"}`)
			return
		}
		// Default to GET
		fmt.Fprintln(w, `{"message": "success"}`)
	}))
	defer mockServer.Close()

	// 2. Start the gRPC server for testing
	addr, stopServer := startTestServer()
	defer stopServer()

	// 3. Connect to the gRPC server
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewToolClient(conn)

	// 4. Run tests
	t.Run("GET request", func(t *testing.T) {
		args, _ := structpb.NewStruct(map[string]interface{}{
			"url": mockServer.URL,
		})
		req := &pb.ToolRunRequest{Arguments: args}
		res, err := client.Run(context.Background(), req)
		if err != nil || res.Error != "" {
			t.Fatalf("GET request failed: %v, %s", err, res.Error)
		}

		expected := map[string]interface{}{"message": "success"}
		got := res.Result.GetStructValue().AsMap()
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("unexpected result for GET:\ngot:  %v\nwant: %v", got, expected)
		}
	})

	t.Run("POST request", func(t *testing.T) {
		args, _ := structpb.NewStruct(map[string]interface{}{
			"url":    mockServer.URL,
			"method": "POST",
			"headers": map[string]interface{}{
				"X-Test-Header": "TestValue",
			},
			"json_body": map[string]interface{}{
				"message": "hello",
			},
		})
		req := &pb.ToolRunRequest{Arguments: args}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		res, err := client.Run(ctx, req)
		if err != nil || res.Error != "" {
			t.Fatalf("POST request failed: %v, %s", err, res.Error)
		}

		expected := map[string]interface{}{"status": "ok", "received": "hello"}
		got := res.Result.GetStructValue().AsMap()
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("unexpected result for POST:\ngot:  %v\nwant: %v", got, expected)
		}
	})
}