//go:build kafka
// +build kafka

/*
Package producer provides the Kafka message producer for the PubSub system.

This package implements the order generation and Kafka publishing logic,
following the Event Carried State Transfer (ECST) pattern.
*/
package producer

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/agbruneau/PubSub/internal/config"
	"github.com/agbruneau/PubSub/pkg/models"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
)

// Config contains the producer service configuration.
// It can be loaded from environment variables.
type Config struct {
	KafkaBroker     string        // Kafka broker address
	Topic           string        // Kafka topic for publishing
	MessageInterval time.Duration // Interval between messages
	FlushTimeout    int           // Timeout in ms for final flush
	TaxRate         float64       // Tax rate to apply
	ShippingFee     float64       // Shipping fee
	Currency        string        // Default currency
	PaymentMethod   string        // Default payment method
	Warehouse       string        // Default warehouse
}

// NewConfig creates a configuration with default values,
// overridden by environment variables if defined.
func NewConfig() *Config {
	cfg := &Config{
		KafkaBroker:     config.DefaultKafkaBroker,
		Topic:           config.DefaultTopic,
		MessageInterval: config.ProducerMessageInterval,
		FlushTimeout:    config.FlushTimeoutMs,
		TaxRate:         config.ProducerDefaultTaxRate,
		ShippingFee:     config.ProducerDefaultShippingFee,
		Currency:        config.ProducerDefaultCurrency,
		PaymentMethod:   config.ProducerDefaultPayment,
		Warehouse:       config.ProducerDefaultWarehouse,
	}

	// Override from environment variables
	if broker := os.Getenv("KAFKA_BROKER"); broker != "" {
		cfg.KafkaBroker = broker
	}
	if topic := os.Getenv("KAFKA_TOPIC"); topic != "" {
		cfg.Topic = topic
	}

	return cfg
}

// OrderTemplate defines a template for generating test orders.
type OrderTemplate struct {
	User     string  // Customer identifier
	Item     string  // Item name
	Quantity int     // Ordered quantity
	Price    float64 // Unit price
}

// DefaultOrderTemplates contains the default order templates.
var DefaultOrderTemplates = []OrderTemplate{
	{User: "client01", Item: "espresso", Quantity: 2, Price: 2.50},
	{User: "client02", Item: "cappuccino", Quantity: 3, Price: 3.20},
	{User: "client03", Item: "latte", Quantity: 4, Price: 3.50},
	{User: "client04", Item: "macchiato", Quantity: 5, Price: 3.00},
	{User: "client05", Item: "flat white", Quantity: 6, Price: 3.30},
	{User: "client06", Item: "mocha", Quantity: 7, Price: 4.00},
	{User: "client07", Item: "americano", Quantity: 8, Price: 2.80},
	{User: "client08", Item: "chai latte", Quantity: 9, Price: 3.80},
	{User: "client09", Item: "matcha", Quantity: 10, Price: 4.50},
	{User: "client10", Item: "smoothie fraise", Quantity: 11, Price: 5.50},
}

// OrderProducer is the main service that manages Kafka message production.
// It encapsulates the Kafka producer, configuration, and order templates.
type OrderProducer struct {
	config       *Config
	producer     *kafka.Producer
	deliveryChan chan kafka.Event
	templates    []OrderTemplate
	sequence     int
	running      bool
}

// New creates a new instance of the OrderProducer service.
func New(cfg *Config) *OrderProducer {
	return &OrderProducer{
		config:    cfg,
		templates: DefaultOrderTemplates,
		sequence:  1,
	}
}

// Initialize initializes the Kafka producer.
func (p *OrderProducer) Initialize() error {
	var err error
	p.producer, err = kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": p.config.KafkaBroker,
	})
	if err != nil {
		return fmt.Errorf("unable to create Kafka producer: %w", err)
	}

	p.deliveryChan = make(chan kafka.Event, config.ProducerDeliveryChannelSize)
	go p.handleDeliveryReports()

	return nil
}

