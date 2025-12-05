//go:build kafka
// +build kafka

package tracker

import (
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// KafkaConsumer defines the interface for Kafka consumer operations.
// This abstraction enables dependency injection and simplifies testing.
type KafkaConsumer interface {
	// SubscribeTopics subscribes the consumer to a set of topics.
	SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error

	// ReadMessage reads the next message from Kafka.
	// It blocks for up to the specified timeout.
	ReadMessage(timeout time.Duration) (*kafka.Message, error)

	// Close closes the consumer, leaving the group and releasing resources.
	Close() error
}

// kafkaConsumerWrapper wraps a real Kafka consumer to implement the interface.
type kafkaConsumerWrapper struct {
	consumer *kafka.Consumer
}

// newKafkaConsumerWrapper creates a wrapper around a real Kafka consumer.
func newKafkaConsumerWrapper(consumer *kafka.Consumer) KafkaConsumer {
	return &kafkaConsumerWrapper{consumer: consumer}
}

func (w *kafkaConsumerWrapper) SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error {
	return w.consumer.SubscribeTopics(topics, rebalanceCb)
}

func (w *kafkaConsumerWrapper) ReadMessage(timeout time.Duration) (*kafka.Message, error) {
	return w.consumer.ReadMessage(timeout)
}

func (w *kafkaConsumerWrapper) Close() error {
	return w.consumer.Close()
}
