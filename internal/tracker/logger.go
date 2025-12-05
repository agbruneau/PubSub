package tracker

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/agbruneau/PubSub/internal/config"
	"github.com/agbruneau/PubSub/pkg/models"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Logger gère l'écriture concurrente et sécurisée dans un fichier de log.
type Logger struct {
	file    *os.File
	encoder *json.Encoder
	mu      sync.Mutex
}

// NewLogger initialise un nouveau Logger pour un fichier donné.
func NewLogger(filename string) (*Logger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("impossible d'ouvrir le fichier %s: %v", filename, err)
	}
	return &Logger{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

// Log écrit une entrée structurée dans le fichier journal.
func (l *Logger) Log(level models.LogLevel, message string, metadata map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := models.LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Service:   config.TrackerServiceName,
		Metadata:  metadata,
	}
	if err := l.encoder.Encode(entry); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur d'encodage du log: %v\n", err)
	}
}

// LogError est un raccourci pour écrire un message d'erreur dans le fichier journal.
func (l *Logger) LogError(message string, err error, metadata map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	entry := models.LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     models.LogLevelERROR,
		Message:   message,
		Service:   config.TrackerServiceName,
		Error:     err.Error(),
		Metadata:  metadata,
	}
	if encodeErr := l.encoder.Encode(entry); encodeErr != nil {
		fmt.Fprintf(os.Stderr, "Erreur d'encodage du log d'erreur: %v\n", encodeErr)
	}
}

// LogEvent écrit un enregistrement complet de message dans le fichier d'événements.
// Cette fonction est le cœur de l'implémentation du modèle "Audit Trail".
// Elle est appelée pour CHAQUE message reçu, valide ou non, garantissant
// qu'aucune donnée entrante n'est perdue.
func (l *Logger) LogEvent(msg *kafka.Message, order *models.Order, deserializationError error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	eventType := "message.received"
	deserialized := order != nil

	if deserializationError != nil {
		eventType = "message.received.deserialization_error"
	}

	event := models.EventEntry{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		EventType:      eventType,
		KafkaTopic:     *msg.TopicPartition.Topic,
		KafkaPartition: msg.TopicPartition.Partition,
		KafkaOffset:    int64(msg.TopicPartition.Offset),
		RawMessage:     string(msg.Value),
		MessageSize:    len(msg.Value),
		Deserialized:   deserialized,
	}

	if deserialized {
		orderJSON, marshalErr := json.Marshal(order)
		if marshalErr != nil {
			fmt.Fprintf(os.Stderr, "Erreur de sérialisation de la commande: %v\n", marshalErr)
		} else {
			event.OrderFull = json.RawMessage(orderJSON)
		}
	}

	if deserializationError != nil {
		event.Error = deserializationError.Error()
	}

	if err := l.encoder.Encode(event); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur d'encodage de l'événement: %v\n", err)
	}
}

// Close ferme proprement les fichiers journaux.
func (l *Logger) Close() {
	if l != nil && l.file != nil {
		if err := l.file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Erreur lors de la fermeture du fichier journal: %v\n", err)
		}
	}
}
