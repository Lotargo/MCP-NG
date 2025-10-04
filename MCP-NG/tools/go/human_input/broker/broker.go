package broker

// Publisher defines the interface for publishing messages.
// This allows for different implementations (e.g., WebSocket, Kafka, Redis).
type Publisher interface {
	// Publish sends a message to a specific topic.
	Publish(topic string, message []byte) error
	// Close closes the connection to the broker.
	Close() error
}

// Message represents the data sent to the message broker.
type Message struct {
	TaskID string `json:"task_id"`
	Prompt string `json:"prompt"`
}