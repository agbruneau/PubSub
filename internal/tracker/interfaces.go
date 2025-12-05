package tracker

import (
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// KafkaConsumer définit l'interface pour les opérations du consommateur Kafka.
// Cette abstraction permet l'injection de dépendances et simplifie les tests.
type KafkaConsumer interface {
	// SubscribeTopics abonne le consommateur à un ensemble de sujets.
	//
	// Paramètres:
	//   - topics: Liste des sujets auxquels s'abonner.
	//   - rebalanceCb: Callback de rééquilibrage (optionnel).
	//
	// Retourne:
	//   - error: Une erreur si l'abonnement échoue.
	SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error

	// ReadMessage lit le prochain message depuis Kafka.
	// Il bloque jusqu'à l'expiration du délai spécifié.
	//
	// Paramètres:
	//   - timeout: Le délai d'attente maximum.
	//
	// Retourne:
	//   - *kafka.Message: Le message lu (ou nil si timeout/erreur).
	//   - error: Une erreur si la lecture échoue.
	ReadMessage(timeout time.Duration) (*kafka.Message, error)

	// Close ferme le consommateur, quittant le groupe et libérant les ressources.
	//
	// Retourne:
	//   - error: Une erreur si la fermeture échoue.
	Close() error
}

// kafkaConsumerWrapper enveloppe un vrai consommateur Kafka pour implémenter l'interface.
type kafkaConsumerWrapper struct {
	consumer *kafka.Consumer // L'instance réelle du consommateur.
}

// newKafkaConsumerWrapper crée une enveloppe autour d'un vrai consommateur Kafka.
//
// Paramètres:
//   - consumer: L'instance du consommateur Kafka à envelopper.
//
// Retourne:
//   - KafkaConsumer: L'interface enveloppant le consommateur.
func newKafkaConsumerWrapper(consumer *kafka.Consumer) KafkaConsumer {
	return &kafkaConsumerWrapper{consumer: consumer}
}

// SubscribeTopics délègue l'abonnement au consommateur réel.
//
// Paramètres:
//   - topics: Les sujets.
//   - rebalanceCb: Le callback.
//
// Retourne:
//   - error: L'erreur éventuelle.
func (w *kafkaConsumerWrapper) SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error {
	return w.consumer.SubscribeTopics(topics, rebalanceCb)
}

// ReadMessage délègue la lecture au consommateur réel.
//
// Paramètres:
//   - timeout: Le délai.
//
// Retourne:
//   - *kafka.Message: Le message.
//   - error: L'erreur.
func (w *kafkaConsumerWrapper) ReadMessage(timeout time.Duration) (*kafka.Message, error) {
	return w.consumer.ReadMessage(timeout)
}

// Close délègue la fermeture au consommateur réel.
//
// Retourne:
//   - error: L'erreur.
func (w *kafkaConsumerWrapper) Close() error {
	return w.consumer.Close()
}
