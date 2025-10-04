package broker

import (
	"fmt"
	"net/url"

	"github.com/gorilla/websocket"
)

// WebSocketPublisher implements the Publisher interface for WebSockets.
type WebSocketPublisher struct {
	conn *websocket.Conn
}

// NewWebSocketPublisher creates and connects a new WebSocket client.
func NewWebSocketPublisher(serverAddr string) (*WebSocketPublisher, error) {
	u := url.URL{Scheme: "ws", Host: serverAddr, Path: "/ws"}
	fmt.Printf("Connecting to WebSocket server at %s\n", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to dial websocket server: %w", err)
	}

	return &WebSocketPublisher{conn: c}, nil
}

// Publish sends a message to the WebSocket server.
// The concept of a "topic" is not native to a single WebSocket connection,
// so we just send the message. The server will decide how to route it.
func (p *WebSocketPublisher) Publish(topic string, message []byte) error {
	// We can optionally prepend the topic to the message if the server is designed to handle it.
	// For now, we assume the server broadcasts all messages from this connection.
	return p.conn.WriteMessage(websocket.TextMessage, message)
}

// Close closes the WebSocket connection.
func (p *WebSocketPublisher) Close() error {
	// Send a close message and wait for the server to close the connection.
	err := p.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return err
	}
	return p.conn.Close()
}