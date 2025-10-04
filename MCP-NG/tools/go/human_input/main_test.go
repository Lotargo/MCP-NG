package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"mcp-ng/human_input-tool/broker"
	pb "mcp-ng/server/pkg/mcp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

var upgrader = websocket.Upgrader{}

// startTestWSServer creates a mock WebSocket server for testing.
// It returns the server, its address, and a channel to receive messages on.
func startTestWSServer(t *testing.T) (*httptest.Server, chan broker.Message) {
	messageChan := make(chan broker.Message, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			t.Logf("Failed to read message: %v", err)
			return
		}

		var msg broker.Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			t.Logf("Failed to unmarshal message: %v", err)
			return
		}
		messageChan <- msg
	})

	server := httptest.NewServer(handler)
	return server, messageChan
}

// startTestGrpcServer starts the human_input gRPC server on a random available port.
func startTestGrpcServer(brokerType, brokerAddr string) (string, func()) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterToolServer(s, &server{
		brokerType:    brokerType,
		brokerAddress: brokerAddr,
	})
	addr := lis.Addr().String()

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("grpc server exited with error: %v", err)
		}
	}()

	return addr, func() {
		s.GracefulStop()
	}
}

func TestRunHumanInput(t *testing.T) {
	// 1. Start the mock WebSocket server
	mockWSServer, msgChan := startTestWSServer(t)
	defer mockWSServer.Close()

	// The WS address needs to be just the host, without the scheme
	wsAddr := strings.TrimPrefix(mockWSServer.URL, "http://")

	// 2. Start the gRPC server for testing
	grpcAddr, stopGrpcServer := startTestGrpcServer("websocket", wsAddr)
	defer stopGrpcServer()

	// 3. Connect to the gRPC server
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewToolClient(conn)

	// 4. Prepare and call the Run method
	testPrompt := "Do you approve this action?"
	args, _ := structpb.NewStruct(map[string]interface{}{
		"prompt": testPrompt,
	})

	req := &pb.ToolRunRequest{Arguments: args}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := client.Run(ctx, req)
	if err != nil || res.Error != "" {
		t.Fatalf("gRPC Run failed: %v, %s", err, res.Error)
	}

	// 5. Check the gRPC response
	resMap := res.Result.GetStructValue().AsMap()
	if status, _ := resMap["status"].(string); status != "waiting_for_human" {
		t.Errorf("unexpected status: got '%s', want 'waiting_for_human'", status)
	}
	taskID, ok := resMap["task_id"].(string)
	if !ok || taskID == "" {
		t.Errorf("task_id is missing or empty in gRPC response")
	}

	// 6. Check the message received by the WebSocket server
	select {
	case receivedMsg := <-msgChan:
		if receivedMsg.TaskID != taskID {
			t.Errorf("mismatched task_id: grpc returned '%s', ws received '%s'", taskID, receivedMsg.TaskID)
		}
		if receivedMsg.Prompt != testPrompt {
			t.Errorf("mismatched prompt: expected '%s', ws received '%s'", testPrompt, receivedMsg.Prompt)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message on WebSocket server")
	}

	t.Logf("Successfully tested human_input tool. Task ID: %s", taskID)
}