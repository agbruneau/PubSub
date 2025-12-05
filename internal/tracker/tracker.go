/*
Package tracker fournit le consommateur de messages Kafka pour le système PubSub.

Ce paquet implémente la consommation de messages, la journalisation et la collecte de métriques,
suivant les modèles "Audit Trail" et "Application Health Monitoring".
*/
package tracker

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/agbruneau/PubSub/internal/config"
	"github.com/agbruneau/PubSub/pkg/models"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Config contient la configuration du service tracker.
// Elle peut être chargée à partir de variables d'environnement.
type Config struct {
	KafkaBroker     string        // Adresse du broker Kafka
	ConsumerGroup   string        // Groupe de consommateurs Kafka
	Topic           string        // Sujet Kafka à consommer
	LogFile         string        // Fichier de journal système
	EventsFile      string        // Fichier de piste d'audit
	MetricsInterval time.Duration // Intervalle entre les métriques périodiques
	ReadTimeout     time.Duration // Délai de lecture des messages
	MaxErrors       int           // Nombre maximum d'erreurs consécutives
}

// NewConfig crée une configuration avec des valeurs par défaut,
// surchargées par les variables d'environnement si elles sont définies.
func NewConfig() *Config {
	cfg := &Config{
		KafkaBroker:     config.DefaultKafkaBroker,
		ConsumerGroup:   config.DefaultConsumerGroup,
		Topic:           config.DefaultTopic,
		LogFile:         config.TrackerLogFile,
		EventsFile:      config.TrackerEventsFile,
		MetricsInterval: config.TrackerMetricsInterval,
		ReadTimeout:     config.TrackerConsumerReadTimeout,
		MaxErrors:       config.TrackerMaxConsecutiveErrors,
	}

	// Surcharger depuis les variables d'environnement
	if broker := os.Getenv("KAFKA_BROKER"); broker != "" {
		cfg.KafkaBroker = broker
	}
	if group := os.Getenv("KAFKA_CONSUMER_GROUP"); group != "" {
		cfg.ConsumerGroup = group
	}
	if topic := os.Getenv("KAFKA_TOPIC"); topic != "" {
		cfg.Topic = topic
	}

	return cfg
}

// SystemMetrics collecte les métriques de performance du consommateur.
// L'accès à cette structure est protégé par un mutex pour la sécurité des threads.
type SystemMetrics struct {
	mu                sync.RWMutex
	StartTime         time.Time
	MessagesReceived  int64
	MessagesProcessed int64
	MessagesFailed    int64
	LastMessageTime   time.Time
}

// recordMetrics met à jour les compteurs de performance.
func (sm *SystemMetrics) recordMetrics(processed, failed bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.MessagesReceived++
	if processed {
		sm.MessagesProcessed++
	}
	if failed {
		sm.MessagesFailed++
	}
	sm.LastMessageTime = time.Now()
}

// Tracker est le service principal qui gère la consommation de messages Kafka.
// Il encapsule les loggers, les métriques et la configuration pour l'injection de dépendances
// et une meilleure testabilité.
type Tracker struct {
	config      *Config
	logLogger   *Logger
	eventLogger *Logger
	metrics     *SystemMetrics
	consumer    KafkaConsumer   // Interface pour la testabilité
	rawConsumer *kafka.Consumer // Garder une référence pour la fermeture
	stopChan    chan struct{}
	running     bool
	mu          sync.Mutex
}

// New crée une nouvelle instance du service Tracker.
func New(cfg *Config) *Tracker {
	return &Tracker{
		config:   cfg,
		metrics:  &SystemMetrics{StartTime: time.Now()},
		stopChan: make(chan struct{}),
	}
}

// Initialize initialise les loggers et le consommateur Kafka.
func (t *Tracker) Initialize() error {
	var err error

	// Initialiser les loggers
	t.logLogger, err = NewLogger(t.config.LogFile)
	if err != nil {
		return fmt.Errorf("impossible d'initialiser le logger système: %w", err)
	}

	t.eventLogger, err = NewLogger(t.config.EventsFile)
	if err != nil {
		t.logLogger.Close()
		return fmt.Errorf("impossible d'initialiser le logger d'événements: %w", err)
	}

	t.logLogger.Log(models.LogLevelINFO, "Système de journalisation initialisé", map[string]interface{}{
		"log_file":    t.config.LogFile,
		"events_file": t.config.EventsFile,
	})

	// Initialiser le consommateur Kafka
	t.rawConsumer, err = kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": t.config.KafkaBroker,
		"group.id":          t.config.ConsumerGroup,
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		t.logLogger.LogError("Erreur lors de la création du consommateur", err, nil)
		t.Close()
		return fmt.Errorf("impossible de créer le consommateur Kafka: %w", err)
	}
	t.consumer = newKafkaConsumerWrapper(t.rawConsumer)

	// S'abonner au sujet
	err = t.consumer.SubscribeTopics([]string{t.config.Topic}, nil)
	if err != nil {
		t.logLogger.LogError("Erreur lors de l'abonnement au sujet", err, map[string]interface{}{"topic": t.config.Topic})
		t.Close()
		return fmt.Errorf("impossible de s'abonner au sujet: %w", err)
	}

	t.logLogger.Log(models.LogLevelINFO, "Consommateur démarré et abonné au sujet '"+t.config.Topic+"'", nil)
	return nil
}

// Run démarre la boucle de consommation des messages.
func (t *Tracker) Run() {
	t.mu.Lock()
	t.running = true
	t.mu.Unlock()

	// Démarrer les métriques périodiques
	go t.logPeriodicMetrics()

	consecutiveErrors := 0

	for t.isRunning() {
		msg, err := t.consumer.ReadMessage(t.config.ReadTimeout)
		if err != nil {
			shouldStop := t.handleKafkaError(err, &consecutiveErrors)
			if shouldStop {
				break
			}
			continue
		}

		consecutiveErrors = 0
		t.processMessage(msg)
	}
}

