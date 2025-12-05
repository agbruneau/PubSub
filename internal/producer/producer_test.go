package producer

import (
	"testing"
)

// TestGenerateOrder vérifie que GenerateOrder crée une commande valide.
func TestGenerateOrder(t *testing.T) {
	cfg := NewConfig()
	producer := New(cfg)

	template := OrderTemplate{
		User:     "test-user",
		Item:     "test-item",
		Quantity: 3,
		Price:    10.00,
	}

	order := producer.GenerateOrder(template, 1)

	// Vérifier les champs de base
	if order.OrderID == "" {
		t.Error("Attendu que OrderID soit défini")
	}
	if order.Sequence != 1 {
		t.Errorf("Attendu que la séquence soit 1, reçu %d", order.Sequence)
	}
	if order.Status != "pending" {
		t.Errorf("Attendu que le statut soit 'pending', reçu %s", order.Status)
	}

	// Vérifier les articles
	if len(order.Items) != 1 {
		t.Errorf("Attendu 1 article, reçu %d", len(order.Items))
	}
	if order.Items[0].ItemName != "test-item" {
		t.Errorf("Attendu que ItemName soit 'test-item', reçu %s", order.Items[0].ItemName)
	}
	if order.Items[0].Quantity != 3 {
		t.Errorf("Attendu que la quantité soit 3, reçu %d", order.Items[0].Quantity)
	}

	// Vérifier les calculs financiers
	expectedSubTotal := 30.00 // 3 * 10.00
	if order.SubTotal != expectedSubTotal {
		t.Errorf("Attendu que SubTotal soit %.2f, reçu %.2f", expectedSubTotal, order.SubTotal)
	}

	expectedTax := expectedSubTotal * cfg.TaxRate
	if order.Tax != expectedTax {
		t.Errorf("Attendu que Tax soit %.2f, reçu %.2f", expectedTax, order.Tax)
	}

	expectedTotal := expectedSubTotal + expectedTax + cfg.ShippingFee
	if order.Total != expectedTotal {
		t.Errorf("Attendu que Total soit %.2f, reçu %.2f", expectedTotal, order.Total)
	}

	// Vérifier les infos client
	if order.CustomerInfo.CustomerID != "test-user" {
		t.Errorf("Attendu que CustomerID soit 'test-user', reçu %s", order.CustomerInfo.CustomerID)
	}

	// Vérifier les métadonnées
	if order.Metadata.EventType != "order.created" {
		t.Errorf("Attendu que EventType soit 'order.created', reçu %s", order.Metadata.EventType)
	}
	if order.Metadata.Source != "producer-service" {
		t.Errorf("Attendu que Source soit 'producer-service', reçu %s", order.Metadata.Source)
	}
}

// TestNewConfig vérifie que la configuration par défaut est correctement créée.
func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg.KafkaBroker == "" {
		t.Error("Attendu que KafkaBroker soit défini")
	}
	if cfg.Topic == "" {
		t.Error("Attendu que Topic soit défini")
	}
	if cfg.TaxRate <= 0 {
		t.Error("Attendu que TaxRate soit positif")
	}
	if cfg.ShippingFee < 0 {
		t.Error("Attendu que ShippingFee soit non-négatif")
	}
	if cfg.Currency == "" {
		t.Error("Attendu que Currency soit défini")
	}
}

// TestDefaultOrderTemplates vérifie que les modèles par défaut sont définis.
func TestDefaultOrderTemplates(t *testing.T) {
	if len(DefaultOrderTemplates) == 0 {
		t.Error("Attendu que DefaultOrderTemplates ait au moins un modèle")
	}

	// Vérifier que tous les modèles ont les champs requis
	for i, template := range DefaultOrderTemplates {
		if template.User == "" {
			t.Errorf("Modèle %d: Attendu que User soit défini", i)
		}
		if template.Item == "" {
			t.Errorf("Modèle %d: Attendu que Item soit défini", i)
		}
		if template.Quantity <= 0 {
			t.Errorf("Modèle %d: Attendu que Quantity soit positif, reçu %d", i, template.Quantity)
		}
		if template.Price <= 0 {
			t.Errorf("Modèle %d: Attendu que Price soit positif, reçu %f", i, template.Price)
		}
	}
}

// TestNew vérifie qu'un nouveau OrderProducer est correctement créé.
func TestNew(t *testing.T) {
	cfg := NewConfig()
	producer := New(cfg)

	if producer.config != cfg {
		t.Error("Attendu que config soit défini")
	}
	if producer.sequence != 1 {
		t.Errorf("Attendu que la séquence commence à 1, reçu %d", producer.sequence)
	}
	if len(producer.templates) == 0 {
		t.Error("Attendu que templates soit défini")
	}
}
