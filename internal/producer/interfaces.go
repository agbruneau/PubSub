package producer

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// KafkaProducer définit l'interface pour les opérations du producteur Kafka.
// Cette abstraction permet l'injection de dépendances et simplifie les tests.
type KafkaProducer interface {
	// Produce envoie un message à Kafka de manière asynchrone.
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error

	// Flush attend que tous les messages soient livrés, jusqu'au délai spécifié en ms.
	// Retourne le nombre de messages restants dans la file d'attente.
	Flush(timeoutMs int) int

	// Close ferme le producteur.
	Close()
}

// kafkaProducerWrapper enveloppe un vrai producteur Kafka pour implémenter l'interface.
type kafkaProducerWrapper struct {
	producer *kafka.Producer
}

// newKafkaProducerWrapper crée une enveloppe autour d'un vrai producteur Kafka.
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
