//go:build kafka
// +build kafka

package retry

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// FailedMessage represents a message that failed processing.
type FailedMessage struct {
	OriginalTopic     string          `json:"original_topic"`
	OriginalPartition int32           `json:"original_partition"`
	OriginalOffset    int64           `json:"original_offset"`
	OriginalTimestamp time.Time       `json:"original_timestamp"`
	FailedAt          time.Time       `json:"failed_at"`
	Attempts          int             `json:"attempts"`
	LastError         string          `json:"last_error"`
	Payload           json.RawMessage `json:"payload"`
}

// DeadLetterQueue manages sending failed messages to a DLQ topic.
type DeadLetterQueue struct {
	producer *kafka.Producer
	topic    string
	enabled  bool
	mu       sync.Mutex
	stats    DLQStats
}

// DLQStats contains statistics about DLQ operations.
type DLQStats struct {
	MessagesSent  int64
	SendErrors    int64
	LastSentTime  time.Time
	LastErrorTime time.Time
}

// NewDeadLetterQueue creates a new DLQ handler.
func NewDeadLetterQueue(broker, topic string, enabled bool) (*DeadLetterQueue, error) {
	if !enabled {
		return &DeadLetterQueue{enabled: false}, nil
	}

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
		"acks":              "all",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create DLQ producer: %w", err)
	}

	dlq := &DeadLetterQueue{
		producer: producer,
		topic:    topic,
		enabled:  true,
	}

	// Start delivery report handler
	go dlq.handleDeliveryReports()

	return dlq, nil
}

// handleDeliveryReports processes delivery reports asynchronously.
func (d *DeadLetterQueue) handleDeliveryReports() {
	for e := range d.producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			d.mu.Lock()
			if ev.TopicPartition.Error != nil {
				d.stats.SendErrors++
				d.stats.LastErrorTime = time.Now()
			} else {
				d.stats.MessagesSent++
				d.stats.LastSentTime = time.Now()
			}
			d.mu.Unlock()
		}
	}
}

// Send sends a failed message to the DLQ.
func (d *DeadLetterQueue) Send(originalMsg *kafka.Message, attempts int, lastErr error) error {
	if !d.enabled {
		return nil
	}

	// Create DLQ message
	failedMsg := FailedMessage{
		OriginalTopic:     *originalMsg.TopicPartition.Topic,
		OriginalPartition: originalMsg.TopicPartition.Partition,
		OriginalOffset:    int64(originalMsg.TopicPartition.Offset),
		OriginalTimestamp: originalMsg.Timestamp,
		FailedAt:          time.Now().UTC(),
		Attempts:          attempts,
		LastError:         lastErr.Error(),
		Payload:           json.RawMessage(originalMsg.Value),
	}

	payload, err := json.Marshal(failedMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ message: %w", err)
	}

	// Send to DLQ topic
	d.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &d.topic, Partition: kafka.PartitionAny},
		Value:          payload,
		Headers: []kafka.Header{
			{Key: "original-topic", Value: []byte(failedMsg.OriginalTopic)},
			{Key: "error", Value: []byte(failedMsg.LastError)},
			{Key: "attempts", Value: []byte(fmt.Sprintf("%d", attempts))},
		},
	}, nil)

	return nil
}

// GetStats returns the current DLQ statistics.
func (d *DeadLetterQueue) GetStats() DLQStats {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.stats
}

// Close closes the DLQ producer.
func (d *DeadLetterQueue) Close() {
	if d.producer != nil {
		d.producer.Flush(5000)
		d.producer.Close()
	}
}

// IsEnabled returns whether the DLQ is enabled.
func (d *DeadLetterQueue) IsEnabled() bool {
	return d.enabled
}
