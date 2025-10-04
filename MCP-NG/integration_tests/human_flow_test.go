package integration_tests

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	// Go can't resolve these without a real import path in go.mod
	// We will use replace directives in go.mod to make this work.
	// For now, these paths are placeholders for the logic.
	"mcp-ng/server/pkg/mcp"
	human_input_broker "mcp-ng/human_input-tool/broker"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

// NOTE: This is a simplified integration test. It doesn't re-use the `main` functions
// from the actual services to avoid circular dependencies and keep the test self-contained.
// It re-implements the core server logic for the test.

// --- Mock Human Bridge ---
type MockBridge struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

func newMockBridge() *MockBridge {
	return &MockBridge{
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		clients:    make(map[*websocket.Conn]bool),
	}
}

func (h *MockBridge) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
					go func(c *websocket.Conn) { h.unregister <- c }(client)
				}
			}
			h.mu.Unlock()
		}
	}
}
// --- End Mock Human Bridge ---

// A simplified version of the human_input tool's server for testing
type HumanInputToolServer struct {
	mcp.UnimplementedToolServer
	brokerAddress string
}

func (s *HumanInputToolServer) Run(ctx context.Context, in *mcp.ToolRunRequest) (*mcp.ToolRunResponse, error) {
	prompt, _ := in.Arguments.Fields["prompt"].AsInterface().(string)
	pub, err := human_input_broker.NewWebSocketPublisher(s.brokerAddress)
	if err != nil { return nil, err }
	defer pub.Close()

	taskID := "test-task-123" // Use a fixed ID for simplicity
	msg, _ := json.Marshal(human_input_broker.Message{TaskID: taskID, Prompt: prompt})
	pub.Publish("human_intervention_required", msg)

	result, _ := structpb.NewStruct(map[string]interface{}{"status": "waiting_for_human", "task_id": taskID})
	return &mcp.ToolRunResponse{Result: &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: result}}}, nil
}
func (s *HumanInputToolServer) GetDescription(ctx context.Context, in *mcp.GetDescriptionRequest) (*mcp.ToolDescription, error) {
	return &mcp.ToolDescription{Name: "human_input"}, nil
}


// A simplified version of the main MCP server for testing
type MCPServer struct {
	mcp.UnimplementedMCPServer
	mu          sync.RWMutex
	tools       map[string]mcp.ToolClient
	humanInputs map[string]*mcp.GetHumanInputResponse
}
func (s *MCPServer) RunTool(ctx context.Context, in *mcp.ToolRunRequest) (*mcp.ToolRunResponse, error) {
	return s.tools[in.Name].Run(ctx, in)
}
func (s *MCPServer) ProvideHumanInput(ctx context.Context, in *mcp.ProvideHumanInputRequest) (*mcp.ProvideHumanInputResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.humanInputs[in.TaskId] = &mcp.GetHumanInputResponse{Status: "completed", Response: in.Response}
	return &mcp.ProvideHumanInputResponse{Status: "received"}, nil
}
func (s *MCPServer) GetHumanInput(ctx context.Context, in *mcp.GetHumanInputRequest) (*mcp.GetHumanInputResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if resp, ok := s.humanInputs[in.TaskId]; ok { return resp, nil }
	return &mcp.GetHumanInputResponse{Status: "pending"}, nil
}


