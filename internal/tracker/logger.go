//go:build kafka
// +build kafka

package tracker

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/agbruneau/PubSub/internal/config"
	"github.com/agbruneau/PubSub/pkg/models"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Logger manages concurrent and safe writing to a log file.
type Logger struct {
	file    *os.File
	encoder *json.Encoder
	mu      sync.Mutex
}

// NewLogger initializes a new Logger for a given file.
func NewLogger(filename string) (*Logger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %s: %v", filename, err)
	}
	return &Logger{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

// Log writes a structured entry to the log file.
func (l *Logger) Log(level models.LogLevel, message string, metadata map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := models.LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Service:   config.TrackerServiceName,
		Metadata:  metadata,
	}
	if err := l.encoder.Encode(entry); err != nil {
		fmt.Fprintf(os.Stderr, "Log encoding error: %v\n", err)
	}
}

// LogError is a shortcut to write an error message to the log file.
func (l *Logger) LogError(message string, err error, metadata map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	entry := models.LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     models.LogLevelERROR,
		Message:   message,
		Service:   config.TrackerServiceName,
		Error:     err.Error(),
		Metadata:  metadata,
	}
	if encodeErr := l.encoder.Encode(entry); encodeErr != nil {
		fmt.Fprintf(os.Stderr, "Error log encoding error: %v\n", encodeErr)
	}
}

// LogEvent writes a complete message record to the events file.
// This function is the core of the "Audit Trail" pattern implementation.
// It is called for EVERY message received, valid or not, ensuring
// no incoming data is lost.
func (l *Logger) LogEvent(msg *kafka.Message, order *models.Order, deserializationError error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	eventType := "message.received"
	deserialized := order != nil

	if deserializationError != nil {
		eventType = "message.received.deserialization_error"
	}

	event := models.EventEntry{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		EventType:      eventType,
		KafkaTopic:     *msg.TopicPartition.Topic,
		KafkaPartition: msg.TopicPartition.Partition,
		KafkaOffset:    int64(msg.TopicPartition.Offset),
		RawMessage:     string(msg.Value),
		MessageSize:    len(msg.Value),
		Deserialized:   deserialized,
	}

	if deserialized {
		orderJSON, marshalErr := json.Marshal(order)
		if marshalErr != nil {
			fmt.Fprintf(os.Stderr, "Order serialization error: %v\n", marshalErr)
		} else {
			event.OrderFull = json.RawMessage(orderJSON)
		}
	}

	if deserializationError != nil {
		event.Error = deserializationError.Error()
	}

	if err := l.encoder.Encode(event); err != nil {
		fmt.Fprintf(os.Stderr, "Event encoding error: %v\n", err)
	}
}

// Close properly closes the log files.
func (l *Logger) Close() {
	if l != nil && l.file != nil {
		if err := l.file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing log file: %v\n", err)
		}
	}
}
