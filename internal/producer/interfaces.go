//go:build kafka
// +build kafka

package producer

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// KafkaProducer defines the interface for Kafka producer operations.
// This abstraction enables dependency injection and simplifies testing.
type KafkaProducer interface {
	// Produce sends a message to Kafka asynchronously.
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error

	// Flush waits for all messages to be delivered, up to timeout ms.
	// Returns the number of outstanding messages still in queue.
	Flush(timeoutMs int) int

	// Close closes the producer.
	Close()
}

// kafkaProducerWrapper wraps a real Kafka producer to implement the interface.
type kafkaProducerWrapper struct {
	producer *kafka.Producer
}

// newKafkaProducerWrapper creates a wrapper around a real Kafka producer.
func newKafkaProducerWrapper(producer *kafka.Producer) KafkaProducer {
	return &kafkaProducerWrapper{producer: producer}
}

func (w *kafkaProducerWrapper) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	return w.producer.Produce(msg, deliveryChan)
}

func (w *kafkaProducerWrapper) Flush(timeoutMs int) int {
	return w.producer.Flush(timeoutMs)
}

func (w *kafkaProducerWrapper) Close() {
	w.producer.Close()
}
