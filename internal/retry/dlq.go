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

// FailedMessage représente un message qui a échoué lors du traitement.
type FailedMessage struct {
	OriginalTopic     string          `json:"original_topic"`     // Le sujet Kafka d'origine.
	OriginalPartition int32           `json:"original_partition"` // La partition Kafka d'origine.
	OriginalOffset    int64           `json:"original_offset"`    // Le décalage (offset) d'origine.
	OriginalTimestamp time.Time       `json:"original_timestamp"` // L'horodatage d'origine du message.
	FailedAt          time.Time       `json:"failed_at"`          // L'heure de l'échec.
	Attempts          int             `json:"attempts"`           // Le nombre de tentatives effectuées.
	LastError         string          `json:"last_error"`         // Le dernier message d'erreur rencontré.
	Payload           json.RawMessage `json:"payload"`            // Le contenu brut du message.
}

// DeadLetterQueue gère l'envoi des messages échoués vers un topic DLQ (Dead Letter Queue).
type DeadLetterQueue struct {
	producer *kafka.Producer // Le producteur Kafka interne.
	topic    string          // Le nom du topic DLQ.
	enabled  bool            // Indique si la DLQ est activée.
	mu       sync.Mutex      // Mutex pour protéger les statistiques.
	stats    DLQStats        // Statistiques d'envoi.
}

// DLQStats contient des statistiques sur les opérations de la DLQ.
type DLQStats struct {
	MessagesSent  int64     // Nombre total de messages envoyés à la DLQ.
	SendErrors    int64     // Nombre d'erreurs lors de l'envoi à la DLQ.
	LastSentTime  time.Time // Heure du dernier envoi réussi.
	LastErrorTime time.Time // Heure de la dernière erreur.
}

// NewDeadLetterQueue crée un nouveau gestionnaire de DLQ.
//
// Paramètres:
//   - broker: L'adresse du broker Kafka.
//   - topic: Le nom du topic DLQ.
//   - enabled: Booléen pour activer ou désactiver la DLQ.
//
// Retourne:
//   - *DeadLetterQueue: Une nouvelle instance initialisée.
//   - error: Une erreur si la création du producteur échoue.
func NewDeadLetterQueue(broker, topic string, enabled bool) (*DeadLetterQueue, error) {
	if !enabled {
		return &DeadLetterQueue{enabled: false}, nil
	}

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
		"acks":              "all",
	})
	if err != nil {
		return nil, fmt.Errorf("échec de la création du producteur DLQ: %w", err)
	}

	dlq := &DeadLetterQueue{
		producer: producer,
		topic:    topic,
		enabled:  true,
	}

	// Démarrer le gestionnaire de rapports de livraison
	go dlq.handleDeliveryReports()

	return dlq, nil
}

// handleDeliveryReports traite les rapports de livraison de manière asynchrone.
// Cette méthode tourne en tâche de fond pour mettre à jour les statistiques.
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

// Send envoie un message échoué vers la DLQ.
//
// Paramètres:
//   - originalMsg: Le message Kafka original qui a échoué.
//   - attempts: Le nombre de tentatives effectuées avant l'abandon.
//   - lastErr: La dernière erreur rencontrée.
//
// Retourne:
//   - error: Une erreur si la sérialisation ou l'envoi échoue.
func (d *DeadLetterQueue) Send(originalMsg *kafka.Message, attempts int, lastErr error) error {
	if !d.enabled {
		return nil
	}

	// Création du message DLQ
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
		return fmt.Errorf("échec de la sérialisation du message DLQ: %w", err)
	}

	// Envoi vers le topic DLQ
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

// GetStats retourne les statistiques actuelles de la DLQ.
//
// Retourne:
//   - DLQStats: Les statistiques d'utilisation de la DLQ.
func (d *DeadLetterQueue) GetStats() DLQStats {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.stats
}

// Close ferme proprement le producteur DLQ.
// Elle attend (flush) les messages en attente avant de fermer.
func (d *DeadLetterQueue) Close() {
	if d.producer != nil {
		d.producer.Flush(5000)
		d.producer.Close()
	}
}

// IsEnabled retourne si la DLQ est activée.
//
// Retourne:
//   - bool: Vrai si la DLQ est activée.
func (d *DeadLetterQueue) IsEnabled() bool {
	return d.enabled
}
