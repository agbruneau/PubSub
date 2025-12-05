package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestLogLevelConstants(t *testing.T) {
	// Verify log level constants are defined correctly
	if LogLevelINFO != "INFO" {
		t.Errorf("LogLevelINFO: expected 'INFO', got '%s'", LogLevelINFO)
	}
	if LogLevelERROR != "ERROR" {
		t.Errorf("LogLevelERROR: expected 'ERROR', got '%s'", LogLevelERROR)
	}
}

func TestLogEntry(t *testing.T) {
	now := time.Now().Format(time.RFC3339)

	entry := LogEntry{
		Timestamp: now,
		Level:     LogLevelINFO,
		Message:   "Test message",
		Service:   "test-service",
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	if entry.Timestamp != now {
		t.Errorf("Timestamp mismatch")
	}
	if entry.Level != LogLevelINFO {
		t.Errorf("Level: expected INFO, got %s", entry.Level)
	}
	if entry.Message != "Test message" {
		t.Errorf("Message mismatch")
	}
	if entry.Service != "test-service" {
		t.Errorf("Service mismatch")
	}
	if entry.Metadata["key"] != "value" {
		t.Errorf("Metadata key mismatch")
	}
}

func TestLogEntryWithError(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     LogLevelERROR,
		Message:   "Error occurred",
		Error:     "connection refused",
	}

	if entry.Error != "connection refused" {
		t.Errorf("Error: expected 'connection refused', got '%s'", entry.Error)
	}
}

func TestLogEntryWithNilMetadata(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     LogLevelERROR,
		Message:   "Error without metadata",
	}

	if entry.Metadata != nil {
		t.Error("Metadata should be nil by default")
	}
}

func TestEventEntry(t *testing.T) {
	now := time.Now().Format(time.RFC3339)

	entry := EventEntry{
		Timestamp:      now,
		EventType:      "order_received",
		KafkaTopic:     "orders",
		KafkaPartition: 0,
		KafkaOffset:    12345,
		RawMessage:     `{"id":"123"}`,
		MessageSize:    12,
		Deserialized:   true,
	}

	if entry.Timestamp != now {
		t.Errorf("Timestamp mismatch")
	}
	if entry.EventType != "order_received" {
		t.Errorf("EventType: expected 'order_received', got %s", entry.EventType)
	}
	if entry.KafkaTopic != "orders" {
		t.Errorf("KafkaTopic mismatch")
	}
	if entry.KafkaPartition != 0 {
		t.Errorf("KafkaPartition: expected 0, got %d", entry.KafkaPartition)
	}
	if entry.KafkaOffset != 12345 {
		t.Errorf("KafkaOffset: expected 12345, got %d", entry.KafkaOffset)
	}
	if entry.RawMessage != `{"id":"123"}` {
		t.Errorf("RawMessage mismatch")
	}
	if entry.MessageSize != 12 {
		t.Errorf("MessageSize: expected 12, got %d", entry.MessageSize)
	}
	if !entry.Deserialized {
		t.Error("Deserialized should be true")
	}
}

func TestEventEntryDeserializationFailed(t *testing.T) {
	entry := EventEntry{
		Timestamp:    time.Now().Format(time.RFC3339),
		EventType:    "deserialization_failed",
		KafkaOffset:  999,
		Deserialized: false,
		Error:        "invalid JSON",
	}

	if entry.Deserialized {
		t.Error("Deserialized should be false")
	}
	if entry.Error != "invalid JSON" {
		t.Errorf("Error message mismatch")
	}
}

func TestEventEntryWithOrderFull(t *testing.T) {
	orderJSON := json.RawMessage(`{"order_id":"ORD-123","items":[]}`)

	entry := EventEntry{
		Timestamp:    time.Now().Format(time.RFC3339),
		EventType:    "order_received",
		Deserialized: true,
		OrderFull:    orderJSON,
	}

	if entry.OrderFull == nil {
		t.Error("OrderFull should not be nil")
	}
	if string(entry.OrderFull) != `{"order_id":"ORD-123","items":[]}` {
		t.Errorf("OrderFull content mismatch")
	}
}

func TestLogLevelType(t *testing.T) {
	var level LogLevel = "CUSTOM"
	if level != "CUSTOM" {
		t.Errorf("LogLevel should accept custom strings")
	}
}
