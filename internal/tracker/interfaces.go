package tracker

import (
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// KafkaConsumer defines the interface for Kafka consumer operations.
// This abstraction allows dependency injection and simplifies testing.
type KafkaConsumer interface {
	// SubscribeTopics subscribes the consumer to a set of topics.
	//
	// Parameters:
	//   - topics: List of topics to subscribe to.
	//   - rebalanceCb: Rebalance callback (optional).
	//
	// Returns:
	//   - error: An error if subscription fails.
	SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error

	// ReadMessage reads the next message from Kafka.
	// It blocks until the specified timeout expires.
	//
	// Parameters:
	//   - timeout: The maximum wait time.
	//
	// Returns:
	//   - *kafka.Message: The read message (or nil if timeout/error).
	//   - error: An error if reading fails.
	ReadMessage(timeout time.Duration) (*kafka.Message, error)

	// Close closes the consumer, leaving the group and releasing resources.
	//
	// Returns:
	//   - error: An error if closing fails.
	Close() error
}

// kafkaConsumerWrapper wraps a real Kafka consumer to implement the interface.
type kafkaConsumerWrapper struct {
	consumer *kafka.Consumer // The real consumer instance.
}

// newKafkaConsumerWrapper creates a wrapper around a real Kafka consumer.
//
// Parameters:
//   - consumer: The Kafka consumer instance to wrap.
//
// Returns:
//   - KafkaConsumer: The interface wrapping the consumer.
func newKafkaConsumerWrapper(consumer *kafka.Consumer) KafkaConsumer {
	return &kafkaConsumerWrapper{consumer: consumer}
}

// SubscribeTopics delegates subscription to the real consumer.
//
// Parameters:
//   - topics: The topics.
//   - rebalanceCb: The callback.
//
// Returns:
//   - error: Any potential error.
func (w *kafkaConsumerWrapper) SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error {
	return w.consumer.SubscribeTopics(topics, rebalanceCb)
}

// ReadMessage delegates reading to the real consumer.
//
// Parameters:
//   - timeout: The timeout.
//
// Returns:
//   - *kafka.Message: The message.
//   - error: The error.
func (w *kafkaConsumerWrapper) ReadMessage(timeout time.Duration) (*kafka.Message, error) {
	return w.consumer.ReadMessage(timeout)
}

// Close delegates closing to the real consumer.
//
// Returns:
//   - error: The error.
func (w *kafkaConsumerWrapper) Close() error {
	return w.consumer.Close()
}
