package tracker

import (
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// KafkaConsumer définit l'interface pour les opérations du consommateur Kafka.
// Cette abstraction permet l'injection de dépendances et simplifie les tests.
type KafkaConsumer interface {
	// SubscribeTopics abonne le consommateur à un ensemble de sujets.
	SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error

	// ReadMessage lit le prochain message depuis Kafka.
	// Il bloque jusqu'à l'expiration du délai spécifié.
	ReadMessage(timeout time.Duration) (*kafka.Message, error)

	// Close ferme le consommateur, quittant le groupe et libérant les ressources.
	Close() error
}

// kafkaConsumerWrapper enveloppe un vrai consommateur Kafka pour implémenter l'interface.
type kafkaConsumerWrapper struct {
	consumer *kafka.Consumer
}

// newKafkaConsumerWrapper crée une enveloppe autour d'un vrai consommateur Kafka.
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
