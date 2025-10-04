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

// startMockWBServer creates a mock HTTP server that mimics the Wildberries API for testing.
func startMockWBServer(t *testing.T, expectedAPIKey string) *http.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check auth header
		if r.Header.Get("Authorization") != expectedAPIKey {
			http.Error(w, "Missing or incorrect Authorization header", http.StatusUnauthorized)
			return
		}

		// Handle GET request for orders
		if r.Method == "GET" && r.URL.Path == "/api/v3/orders" {
			if r.URL.Query().Get("limit") != "100" {
				http.Error(w, "Missing or incorrect 'limit' query param", http.StatusBadRequest)
				return
			}
			fmt.Fprintln(w, `{"orders": [{"id": 123}], "next": 1}`)
			return
		}

		// Handle POST request for prices
		if r.Method == "POST" && r.URL.Path == "/public/api/v1/prices" {
			var body []map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "Invalid JSON body", http.StatusBadRequest)
				return
			}
			if len(body) == 0 || body[0]["nmId"] == nil {
				http.Error(w, "Invalid payload for price update", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent) // Success with no content
			return
		}

		http.Error(w, "Not Found", http.StatusNotFound)
	})

	// Use the fixed port from the original python script
	server := &http.Server{Addr: ":8003", Handler: handler}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			t.Logf("Mock WB server ListenAndServe error: %v", err)
		}
	}()

	return server
}

// startTestGrpcServer starts the wildberries gRPC server on a random available port.
func startTestGrpcServer(apiKey string) (string, func()) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterToolServer(s, &server{apiKey: apiKey})
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

func TestRunWildberries(t *testing.T) {
	testAPIKey := "test-wb-api-key"
	mockServer := startMockWBServer(t, testAPIKey)
	defer mockServer.Shutdown(context.Background())

	grpcAddr, stopGrpcServer := startTestGrpcServer(testAPIKey)
	defer stopGrpcServer()

	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewToolClient(conn)

	t.Run("GET request with query params", func(t *testing.T) {
		args, _ := structpb.NewStruct(map[string]interface{}{
			"method":       "GET",
			"endpoint":     "/api/v3/orders",
			"query_params": map[string]interface{}{"limit": 100, "next": 0},
		})
		req := &pb.ToolRunRequest{Arguments: args}
		res, err := client.Run(context.Background(), req)
		if err != nil || res.Error != "" {
			t.Fatalf("GET request failed: %v, %s", err, res.Error)
		}

		expected := map[string]interface{}{
			"orders": []interface{}{map[string]interface{}{"id": float64(123)}},
			"next":   float64(1),
		}
		got := res.Result.GetStructValue().AsMap()
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("unexpected result for GET:\ngot:  %#v\nwant: %#v", got, expected)
		}
	})

	t.Run("POST request with JSON body", func(t *testing.T) {
		args, _ := structpb.NewStruct(map[string]interface{}{
			"method":     "POST",
			"endpoint":   "/public/api/v1/prices",
			"json_body":  []interface{}{map[string]interface{}{"nmId": 12345678, "price": 1500}},
		})
		req := &pb.ToolRunRequest{Arguments: args}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		res, err := client.Run(ctx, req)
		if err != nil || res.Error != "" {
			t.Fatalf("POST request failed: %v, %s", err, res.Error)
		}

		expected := "Success with no content"
		got := res.Result.GetStringValue()
		if got != expected {
			t.Errorf("unexpected result for POST: got '%s', want '%s'", got, expected)
		}
	})
}