package producer

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// KafkaProducer définit l'interface pour les opérations du producteur Kafka.
// Cette abstraction permet l'injection de dépendances et simplifie les tests.
type KafkaProducer interface {
	// Produce envoie un message à Kafka de manière asynchrone.
	//
	// Paramètres:
	//   - msg: Le message Kafka à envoyer.
	//   - deliveryChan: Le canal de notification de livraison (optionnel).
	//
	// Retourne:
	//   - error: Une erreur si l'envoi échoue.
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error

	// Flush attend que tous les messages soient livrés, jusqu'au délai spécifié en ms.
	//
	// Paramètres:
	//   - timeoutMs: Le délai d'attente maximum en millisecondes.
	//
	// Retourne:
	//   - int: Le nombre de messages restants dans la file d'attente.
	Flush(timeoutMs int) int

	// Close ferme le producteur et libère les ressources.
	Close()
}

// kafkaProducerWrapper enveloppe un vrai producteur Kafka pour implémenter l'interface.
type kafkaProducerWrapper struct {
	producer *kafka.Producer // L'instance réelle du producteur.
}

// newKafkaProducerWrapper crée une enveloppe autour d'un vrai producteur Kafka.
//
// Paramètres:
//   - producer: L'instance du producteur Kafka à envelopper.
//
// Retourne:
//   - KafkaProducer: L'interface enveloppant le producteur.
func newKafkaProducerWrapper(producer *kafka.Producer) KafkaProducer {
	return &kafkaProducerWrapper{producer: producer}
}

// Produce envoie un message via le producteur enveloppé.
//
// Paramètres:
//   - msg: Le message à produire.
//   - deliveryChan: Le canal de notification.
//
// Retourne:
//   - error: Une erreur éventuelle.
func (w *kafkaProducerWrapper) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	return w.producer.Produce(msg, deliveryChan)
}

// Flush attend l'envoi des messages.
//
// Paramètres:
//   - timeoutMs: Le délai.
//
// Retourne:
//   - int: Le nombre de messages non envoyés.
func (w *kafkaProducerWrapper) Flush(timeoutMs int) int {
	return w.producer.Flush(timeoutMs)
}

// Close ferme le producteur sous-jacent.
func (w *kafkaProducerWrapper) Close() {
	w.producer.Close()
}
