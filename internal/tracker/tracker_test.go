package tracker

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// newTestLogger crée un logger qui écrit dans un buffer pour les tests.
func newTestLogger(buf *bytes.Buffer) *Logger {
	return &Logger{
		file:    nil, // Non utilisé dans les tests
		encoder: json.NewEncoder(buf),
		mu:      sync.Mutex{},
	}
}

// newTestTracker crée un Tracker avec des loggers de test pour les tests.
func newTestTracker(eventBuf, logBuf *bytes.Buffer) *Tracker {
	cfg := &Config{
		LogFile:         "test.log",
		EventsFile:      "test.events",
		MetricsInterval: 30 * time.Second,
		ReadTimeout:     1 * time.Second,
		MaxErrors:       3,
	}

	tracker := &Tracker{
		config:      cfg,
		logLogger:   newTestLogger(logBuf),
		eventLogger: newTestLogger(eventBuf),
		metrics:     &SystemMetrics{StartTime: time.Now()},
		stopChan:    make(chan struct{}),
	}

	return tracker
}

// TestProcessMessageDeserializationFailure vérifie qu'un message avec un JSON invalide
// est correctement journalisé comme un échec de désérialisation.
func TestProcessMessageDeserializationFailure(t *testing.T) {
	// Configuration: créer un tracker de test avec des buffers
	var eventBuf bytes.Buffer
	var logBuf bytes.Buffer
	tracker := newTestTracker(&eventBuf, &logBuf)

	// Créer un message Kafka avec un JSON invalide
	topic := "orders"
	kafkaMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: 1, Offset: 101},
		Value:          []byte(`{"invalid-json"`),
		Timestamp:      time.Now(),
	}

	// Appeler la fonction testée
	tracker.processMessage(kafkaMsg)

	// Vérification: Vérifier le journal des événements pour le statut correct
	eventLogOutput := eventBuf.String()
	if !strings.Contains(eventLogOutput, `"deserialized":false`) {
		t.Errorf("Attendu que le journal d'événements contienne '\"deserialized\":false', mais ce n'est pas le cas. Log: %s", eventLogOutput)
	}
	if !strings.Contains(eventLogOutput, `"error":"unexpected end of JSON input"`) {
		t.Errorf("Attendu que le journal d'événements contienne l'erreur de parsing JSON, mais ce n'est pas le cas. Log: %s", eventLogOutput)
	}

	// Vérification: Vérifier le journal système pour un message d'erreur spécifique
	// Note: Le message en français est "Erreur de désérialisation du message"
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, `"message":"Erreur de désérialisation du message"`) {
		t.Errorf("Attendu que le journal système contienne le message d'erreur de désérialisation, mais ce n'est pas le cas. Log: %s", logOutput)
	}
}

// TestProcessMessageSuccess vérifie qu'un message valide est correctement traité.
func TestProcessMessageSuccess(t *testing.T) {
	var eventBuf bytes.Buffer
	var logBuf bytes.Buffer
	tracker := newTestTracker(&eventBuf, &logBuf)

	topic := "orders"
	validOrder := `{"order_id":"test-123","sequence":1,"status":"pending","items":[],"customer_info":{"customer_id":"c1","name":"Test"}}`
	kafkaMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: 0, Offset: 1},
		Value:          []byte(validOrder),
		Timestamp:      time.Now(),
	}

	tracker.processMessage(kafkaMsg)

	eventLogOutput := eventBuf.String()
	if !strings.Contains(eventLogOutput, `"deserialized":true`) {
		t.Errorf("Attendu que le journal d'événements contienne '\"deserialized\":true', mais ce n'est pas le cas. Log: %s", eventLogOutput)
	}

	// Vérifier que les métriques ont été mises à jour
	if tracker.metrics.MessagesReceived != 1 {
		t.Errorf("Attendu que MessagesReceived soit 1, reçu %d", tracker.metrics.MessagesReceived)
	}
	if tracker.metrics.MessagesProcessed != 1 {
		t.Errorf("Attendu que MessagesProcessed soit 1, reçu %d", tracker.metrics.MessagesProcessed)
	}
}

// TestRecordMetrics vérifie que les métriques sont correctement mises à jour.
func TestRecordMetrics(t *testing.T) {
	metrics := &SystemMetrics{StartTime: time.Now()}

	// Tester un message réussi
	metrics.recordMetrics(true, false)
	if metrics.MessagesReceived != 1 {
		t.Errorf("Attendu que MessagesReceived soit 1, reçu %d", metrics.MessagesReceived)
	}
	if metrics.MessagesProcessed != 1 {
		t.Errorf("Attendu que MessagesProcessed soit 1, reçu %d", metrics.MessagesProcessed)
	}
	if metrics.MessagesFailed != 0 {
		t.Errorf("Attendu que MessagesFailed soit 0, reçu %d", metrics.MessagesFailed)
	}

	// Tester un message échoué
	metrics.recordMetrics(false, true)
	if metrics.MessagesReceived != 2 {
		t.Errorf("Attendu que MessagesReceived soit 2, reçu %d", metrics.MessagesReceived)
	}
	if metrics.MessagesProcessed != 1 {
		t.Errorf("Attendu que MessagesProcessed soit 1, reçu %d", metrics.MessagesProcessed)
	}
	if metrics.MessagesFailed != 1 {
		t.Errorf("Attendu que MessagesFailed soit 1, reçu %d", metrics.MessagesFailed)
	}
}

// TestNewConfig vérifie que la configuration par défaut est correctement créée.
func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg.KafkaBroker == "" {
		t.Error("Attendu que KafkaBroker soit défini")
	}
	if cfg.ConsumerGroup == "" {
		t.Error("Attendu que ConsumerGroup soit défini")
	}
	if cfg.Topic == "" {
		t.Error("Attendu que Topic soit défini")
	}
	if cfg.MaxErrors <= 0 {
		t.Error("Attendu que MaxErrors soit positif")
	}
}