func TestHumanInputEndToEnd(t *testing.T) {
	// 1. Start the Human Bridge (WebSocket server)
	bridge := newMockBridge()
	go bridge.run()
	upgrader := websocket.Upgrader{}
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)
		bridge.register <- conn
		go func() {
			defer func() { bridge.unregister <- conn }()
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil { break }
				bridge.broadcast <- msg
			}
		}()
	}))
	defer wsServer.Close()

	// 2. Start the Human Input Tool (gRPC server)
	humanInputLis, _ := net.Listen("tcp", ":0")
	humanInputGrpc := grpc.NewServer()
	mcp.RegisterToolServer(humanInputGrpc, &HumanInputToolServer{brokerAddress: strings.TrimPrefix(wsServer.URL, "http://")})
	go humanInputGrpc.Serve(humanInputLis)
	defer humanInputGrpc.Stop()

	// 3. Start the Main MCP Server (gRPC server)
	mcpLis, _ := net.Listen("tcp", ":0")
	mcpGrpc := grpc.NewServer()
	humanInputConn, _ := grpc.NewClient(humanInputLis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	mainMcpServer := &MCPServer{
		tools:       map[string]mcp.ToolClient{"human_input": mcp.NewToolClient(humanInputConn)},
		humanInputs: make(map[string]*mcp.GetHumanInputResponse),
	}
	mcp.RegisterMCPServer(mcpGrpc, mainMcpServer)
	go mcpGrpc.Serve(mcpLis)
	defer mcpGrpc.Stop()

	// --- The Test Flow ---

	// 4. Connect a "UI Client" to the bridge
	uiConn, _, err := websocket.DefaultDialer.Dial(strings.Replace(wsServer.URL, "http", "ws", 1), nil)
	if err != nil { t.Fatalf("UI client failed to connect to bridge: %v", err) }
	defer uiConn.Close()

	// 5. Connect an "Agent Client" to the main MCP server
	agentConn, err := grpc.NewClient(mcpLis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { t.Fatalf("Agent client failed to connect to MCP: %v", err) }
	defer agentConn.Close()
	agentClient := mcp.NewMCPClient(agentConn)

	// 6. Agent calls human_input tool
	prompt := "Please approve the transaction."
	args, _ := structpb.NewStruct(map[string]interface{}{"prompt": prompt})
	runRes, err := agentClient.RunTool(context.Background(), &mcp.ToolRunRequest{Name: "human_input", Arguments: args})
	if err != nil { t.Fatalf("RunTool failed: %v", err) }

	taskID := runRes.Result.GetStructValue().AsMap()["task_id"].(string)
	if taskID == "" { t.Fatal("RunTool did not return a task_id") }

	// 7. UI client receives the prompt from the bridge
	var receivedMsg human_input_broker.Message
	uiConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msgBytes, err := uiConn.ReadMessage()
	if err != nil { t.Fatalf("UI client failed to read message from bridge: %v", err) }
	json.Unmarshal(msgBytes, &receivedMsg)

	if receivedMsg.TaskID != taskID || receivedMsg.Prompt != prompt {
		t.Fatalf("UI received incorrect message. Got: %+v", receivedMsg)
	}
	log.Println("UI client successfully received prompt.")

	// 8. Agent checks for input, should be pending
	getRes1, _ := agentClient.GetHumanInput(context.Background(), &mcp.GetHumanInputRequest{TaskId: taskID})
	if getRes1.Status != "pending" { t.Fatalf("Expected status 'pending', got '%s'", getRes1.Status) }
	log.Println("Agent correctly sees status as 'pending'.")

	// 9. UI client provides the input via the main MCP server
	humanResponse, _ := structpb.NewValue("approved")
	_, err = agentClient.ProvideHumanInput(context.Background(), &mcp.ProvideHumanInputRequest{TaskId: taskID, Response: humanResponse})
	if err != nil { t.Fatalf("ProvideHumanInput failed: %v", err) }
	log.Println("UI client successfully provided input.")

	// 10. Agent checks for input again, should be completed
	getRes2, _ := agentClient.GetHumanInput(context.Background(), &mcp.GetHumanInputRequest{TaskId: taskID})
	if getRes2.Status != "completed" { t.Fatalf("Expected status 'completed', got '%s'", getRes2.Status) }
	if getRes2.Response.GetStringValue() != "approved" { t.Fatalf("Agent received incorrect final response.")}

	log.Println("E2E test successful! Agent received the correct human response.")
}