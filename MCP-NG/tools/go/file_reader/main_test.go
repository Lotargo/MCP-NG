package main

import (
	"context"
	"io/ioutil"
	"log"
	"net"
	"os"
	"testing"
	"time"

	pb "mcp-ng/server/pkg/mcp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

// startTestServer starts the file_reader gRPC server on a random available port.
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

func TestRunFileReader(t *testing.T) {
	// 1. Create a temporary file for testing
	content := "Hello, this is a test file."
	// Note: The main code assumes a root of /app. So we create the temp file relative to that.
	// We create a temp directory to avoid polluting the root.
	tempDir, err := ioutil.TempDir("/app", "test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up the directory afterwards

	tmpfile, err := ioutil.TempFile(tempDir, "test_file_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name()) // Clean up the file

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// The path we pass to the tool should be relative to /app
	relativePath := tmpfile.Name()[len("/app/"):]

	// 2. Start the server for testing
	addr, stopServer := startTestServer()
	defer stopServer()

	// 3. Connect to the server
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewToolClient(conn)

	// 4. Prepare and call the Run method
	args, err := structpb.NewStruct(map[string]interface{}{
		"filepath": relativePath,
	})
	if err != nil {
		t.Fatalf("failed to create args struct: %v", err)
	}

	req := &pb.ToolRunRequest{Arguments: args}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := client.Run(ctx, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res.Error != "" {
		t.Fatalf("Run returned an error: %v", res.Error)
	}

	// 5. Check the result
	result := res.Result.GetStringValue()
	if result != content {
		t.Errorf("unexpected result: got '%s', want '%s'", result, content)
	}

	t.Logf("Successfully tested file_reader tool with path '%s'", relativePath)
}