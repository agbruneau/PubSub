/*
Package models defines the shared data structures for the PubSub system.

This file contains structures for structured logging and audit trail,
used by the tracker and log_monitor components.
*/
package models

import "encoding/json"

// LogLevel defines severity levels for structured logs.
type LogLevel string

const (
	LogLevelINFO  LogLevel = "INFO"
	LogLevelERROR LogLevel = "ERROR"
)

// LogEntry is the structure of a log written to `tracker.log`.
// It is designed for the "Application Health Monitoring" pattern.
// Each entry is a structured log (JSON) containing information about the
// application state (startup, shutdown, errors, metrics). This format is optimized
// for ingestion, parsing, and visualization by monitoring and alerting tools.
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`          // Log timestamp in RFC3339 format.
	Level     LogLevel               `json:"level"`              // Severity level (INFO, ERROR).
	Message   string                 `json:"message"`            // Main log message.
	Service   string                 `json:"service"`            // Emitting service name.
	Error     string                 `json:"error,omitempty"`    // Error message, if applicable.
	Metadata  map[string]interface{} `json:"metadata,omitempty"` // Additional contextual data.
}

// EventEntry is the structure of an event written to `tracker.events`.
// It implements the "Audit Trail" pattern by capturing a faithful and immutable copy
// of each message received from Kafka, with its metadata.
//
// Each entry contains the raw message, the deserialization result,
// and contextual information like topic, partition, and offset.
// This log is the source of truth for auditing, event replay, and debugging.
type EventEntry struct {
	Timestamp      string          `json:"timestamp"`            // Reception timestamp in RFC3339 format.
	EventType      string          `json:"event_type"`           // Event type (e.g., "message.received").
	KafkaTopic     string          `json:"kafka_topic"`          // Source Kafka topic.
	KafkaPartition int32           `json:"kafka_partition"`      // Source Kafka partition.
	KafkaOffset    int64           `json:"kafka_offset"`         // Message offset in the partition.
	RawMessage     string          `json:"raw_message"`          // Raw message content.
	MessageSize    int             `json:"message_size"`         // Message size in bytes.
	Deserialized   bool            `json:"deserialized"`         // Indicates if deserialization was successful.
	Error          string          `json:"error,omitempty"`      // Deserialization error, if applicable.
	OrderFull      json.RawMessage `json:"order_full,omitempty"` // Full deserialized order content.
}
