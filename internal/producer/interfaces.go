package producer

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// KafkaProducer defines the interface for Kafka producer operations.
// This abstraction allows dependency injection and simplifies testing.
type KafkaProducer interface {
	// Produce sends a message to Kafka asynchronously.
	//
	// Parameters:
	//   - msg: The Kafka message to send.
	//   - deliveryChan: The delivery notification channel (optional).
	//
	// Returns:
	//   - error: An error if sending fails.
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error

	// Flush waits for all messages to be delivered, up to the specified timeout in ms.
	//
	// Parameters:
	//   - timeoutMs: The maximum wait time in milliseconds.
	//
	// Returns:
	//   - int: The number of remaining messages in the queue.
	Flush(timeoutMs int) int

	// Close closes the producer and releases resources.
	Close()
}

// kafkaProducerWrapper wraps a real Kafka producer to implement the interface.
type kafkaProducerWrapper struct {
	producer *kafka.Producer // The real producer instance.
}

// newKafkaProducerWrapper creates a wrapper around a real Kafka producer.
//
// Parameters:
//   - producer: The Kafka producer instance to wrap.
//
// Returns:
//   - KafkaProducer: The interface wrapping the producer.
func newKafkaProducerWrapper(producer *kafka.Producer) KafkaProducer {
	return &kafkaProducerWrapper{producer: producer}
}

// Produce sends a message via the wrapped producer.
//
// Parameters:
//   - msg: The message to produce.
//   - deliveryChan: The notification channel.
//
// Returns:
//   - error: Any potential error.
func (w *kafkaProducerWrapper) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	return w.producer.Produce(msg, deliveryChan)
}

// Flush waits for message delivery.
//
// Parameters:
//   - timeoutMs: The timeout.
//
// Returns:
//   - int: The number of unsent messages.
func (w *kafkaProducerWrapper) Flush(timeoutMs int) int {
	return w.producer.Flush(timeoutMs)
}

// Close closes the underlying producer.
func (w *kafkaProducerWrapper) Close() {
	w.producer.Close()
}
