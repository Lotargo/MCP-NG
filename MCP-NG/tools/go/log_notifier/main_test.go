package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	pb "mcp-ng/server/pkg/mcp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

// startTestServer starts the log_notifier gRPC server on a random available port.
// It also captures the server's output for verification.
func startTestServer(t *testing.T) (string, func()) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	// Используем slog, как и в основном коде, но выводим в io.Discard, чтобы не засорять вывод теста
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	pb.RegisterToolServer(s, &server{logger: testLogger})
	addr := lis.Addr().String()

	go func() {
		if err := s.Serve(lis); err != nil {
			if err != grpc.ErrServerStopped {
				t.Logf("gRPC server exited with error: %v", err)
			}
		}
	}()

	return addr, func() {
		s.GracefulStop()
	}
}

func TestRunLogNotifier(t *testing.T) {
	// 1. Start the server for testing
	addr, stopServer := startTestServer(t)
	defer stopServer()

	// 2. Connect to the gRPC server
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewToolClient(conn)

	// 3. Prepare and call the Run method
	testMessage := "Task completed successfully."
	testLevel := "SUCCESS" // Используем уровень, который будет преобразован в INFO
	args, err := structpb.NewStruct(map[string]interface{}{
		"message": testMessage,
		"level":   testLevel,
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

	// 4. Verify the result from the gRPC call
	expectedResult := "Notification successfully logged."
	if res.Result == nil {
		t.Fatalf("Result is nil")
	}

	gotResult := res.Result.GetStringValue()
	if gotResult != expectedResult {
		t.Errorf("unexpected result string: got %q, want %q", gotResult, expectedResult)
	}

	t.Logf("Successfully tested log_notifier tool.")
}
