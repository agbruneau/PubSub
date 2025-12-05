/*
Package models définit les structures de données partagées pour le système PubSub.

Ce paquet contient les types Order et associés utilisés pour le transfert d'état porté par événement (ECST).
Chaque message de commande contient toutes les informations nécessaires au traitement, éliminant le besoin
pour les consommateurs d'interroger des services externes.
*/
package models

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
)

// Erreurs de validation
var (
	ErrEmptyOrderID        = errors.New("order_id est requis")
	ErrInvalidSequence     = errors.New("la séquence doit être positive")
	ErrEmptyStatus         = errors.New("le statut est requis")
	ErrNoItems             = errors.New("la commande doit contenir au moins un article")
	ErrInvalidCustomerID   = errors.New("customer_id est requis")
	ErrInvalidCustomerName = errors.New("le nom du client est requis")
	ErrInvalidEmail        = errors.New("format d'email invalide")
	ErrInvalidItemID       = errors.New("item_id est requis")
	ErrInvalidItemName     = errors.New("item_name est requis")
	ErrInvalidQuantity     = errors.New("la quantité doit être positive")
	ErrInvalidUnitPrice    = errors.New("le prix unitaire doit être positif")
	ErrInvalidTotalPrice   = errors.New("total_price ne correspond pas à quantité * prix_unitaire")
	ErrInvalidSubtotal     = errors.New("le sous-total ne correspond pas à la somme des articles")
	ErrInvalidTax          = errors.New("la taxe doit être nulle ou positive")
	ErrInvalidTotal        = errors.New("le total est incohérent avec sous-total + taxe + frais de port")
)

// emailRegex valide le format de l'email
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// CustomerInfo contient les informations détaillées sur le client.
// Ces données sont intégrées dans chaque message de commande.
type CustomerInfo struct {
	CustomerID   string `json:"customer_id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	LoyaltyLevel string `json:"loyalty_level"`
}

// Validate vérifie que les informations du client sont valides.
func (c *CustomerInfo) Validate() error {
	if strings.TrimSpace(c.CustomerID) == "" {
		return ErrInvalidCustomerID
	}
	if strings.TrimSpace(c.Name) == "" {
		return ErrInvalidCustomerName
	}
	if c.Email != "" && !emailRegex.MatchString(c.Email) {
		return ErrInvalidEmail
	}
	return nil
}

// InventoryStatus représente l'état de l'inventaire pour un article spécifique au moment de la commande.
// Cette information permet de comprendre le contexte du stock sans interroger un service d'inventaire.
type InventoryStatus struct {
	ItemID       string  `json:"item_id"`
	ItemName     string  `json:"item_name"`
	AvailableQty int     `json:"available_qty"`
	ReservedQty  int     `json:"reserved_qty"`
	UnitPrice    float64 `json:"unit_price"`
	InStock      bool    `json:"in_stock"`
	Warehouse    string  `json:"warehouse"`
}

// OrderItem représente un article individuel au sein d'une commande.
type OrderItem struct {
	ItemID     string  `json:"item_id"`
	ItemName   string  `json:"item_name"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	TotalPrice float64 `json:"total_price"`
}

// Validate vérifie qu'un article de commande est valide.
func (item *OrderItem) Validate() error {
	if strings.TrimSpace(item.ItemID) == "" {
		return ErrInvalidItemID
	}
	if strings.TrimSpace(item.ItemName) == "" {
		return ErrInvalidItemName
	}
	if item.Quantity <= 0 {
		return ErrInvalidQuantity
	}
	if item.UnitPrice <= 0 {
		return ErrInvalidUnitPrice
	}
	expectedTotal := float64(item.Quantity) * item.UnitPrice
	if math.Abs(item.TotalPrice-expectedTotal) > 0.01 {
		return fmt.Errorf("%w: attendu %.2f, reçu %.2f", ErrInvalidTotalPrice, expectedTotal, item.TotalPrice)
	}
	return nil
}

// OrderMetadata contient les métadonnées techniques et contextuelles pour l'événement de commande.
// Ces informations sont essentielles pour le suivi, le débogage et l'analyse du flux de messages.
type OrderMetadata struct {
	Timestamp     string `json:"timestamp"`
	Version       string `json:"version"`
	EventType     string `json:"event_type"`
	Source        string `json:"source"`
	CorrelationID string `json:"correlation_id"`
}

// Order est la structure principale représentant une commande client complète.
// Elle agrège toutes les informations nécessaires au traitement dans un seul message,
// suivant le principe de transfert d'état porté par événement (ECST).
type Order struct {
	// Identifiants
	OrderID  string `json:"order_id"`
	Sequence int    `json:"sequence"`
	Status   string `json:"status"`

	// Informations client (dénormalisées pour ECST)
	CustomerInfo CustomerInfo `json:"customer_info"`

	// Articles de la commande
	Items []OrderItem `json:"items"`

	// Instantané de l'inventaire au moment de la commande
	Inventory InventoryStatus `json:"inventory"`

	// Détails financiers
	SubTotal    float64 `json:"subtotal"`
	Tax         float64 `json:"tax"`
	ShippingFee float64 `json:"shipping_fee"`
	Total       float64 `json:"total"`
	Currency    string  `json:"currency"`

	// Paiement et livraison
	PaymentMethod string `json:"payment_method"`
	DeliveryNotes string `json:"delivery_notes,omitempty"`

	// Métadonnées de l'événement
	Metadata OrderMetadata `json:"metadata"`
}

// Validate vérifie qu'une commande est valide.
// Elle valide tous les champs requis et la cohérence des montants.
func (o *Order) Validate() error {
	// Validations de base
	if strings.TrimSpace(o.OrderID) == "" {
		return ErrEmptyOrderID
	}
	if o.Sequence <= 0 {
		return ErrInvalidSequence
	}
	if strings.TrimSpace(o.Status) == "" {
		return ErrEmptyStatus
	}

	// Validation du client
	if err := o.CustomerInfo.Validate(); err != nil {
		return err
	}

	// Validation des articles
	if len(o.Items) == 0 {
		return ErrNoItems
	}

	var calculatedSubtotal float64
	for i, item := range o.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("article %d: %w", i+1, err)
		}
		calculatedSubtotal += item.TotalPrice
	}

	// Validations financières
	if math.Abs(o.SubTotal-calculatedSubtotal) > 0.01 {
		return fmt.Errorf("%w: attendu %.2f, reçu %.2f", ErrInvalidSubtotal, calculatedSubtotal, o.SubTotal)
	}

	if o.Tax < 0 {
		return ErrInvalidTax
	}

	expectedTotal := o.SubTotal + o.Tax + o.ShippingFee
	if math.Abs(o.Total-expectedTotal) > 0.01 {
		return fmt.Errorf("%w: attendu %.2f, reçu %.2f", ErrInvalidTotal, expectedTotal, o.Total)
	}

	return nil
}

// IsValid retourne vrai si la commande est valide.
func (o *Order) IsValid() bool {
	return o.Validate() == nil
}
