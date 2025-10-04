package main

import (
	"context"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	pb "mcp-ng/server/pkg/mcp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

// startTestServer starts the list_directory gRPC server on a random available port.
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

func TestRunListDirectory(t *testing.T) {
	// 1. Create a temporary directory structure for testing
	tempDir, err := ioutil.TempDir("/app", "test-list-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a subdirectory and a file inside
	subDir := filepath.Join(tempDir, "sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create sub-directory: %v", err)
	}
	testFile := filepath.Join(tempDir, "file.txt")
	if err := ioutil.WriteFile(testFile, []byte("data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// The path we pass to the tool should be relative to /app
	relativePath := tempDir[len("/app/"):]
	expectedEntries := []string{"sub/", "file.txt"}
	sort.Strings(expectedEntries) // Sort for consistent comparison

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
		"path": relativePath,
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
	resultList := res.Result.GetListValue()
	if resultList == nil {
		t.Fatalf("Result is not a list")
	}

	var gotEntries []string
	for _, v := range resultList.Values {
		gotEntries = append(gotEntries, v.GetStringValue())
	}
	sort.Strings(gotEntries) // Sort for consistent comparison

	if !reflect.DeepEqual(gotEntries, expectedEntries) {
		t.Errorf("unexpected result: got %v, want %v", gotEntries, expectedEntries)
	}

	t.Logf("Successfully tested list_directory tool with path '%s'", relativePath)
}