// handleDeliveryReports processes delivery reports in a dedicated goroutine.
func (p *OrderProducer) handleDeliveryReports() {
	for e := range p.deliveryChan {
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			fmt.Printf("❌ Message delivery failed: %v\n", m.TopicPartition.Error)
		} else {
			fmt.Printf("✅ Message delivered to topic %s (partition %d) at offset %d\n",
				*m.TopicPartition.Topic,
				m.TopicPartition.Partition,
				m.TopicPartition.Offset)
		}
	}
}

// GenerateOrder creates an enriched order from a template and sequence number.
func (p *OrderProducer) GenerateOrder(template OrderTemplate, sequence int) models.Order {
	// Financial calculations
	itemTotal := float64(template.Quantity) * template.Price
	tax := itemTotal * p.config.TaxRate
	total := itemTotal + tax + p.config.ShippingFee

	return models.Order{
		OrderID:  uuid.New().String(),
		Sequence: sequence,
		Status:   "pending",
		Items: []models.OrderItem{
			{
				ItemID:     fmt.Sprintf("item-%s", template.Item),
				ItemName:   template.Item,
				Quantity:   template.Quantity,
				UnitPrice:  template.Price,
				TotalPrice: itemTotal,
			},
		},
		SubTotal:      itemTotal,
		Tax:           tax,
		ShippingFee:   p.config.ShippingFee,
		Total:         total,
		Currency:      p.config.Currency,
		PaymentMethod: p.config.PaymentMethod,
		DeliveryNotes: fmt.Sprintf("Deliver to %d Rue de la Paix, 75000 Paris", sequence),
		Metadata: models.OrderMetadata{
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
			Version:       "1.1",
			EventType:     "order.created",
			Source:        "producer-service",
			CorrelationID: uuid.New().String(),
		},
		CustomerInfo: models.CustomerInfo{
			CustomerID:   template.User,
			Name:         fmt.Sprintf("Client %s", template.User),
			Email:        fmt.Sprintf("%s@example.com", template.User),
			Phone:        "+33 6 00 00 00 00",
			Address:      fmt.Sprintf("%d Rue de la Paix, 75000 Paris", sequence),
			LoyaltyLevel: "silver",
		},
		Inventory: models.InventoryStatus{
			ItemID:       fmt.Sprintf("item-%s", template.Item),
			ItemName:     template.Item,
			AvailableQty: 100 - template.Quantity,
			ReservedQty:  template.Quantity,
			UnitPrice:    template.Price,
			InStock:      true,
			Warehouse:    p.config.Warehouse,
		},
	}
}

// ProduceOrder generates and sends an order to the Kafka topic.
func (p *OrderProducer) ProduceOrder() error {
	template := p.templates[p.sequence%len(p.templates)]
	order := p.GenerateOrder(template, p.sequence)

	value, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("JSON serialization error: %w", err)
	}

	topic := p.config.Topic
	err = p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          value,
	}, p.deliveryChan)

	if err != nil {
		return fmt.Errorf("error producing message: %w", err)
	}

	p.sequence++
	return nil
}

// Run starts the message production loop.
func (p *OrderProducer) Run(stopChan <-chan os.Signal) {
	p.running = true
	for p.running {
		select {
		case <-stopChan:
			fmt.Println("\n⚠️  Stop signal received. Stopping new message production...")
			p.running = false
		default:
			if err := p.ProduceOrder(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			time.Sleep(p.config.MessageInterval)
		}
	}
}

// Close properly closes the producer and sends pending messages.
func (p *OrderProducer) Close() {
	fmt.Println("⏳ Sending remaining messages in queue...")
	remainingMessages := p.producer.Flush(p.config.FlushTimeout)
	if remainingMessages > 0 {
		fmt.Printf("⚠️  %d messages could not be sent.\n", remainingMessages)
	} else {
		fmt.Println("✅ All messages sent successfully.")
	}
	p.producer.Close()
}
