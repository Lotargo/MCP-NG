// File: MCP-NG/server/cmd/server/main_test.go
package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	pb "mcp-ng/server/pkg/mcp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestMain runs before any other tests in this package.
func TestMain(m *testing.M) {
	logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	os.Exit(m.Run())
}

// startTestServer is a helper function that starts the main MCP gRPC server.
func startTestServer(t *testing.T) (string, func()) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	mcpServer := newServer("../../../..") // Provide path to project root
	pb.RegisterMCPServer(grpcServer, mcpServer)
	addr := lis.Addr().String()
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("server exited with error: %v", err)
		}
	}()
	return addr, func() {
		grpcServer.GracefulStop()
		mcpServer.cleanup()
	}
}

// TestMCPServer is a single, top-level test function that sets up the server once
// and runs all other test cases as sub-tests using t.Run().
// This is the ideal structure to prevent resource leaks (like ports being in use).
func TestMCPServer(t *testing.T) {
	// 1. Setup the server and client ONCE for all sub-tests.
	addr, stopServer := startTestServer(t)
	defer stopServer()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewMCPClient(conn)
	
	// The context timeout must be long enough for all sub-tests.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 2. Wait for the server to be ready.
	// We only need to do this once.
	var calculatorReady bool
	for i := 0; i < 25; i++ { // Wait up to 25 seconds
		listResp, err := client.ListTools(ctx, &pb.ListToolsRequest{})
		if err == nil {
			for _, tool := range listResp.Tools {
				if tool.Name == "calculator" {
					calculatorReady = true
					break
				}
			}
		}
		if calculatorReady {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !calculatorReady {
		t.Fatal("timed out waiting for the 'calculator' tool to become available")
	}

	// 3. Run all test cases as sub-tests against the single, running server instance.
	t.Run("ExecuteToolFlow", func(t *testing.T) {
		// Test Case: Successful execution
		t.Run("ExecuteTool_Success", func(t *testing.T) {
			taskID := uuid.New().String()
			argsMap := map[string]interface{}{"expression": "2 + 2"}
			argsStruct, err := structpb.NewStruct(argsMap)
			if err != nil {
				t.Fatalf("failed to create arguments struct: %v", err)
			}
			req := &pb.ExecuteToolRequest{
				TaskId:    taskID,
				ToolName:  "calculator",
				Arguments: argsStruct,
			}
			res, err := client.ExecuteTool(ctx, req)
			if err != nil {
				t.Fatalf("ExecuteTool failed: %v", err)
			}
			if res.Result.Fields["result"].GetNumberValue() != 4 {
				t.Errorf("expected result '4', got '%f'", res.Result.Fields["result"].GetNumberValue())
			}
		})

		// Test Case: Tool not found
		t.Run("ExecuteTool_NotFound", func(t *testing.T) {
			req := &pb.ExecuteToolRequest{ToolName: "non_existent_tool"}
			_, err := client.ExecuteTool(ctx, req)
			if status.Code(err) != codes.NotFound {
				t.Errorf("expected gRPC status code 'NotFound', but got '%s'", status.Code(err))
			}
		})
	})

	t.Run("HumanInputFlow", func(t *testing.T) {
		taskID := uuid.New().String()
		
		// Test Case: Pending
		t.Run("GetHumanInput_Pending", func(t *testing.T) {
			req := &pb.GetHumanInputRequest{TaskId: taskID}
			res, err := client.GetHumanInput(ctx, req)
			if err != nil {
				t.Fatalf("GetHumanInput failed: %v", err)
			}
			if res.Status != "pending" {
				t.Errorf("expected status 'pending', got '%s'", res.Status)
			}
		})

		// Test Case: Completed
		t.Run("ProvideAndGetHumanInput_Completed", func(t *testing.T) {
			humanResponse, _ := structpb.NewValue("This is the human's answer")
			provideReq := &pb.ProvideHumanInputRequest{TaskId: taskID, Response: humanResponse}
			_, err := client.ProvideHumanInput(ctx, provideReq)
			if err != nil {
				t.Fatalf("ProvideHumanInput failed: %v", err)
			}
			
			getReq := &pb.GetHumanInputRequest{TaskId: taskID}
			getRes, err := client.GetHumanInput(ctx, getReq)
			if err != nil {
				t.Fatalf("GetHumanInput after providing failed: %v", err)
			}
			if getRes.Status != "completed" {
				t.Errorf("expected status 'completed', got '%s'", getRes.Status)
			}
			if getRes.Response.GetStringValue() != "This is the human's answer" {
				t.Errorf("unexpected response data: got '%s'", getRes.Response.GetStringValue())
			}
		})
	})
}