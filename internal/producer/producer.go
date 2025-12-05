/*
Package producer fournit le producteur de messages Kafka pour le système PubSub.

Ce paquet implémente la logique de génération de commandes et de publication Kafka,
suivant le modèle de transfert d'état porté par événement (ECST).
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

// Config contient la configuration du service producteur.
// Elle peut être chargée à partir de variables d'environnement.
type Config struct {
	KafkaBroker     string        // Adresse du broker Kafka.
	Topic           string        // Sujet Kafka pour la publication.
	MessageInterval time.Duration // Intervalle entre les messages.
	FlushTimeout    int           // Délai en ms pour le flush final.
	TaxRate         float64       // Taux de taxe à appliquer.
	ShippingFee     float64       // Frais de port.
	Currency        string        // Devise par défaut.
	PaymentMethod   string        // Méthode de paiement par défaut.
	Warehouse       string        // Entrepôt par défaut.
}

// NewConfig crée une configuration avec des valeurs par défaut,
// surchargées par les variables d'environnement si elles sont définies.
//
// Retourne:
//   - *Config: La configuration initialisée.
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

	// Surcharger depuis les variables d'environnement
	if broker := os.Getenv("KAFKA_BROKER"); broker != "" {
		cfg.KafkaBroker = broker
	}
	if topic := os.Getenv("KAFKA_TOPIC"); topic != "" {
		cfg.Topic = topic
	}

	return cfg
}

// OrderTemplate définit un modèle pour générer des commandes de test.
type OrderTemplate struct {
	User     string  // Identifiant du client.
	Item     string  // Nom de l'article.
	Quantity int     // Quantité commandée.
	Price    float64 // Prix unitaire.
}

// DefaultOrderTemplates contient les modèles de commandes par défaut.
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

// OrderProducer est le service principal qui gère la production de messages Kafka.
// Il encapsule le producteur Kafka, la configuration et les modèles de commandes.
type OrderProducer struct {
	config       *Config         // Configuration du producteur.
	producer     KafkaProducer   // Interface pour la testabilité.
	rawProducer  *kafka.Producer // Garder une référence pour les rapports de livraison.
	deliveryChan chan kafka.Event
	templates    []OrderTemplate // Modèles de commandes à utiliser.
	sequence     int             // Séquenceur interne pour les ID.
	running      bool            // État de fonctionnement.
}

// New crée une nouvelle instance du service OrderProducer.
//
// Paramètres:
//   - cfg: La configuration du producteur.
//
// Retourne:
//   - *OrderProducer: L'instance créée.
func New(cfg *Config) *OrderProducer {
	return &OrderProducer{
		config:    cfg,
		templates: DefaultOrderTemplates,
		sequence:  1,
	}
}

// Initialize initialise le producteur Kafka.
// Crée la connexion au broker et démarre le gestionnaire de rapports.
//
// Retourne:
//   - error: Une erreur si la connexion échoue.
func (p *OrderProducer) Initialize() error {
	var err error
	p.rawProducer, err = kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": p.config.KafkaBroker,
	})
	if err != nil {
		return fmt.Errorf("impossible de créer le producteur Kafka: %w", err)
	}
	p.producer = newKafkaProducerWrapper(p.rawProducer)

	p.deliveryChan = make(chan kafka.Event, config.ProducerDeliveryChannelSize)
	go p.handleDeliveryReports()

	return nil
}

// handleDeliveryReports traite les rapports de livraison dans une goroutine dédiée.
// Affiche le succès ou l'échec de chaque message produit.
func (p *OrderProducer) handleDeliveryReports() {
	for e := range p.deliveryChan {
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			fmt.Printf("❌ Échec de la livraison du message: %v\n", m.TopicPartition.Error)
		} else {
			fmt.Printf("✅ Message livré au sujet %s (partition %d) à l'offset %d\n",
				*m.TopicPartition.Topic,
				m.TopicPartition.Partition,
				m.TopicPartition.Offset)
		}
	}
}

// GenerateOrder crée une commande enrichie à partir d'un modèle et d'un numéro de séquence.
//
// Paramètres:
//   - template: Le modèle de commande à utiliser.
//   - sequence: Le numéro de séquence unique.
//
// Retourne:
//   - models.Order: La commande complète générée.
func (p *OrderProducer) GenerateOrder(template OrderTemplate, sequence int) models.Order {
	// Calculs financiers
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
		DeliveryNotes: fmt.Sprintf("Livrer au %d Rue de la Paix, 75000 Paris", sequence),
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

// ProduceOrder génère et envoie une commande au sujet Kafka.
// Sélectionne un modèle de commande de manière cyclique.
//
// Retourne:
//   - error: Une erreur si la production échoue.
func (p *OrderProducer) ProduceOrder() error {
	template := p.templates[p.sequence%len(p.templates)]
	order := p.GenerateOrder(template, p.sequence)

	value, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("erreur de sérialisation JSON: %w", err)
	}

	topic := p.config.Topic
	err = p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          value,
	}, p.deliveryChan)

	if err != nil {
		return fmt.Errorf("erreur lors de la production du message: %w", err)
	}

	p.sequence++
	return nil
}

// Run démarre la boucle de production de messages.
// Continue jusqu'à ce qu'un signal d'arrêt soit reçu sur stopChan.
//
// Paramètres:
//   - stopChan: Le canal de signal d'arrêt.
func (p *OrderProducer) Run(stopChan <-chan os.Signal) {
	p.running = true
	for p.running {
		select {
		case <-stopChan:
			fmt.Println("\n⚠️  Signal d'arrêt reçu. Arrêt de la production de nouveaux messages...")
			p.running = false
		default:
			if err := p.ProduceOrder(); err != nil {
				fmt.Printf("Erreur: %v\n", err)
			}
			time.Sleep(p.config.MessageInterval)
		}
	}
}

// Close ferme proprement le producteur et envoie les messages en attente.
// Cette méthode bloque jusqu'à ce que les messages soient vidés ou le timeout atteint.
func (p *OrderProducer) Close() {
	fmt.Println("⏳ Envoi des messages restants dans la file d'attente...")
	remainingMessages := p.producer.Flush(p.config.FlushTimeout)
	if remainingMessages > 0 {
		fmt.Printf("⚠️  %d messages n'ont pas pu être envoyés.\n", remainingMessages)
	} else {
		fmt.Println("✅ Tous les messages ont été envoyés avec succès.")
	}
	p.rawProducer.Close()
}
