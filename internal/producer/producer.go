/*
Package producer provides the Kafka message producer for the PubSub system.

This package implements the logic for generating orders and publishing to Kafka,
following the Event-Carried State Transfer (ECST) pattern.
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
	KafkaBroker     string        // Kafka broker address.
	Topic           string        // Kafka topic for publication.
	MessageInterval time.Duration // Interval between messages.
	FlushTimeout    int           // Timeout in ms for final flush.
	TaxRate         float64       // Tax rate to apply.
	ShippingFee     float64       // Shipping fee.
	Currency        string        // Default currency.
	PaymentMethod   string        // Default payment method.
	Warehouse       string        // Default warehouse.
}

// NewConfig creates a configuration with default values,
// overridden by environment variables if defined.
//
// Returns:
//   - *Config: The initialized configuration.
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
	User     string  // Customer identifier.
	Item     string  // Item name.
	Quantity int     // Ordered quantity.
	Price    float64 // Unit price.
}

// DefaultOrderTemplates contains default order templates.
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
	{User: "client10", Item: "strawberry smoothie", Quantity: 11, Price: 5.50},
}

// OrderProducer is the main service handling Kafka message production.
// It encapsulates the Kafka producer, configuration, and order templates.
type OrderProducer struct {
	config       *Config         // Producer configuration.
	producer     KafkaProducer   // Interface for testability.
	rawProducer  *kafka.Producer // Keep a reference for delivery reports.
	deliveryChan chan kafka.Event
	templates    []OrderTemplate // Order templates to use.
	sequence     int             // Internal sequencer for IDs.
	running      bool            // Running state.
}

// New creates a new instance of the OrderProducer service.
//
// Parameters:
//   - cfg: The producer configuration.
//
// Returns:
//   - *OrderProducer: The created instance.
func New(cfg *Config) *OrderProducer {
	return &OrderProducer{
		config:    cfg,
		templates: DefaultOrderTemplates,
		sequence:  1,
	}
}

// Initialize initializes the Kafka producer.
// Creates the connection to the broker and starts the report handler.
//
// Returns:
//   - error: An error if connection fails.
func (p *OrderProducer) Initialize() error {
	var err error
	p.rawProducer, err = kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": p.config.KafkaBroker,
	})
	if err != nil {
		return fmt.Errorf("failed to create Kafka producer: %w", err)
	}
	p.producer = newKafkaProducerWrapper(p.rawProducer)

	p.deliveryChan = make(chan kafka.Event, config.ProducerDeliveryChannelSize)
	go p.handleDeliveryReports()

	return nil
}

// handleDeliveryReports processes delivery reports in a dedicated goroutine.
// Logs success or failure for each produced message.
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

// GenerateOrder creates an enriched order from a template and a sequence number.
//
// Parameters:
//   - template: The order template to use.
//   - sequence: The unique sequence number.
//
// Returns:
//   - models.Order: The complete generated order.
func (p *OrderProducer) GenerateOrder(template OrderTemplate, sequence int) models.Order {
	// Financial calculations
	itemTotal := float64(template.Quantity) * template.Price
	tax := itemTotal * p.config.TaxRate
	total := itemTotal + tax + p.config.ShippingFee

	const initialStock = 100
	availableQty := initialStock - template.Quantity
	inStock := availableQty >= 0

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
			AvailableQty: availableQty,
			ReservedQty:  template.Quantity,
			UnitPrice:    template.Price,
			InStock:      inStock,
			Warehouse:    p.config.Warehouse,
		},
	}
}

// ProduceOrder generates and sends an order to the Kafka topic.
// Selects an order template in a round-robin fashion.
//
// Returns:
//   - error: An error if production fails.
func (p *OrderProducer) ProduceOrder() error {
	template := p.templates[(p.sequence-1)%len(p.templates)]
	order := p.GenerateOrder(template, p.sequence)

	value, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("JSON marshaling error: %w", err)
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
// Continues until a stop signal is received on stopChan.
//
// Parameters:
//   - stopChan: The stop signal channel.
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

// Close gracefully closes the producer and flushes pending messages.
// This method blocks until messages are flushed or timeout is reached.
func (p *OrderProducer) Close() {
	fmt.Println("⏳ Sending remaining messages in queue...")
	remainingMessages := p.producer.Flush(p.config.FlushTimeout)
	if remainingMessages > 0 {
		fmt.Printf("⚠️  %d messages could not be sent.\n", remainingMessages)
	} else {
		fmt.Println("✅ All messages sent successfully.")
	}
	if p.rawProducer != nil {
		p.rawProducer.Close()
	}
}
