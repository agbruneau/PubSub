//go:build kafka
// +build kafka

package tracker

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// newTestLogger creates a logger that writes to a buffer for testing purposes.
func newTestLogger(buf *bytes.Buffer) *Logger {
	return &Logger{
		file:    nil, // Not used in test
		encoder: json.NewEncoder(buf),
		mu:      sync.Mutex{},
	}
}

// newTestTracker creates a Tracker with test loggers for testing purposes.
func newTestTracker(eventBuf, logBuf *bytes.Buffer) *Tracker {
	cfg := &Config{
		LogFile:         "test.log",
		EventsFile:      "test.events",
		MetricsInterval: 30 * time.Second,
		ReadTimeout:     1 * time.Second,
		MaxErrors:       3,
	}

	tracker := &Tracker{
		config:      cfg,
		logLogger:   newTestLogger(logBuf),
		eventLogger: newTestLogger(eventBuf),
		metrics:     &SystemMetrics{StartTime: time.Now()},
		stopChan:    make(chan struct{}),
	}

	return tracker
}

// TestProcessMessageDeserializationFailure verifies that a message with invalid JSON
// is correctly logged as a deserialization failure.
func TestProcessMessageDeserializationFailure(t *testing.T) {
	// Setup: create a test tracker with buffers
	var eventBuf bytes.Buffer
	var logBuf bytes.Buffer
	tracker := newTestTracker(&eventBuf, &logBuf)

	// Create a Kafka message with invalid JSON
	topic := "orders"
	kafkaMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: 1, Offset: 101},
		Value:          []byte(`{"invalid-json"`),
		Timestamp:      time.Now(),
	}

	// Call the function under test
	tracker.processMessage(kafkaMsg)

	// Assert: Check the event log for correct deserialization status
	eventLogOutput := eventBuf.String()
	if !strings.Contains(eventLogOutput, `"deserialized":false`) {
		t.Errorf("Expected event log to contain '\"deserialized\":false', but it did not. Log: %s", eventLogOutput)
	}
	if !strings.Contains(eventLogOutput, `"error":"unexpected end of JSON input"`) {
		t.Errorf("Expected event log to contain the JSON parsing error, but it did not. Log: %s", eventLogOutput)
	}

	// Assert: Check the system log for a specific error message
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, `"message":"Message deserialization error"`) {
		t.Errorf("Expected system log to contain deserialization error message, but it did not. Log: %s", logOutput)
	}
}

// TestProcessMessageSuccess verifies that a valid message is correctly processed.
func TestProcessMessageSuccess(t *testing.T) {
	var eventBuf bytes.Buffer
	var logBuf bytes.Buffer
	tracker := newTestTracker(&eventBuf, &logBuf)

	topic := "orders"
	validOrder := `{"order_id":"test-123","sequence":1,"status":"pending","items":[],"customer_info":{"customer_id":"c1","name":"Test"}}`
	kafkaMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: 0, Offset: 1},
		Value:          []byte(validOrder),
		Timestamp:      time.Now(),
	}

	tracker.processMessage(kafkaMsg)

	eventLogOutput := eventBuf.String()
	if !strings.Contains(eventLogOutput, `"deserialized":true`) {
		t.Errorf("Expected event log to contain '\"deserialized\":true', but it did not. Log: %s", eventLogOutput)
	}

	// Verify metrics were updated
	if tracker.metrics.MessagesReceived != 1 {
		t.Errorf("Expected MessagesReceived to be 1, got %d", tracker.metrics.MessagesReceived)
	}
	if tracker.metrics.MessagesProcessed != 1 {
		t.Errorf("Expected MessagesProcessed to be 1, got %d", tracker.metrics.MessagesProcessed)
	}
}

// TestRecordMetrics verifies that metrics are correctly updated.
func TestRecordMetrics(t *testing.T) {
	metrics := &SystemMetrics{StartTime: time.Now()}

	// Test successful message
	metrics.recordMetrics(true, false)
	if metrics.MessagesReceived != 1 {
		t.Errorf("Expected MessagesReceived to be 1, got %d", metrics.MessagesReceived)
	}
	if metrics.MessagesProcessed != 1 {
		t.Errorf("Expected MessagesProcessed to be 1, got %d", metrics.MessagesProcessed)
	}
	if metrics.MessagesFailed != 0 {
		t.Errorf("Expected MessagesFailed to be 0, got %d", metrics.MessagesFailed)
	}

	// Test failed message
	metrics.recordMetrics(false, true)
	if metrics.MessagesReceived != 2 {
		t.Errorf("Expected MessagesReceived to be 2, got %d", metrics.MessagesReceived)
	}
	if metrics.MessagesProcessed != 1 {
		t.Errorf("Expected MessagesProcessed to be 1, got %d", metrics.MessagesProcessed)
	}
	if metrics.MessagesFailed != 1 {
		t.Errorf("Expected MessagesFailed to be 1, got %d", metrics.MessagesFailed)
	}
}

// TestNewConfig verifies that the default configuration is correctly created.
func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg.KafkaBroker == "" {
		t.Error("Expected KafkaBroker to be set")
	}
	if cfg.ConsumerGroup == "" {
		t.Error("Expected ConsumerGroup to be set")
	}
	if cfg.Topic == "" {
		t.Error("Expected Topic to be set")
	}
	if cfg.MaxErrors <= 0 {
		t.Error("Expected MaxErrors to be positive")
	}
}