// isRunning retourne vrai si le tracker est en cours d'exécution.
func (t *Tracker) isRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

// handleKafkaError gère les erreurs de lecture Kafka.
// Retourne vrai si le tracker doit s'arrêter.
func (t *Tracker) handleKafkaError(err error, consecutiveErrors *int) bool {
	kafkaErr, ok := err.(kafka.Error)
	if !ok {
		return false
	}

	// Timeout normal, pas une erreur
	if kafkaErr.Code() == kafka.ErrTimedOut {
		*consecutiveErrors = 0
		return false
	}

	// Vérifier si c'est une erreur de connexion critique
	errorMsg := err.Error()
	isShutdownError := strings.Contains(errorMsg, "brokers are down") ||
		strings.Contains(errorMsg, "Connection refused") ||
		kafkaErr.Code() == kafka.ErrAllBrokersDown

	if isShutdownError {
		*consecutiveErrors++
		if *consecutiveErrors >= t.config.MaxErrors {
			t.logLogger.Log(models.LogLevelINFO, "Kafka semble indisponible, arrêt du consommateur", map[string]interface{}{
				"consecutive_errors": *consecutiveErrors,
				"reason":             "brokers_unavailable",
			})
			return true
		}
		return false
	}

	// Autres erreurs
	t.logLogger.LogError("Erreur de lecture du message Kafka", err, nil)
	*consecutiveErrors++
	if *consecutiveErrors >= t.config.MaxErrors {
		t.logLogger.LogError("Trop d'erreurs consécutives, arrêt du consommateur", err, map[string]interface{}{
			"consecutive_errors": *consecutiveErrors,
		})
		return true
	}

	return false
}

// processMessage traite un message Kafka individuel.
func (t *Tracker) processMessage(msg *kafka.Message) {
	var order models.Order
	deserializationErr := json.Unmarshal(msg.Value, &order)

	// Log de l'événement (toujours)
	var orderForLog *models.Order
	if deserializationErr == nil {
		orderForLog = &order
	}
	t.eventLogger.LogEvent(msg, orderForLog, deserializationErr)

	// Mettre à jour les métriques et traiter le message
	if deserializationErr != nil {
		t.metrics.recordMetrics(false, true)
		t.logLogger.LogError("Erreur de désérialisation du message", deserializationErr, map[string]interface{}{
			"kafka_offset": msg.TopicPartition.Offset,
			"raw_message":  string(msg.Value),
		})
	} else {
		t.metrics.recordMetrics(true, false)
		displayOrder(&order)
	}
}

// logPeriodicMetrics écrit les métriques périodiques.
func (t *Tracker) logPeriodicMetrics() {
	ticker := time.NewTicker(t.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopChan:
			return
		case <-ticker.C:
			t.metrics.mu.RLock()
			uptime := time.Since(t.metrics.StartTime)
			var successRate float64
			if t.metrics.MessagesReceived > 0 {
				successRate = float64(t.metrics.MessagesProcessed) / float64(t.metrics.MessagesReceived) * 100
			}
			var messagesPerSecond float64
			if uptime.Seconds() > 0 {
				messagesPerSecond = float64(t.metrics.MessagesReceived) / uptime.Seconds()
			}
			t.metrics.mu.RUnlock()

			t.logLogger.Log(models.LogLevelINFO, "Métriques système périodiques", map[string]interface{}{
				"uptime_seconds":       uptime.Seconds(),
				"messages_received":    t.metrics.MessagesReceived,
				"messages_processed":   t.metrics.MessagesProcessed,
				"messages_failed":      t.metrics.MessagesFailed,
				"success_rate_percent": fmt.Sprintf("%.2f", successRate),
				"messages_per_second":  fmt.Sprintf("%.2f", messagesPerSecond),
			})
		}
	}
}

// Stop arrête proprement le tracker.
func (t *Tracker) Stop() {
	t.mu.Lock()
	t.running = false
	t.mu.Unlock()

	close(t.stopChan)

	// Log final
	uptime := time.Since(t.metrics.StartTime)
	t.logLogger.Log(models.LogLevelINFO, "Consommateur arrêté proprement", map[string]interface{}{
		"uptime_seconds":           uptime.Seconds(),
		"total_messages_received":  t.metrics.MessagesReceived,
		"total_messages_processed": t.metrics.MessagesProcessed,
		"total_messages_failed":    t.metrics.MessagesFailed,
	})
}

// Close libère toutes les ressources.
func (t *Tracker) Close() {
	if t.rawConsumer != nil {
		t.rawConsumer.Close()
	}
	if t.logLogger != nil {
		t.logLogger.Close()
	}
	if t.eventLogger != nil {
		t.eventLogger.Close()
	}
}

// displayOrder affiche les détails formatés de la commande dans la console.
func displayOrder(order *models.Order) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("📦 COMMANDE REÇUE #%d (ID: %s)\n", order.Sequence, order.OrderID)
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Client: %s (%s)\n", order.CustomerInfo.Name, order.CustomerInfo.CustomerID)
	fmt.Printf("Statut: %s | Total: %.2f %s\n", order.Status, order.Total, order.Currency)
	fmt.Println("Articles:")
	for _, item := range order.Items {
		fmt.Printf("  - %s (x%d) @ %.2f %s\n", item.ItemName, item.Quantity, item.UnitPrice, order.Currency)
	}
	fmt.Println(strings.Repeat("=", 80))
}
