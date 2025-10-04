package main

import (
	"context"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	pb "mcp-ng/server/pkg/mcp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

// startTestServer starts the file_writer gRPC server on a random available port.
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

func TestRunFileWriter(t *testing.T) {
	// 1. Create a temporary directory and define a file path for testing
	tempDir, err := ioutil.TempDir("/app", "test-writer-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up the directory afterwards

	// The path we will pass to the tool. It's relative to /app.
	relativePath := filepath.Join(filepath.Base(tempDir), "output.txt")
	fullPath := filepath.Join("/app", relativePath)

	contentToWrite := "This content should be written to the file."

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
		"content":  contentToWrite,
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

	// 5. Verify the file was written correctly
	readContent, err := ioutil.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read back the test file: %v", err)
	}

	if string(readContent) != contentToWrite {
		t.Errorf("unexpected content: got '%s', want '%s'", string(readContent), contentToWrite)
	}

	t.Logf("Successfully tested file_writer tool, wrote to '%s'", relativePath)
}