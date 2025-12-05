//go:build kafka
// +build kafka

/*
Package tracker provides the Kafka message consumer for the PubSub system.

This package implements message consumption, logging, and metrics collection,
following the Audit Trail and Application Health Monitoring patterns.
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

// Config contains the tracker service configuration.
// It can be loaded from environment variables.
type Config struct {
	KafkaBroker     string        // Kafka broker address
	ConsumerGroup   string        // Kafka consumer group
	Topic           string        // Kafka topic to consume
	LogFile         string        // System log file
	EventsFile      string        // Audit trail file
	MetricsInterval time.Duration // Interval between periodic metrics
	ReadTimeout     time.Duration // Message read timeout
	MaxErrors       int           // Maximum consecutive errors
}

// NewConfig creates a configuration with default values,
// overridden by environment variables if defined.
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

	// Override from environment variables
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

// SystemMetrics collects consumer performance metrics.
// Access to this structure is protected by a mutex for thread safety.
type SystemMetrics struct {
	mu                sync.RWMutex
	StartTime         time.Time
	MessagesReceived  int64
	MessagesProcessed int64
	MessagesFailed    int64
	LastMessageTime   time.Time
}

// recordMetrics updates performance counters.
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

// Tracker is the main service that manages Kafka message consumption.
// It encapsulates loggers, metrics, and configuration for dependency injection
// and better testability.
type Tracker struct {
	config      *Config
	logLogger   *Logger
	eventLogger *Logger
	metrics     *SystemMetrics
	consumer    *kafka.Consumer
	stopChan    chan struct{}
	running     bool
	mu          sync.Mutex
}

// New creates a new instance of the Tracker service.
func New(cfg *Config) *Tracker {
	return &Tracker{
		config:   cfg,
		metrics:  &SystemMetrics{StartTime: time.Now()},
		stopChan: make(chan struct{}),
	}
}

// Initialize initializes the loggers and Kafka consumer.
func (t *Tracker) Initialize() error {
	var err error

	// Initialize loggers
	t.logLogger, err = NewLogger(t.config.LogFile)
	if err != nil {
		return fmt.Errorf("unable to initialize system logger: %w", err)
	}

	t.eventLogger, err = NewLogger(t.config.EventsFile)
	if err != nil {
		t.logLogger.Close()
		return fmt.Errorf("unable to initialize event logger: %w", err)
	}

	t.logLogger.Log(models.LogLevelINFO, "Logging system initialized", map[string]interface{}{
		"log_file":    t.config.LogFile,
		"events_file": t.config.EventsFile,
	})

	// Initialize Kafka consumer
	t.consumer, err = kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": t.config.KafkaBroker,
		"group.id":          t.config.ConsumerGroup,
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		t.logLogger.LogError("Error creating consumer", err, nil)
		t.Close()
		return fmt.Errorf("unable to create Kafka consumer: %w", err)
	}

	// Subscribe to topic
	err = t.consumer.SubscribeTopics([]string{t.config.Topic}, nil)
	if err != nil {
		t.logLogger.LogError("Error subscribing to topic", err, map[string]interface{}{"topic": t.config.Topic})
		t.Close()
		return fmt.Errorf("unable to subscribe to topic: %w", err)
	}

	t.logLogger.Log(models.LogLevelINFO, "Consumer started and subscribed to topic '"+t.config.Topic+"'", nil)
	return nil
}

// Run starts the message consumption loop.
func (t *Tracker) Run() {
	t.mu.Lock()
	t.running = true
	t.mu.Unlock()

	// Start periodic metrics
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

// isRunning returns true if the tracker is running.
func (t *Tracker) isRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

// handleKafkaError handles Kafka read errors.
// Returns true if the tracker should stop.
func (t *Tracker) handleKafkaError(err error, consecutiveErrors *int) bool {
	kafkaErr, ok := err.(kafka.Error)
	if !ok {
		return false
	}

	// Normal timeout, not an error
	if kafkaErr.Code() == kafka.ErrTimedOut {
		*consecutiveErrors = 0
		return false
	}

	// Check if it's a critical connection error
	errorMsg := err.Error()
	isShutdownError := strings.Contains(errorMsg, "brokers are down") ||
		strings.Contains(errorMsg, "Connection refused") ||
		kafkaErr.Code() == kafka.ErrAllBrokersDown

	if isShutdownError {
		*consecutiveErrors++
		if *consecutiveErrors >= t.config.MaxErrors {
			t.logLogger.Log(models.LogLevelINFO, "Kafka appears to be down, stopping consumer", map[string]interface{}{
				"consecutive_errors": *consecutiveErrors,
				"reason":             "brokers_unavailable",
			})
			return true
		}
		return false
	}

	// Other errors
	t.logLogger.LogError("Kafka message read error", err, nil)
	*consecutiveErrors++
	if *consecutiveErrors >= t.config.MaxErrors {
		t.logLogger.LogError("Too many consecutive errors, stopping consumer", err, map[string]interface{}{
			"consecutive_errors": *consecutiveErrors,
		})
		return true
	}

	return false
}

// processMessage processes an individual Kafka message.
func (t *Tracker) processMessage(msg *kafka.Message) {
	var order models.Order
	deserializationErr := json.Unmarshal(msg.Value, &order)

	// Log the event (always)
	var orderForLog *models.Order
	if deserializationErr == nil {
		orderForLog = &order
	}
	t.eventLogger.LogEvent(msg, orderForLog, deserializationErr)

	// Update metrics and process the message
	if deserializationErr != nil {
		t.metrics.recordMetrics(false, true)
		t.logLogger.LogError("Message deserialization error", deserializationErr, map[string]interface{}{
			"kafka_offset": msg.TopicPartition.Offset,
			"raw_message":  string(msg.Value),
		})
	} else {
		t.metrics.recordMetrics(true, false)
		displayOrder(&order)
	}
}

// logPeriodicMetrics writes periodic metrics.
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

			t.logLogger.Log(models.LogLevelINFO, "Periodic system metrics", map[string]interface{}{
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

// Stop properly stops the tracker.
func (t *Tracker) Stop() {
	t.mu.Lock()
	t.running = false
	t.mu.Unlock()

	close(t.stopChan)

	// Final log
	uptime := time.Since(t.metrics.StartTime)
	t.logLogger.Log(models.LogLevelINFO, "Consumer stopped properly", map[string]interface{}{
		"uptime_seconds":           uptime.Seconds(),
		"total_messages_received":  t.metrics.MessagesReceived,
		"total_messages_processed": t.metrics.MessagesProcessed,
		"total_messages_failed":    t.metrics.MessagesFailed,
	})
}

// Close releases all resources.
func (t *Tracker) Close() {
	if t.consumer != nil {
		t.consumer.Close()
	}
	if t.logLogger != nil {
		t.logLogger.Close()
	}
	if t.eventLogger != nil {
		t.eventLogger.Close()
	}
}

// displayOrder displays formatted order details to the console.
func displayOrder(order *models.Order) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("📦 ORDER RECEIVED #%d (ID: %s)\n", order.Sequence, order.OrderID)
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Customer: %s (%s)\n", order.CustomerInfo.Name, order.CustomerInfo.CustomerID)
	fmt.Printf("Status: %s | Total: %.2f %s\n", order.Status, order.Total, order.Currency)
	fmt.Println("Items:")
	for _, item := range order.Items {
		fmt.Printf("  - %s (x%d) @ %.2f %s\n", item.ItemName, item.Quantity, item.UnitPrice, order.Currency)
	}
	fmt.Println(strings.Repeat("=", 80))
}
