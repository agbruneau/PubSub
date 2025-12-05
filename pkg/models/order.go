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
	CustomerID   string `json:"customer_id"`   // Identifiant unique du client.
	Name         string `json:"name"`          // Nom complet du client.
	Email        string `json:"email"`         // Adresse email du client.
	Phone        string `json:"phone"`         // Numéro de téléphone du client.
	Address      string `json:"address"`       // Adresse physique du client.
	LoyaltyLevel string `json:"loyalty_level"` // Niveau de fidélité (ex: "silver", "gold").
}

// Validate vérifie que les informations du client sont valides.
//
// Retourne:
//   - error: Une erreur si un champ requis est manquant ou invalide.
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
	ItemID       string  `json:"item_id"`       // Identifiant de l'article en stock.
	ItemName     string  `json:"item_name"`     // Nom de l'article.
	AvailableQty int     `json:"available_qty"` // Quantité disponible avant la commande.
	ReservedQty  int     `json:"reserved_qty"`  // Quantité réservée par cette commande.
	UnitPrice    float64 `json:"unit_price"`    // Prix unitaire.
	InStock      bool    `json:"in_stock"`      // Indicateur de disponibilité (vrai si stock > 0).
	Warehouse    string  `json:"warehouse"`     // Entrepôt d'origine.
}

// OrderItem représente un article individuel au sein d'une commande.
type OrderItem struct {
	ItemID     string  `json:"item_id"`     // Identifiant unique de l'article.
	ItemName   string  `json:"item_name"`   // Nom de l'article.
	Quantity   int     `json:"quantity"`    // Quantité commandée.
	UnitPrice  float64 `json:"unit_price"`  // Prix unitaire.
	TotalPrice float64 `json:"total_price"` // Prix total pour cet article (Quantité * Prix).
}

// Validate vérifie qu'un article de commande est valide.
//
// Retourne:
//   - error: Une erreur si les données de l'article sont invalides ou incohérentes.
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
	Timestamp     string `json:"timestamp"`      // Horodatage de la création de l'événement (RFC3339).
	Version       string `json:"version"`        // Version du schéma de données.
	EventType     string `json:"event_type"`     // Type d'événement (ex: "order.created").
	Source        string `json:"source"`         // Source de l'événement (ex: "producer-service").
	CorrelationID string `json:"correlation_id"` // Identifiant de corrélation pour le traçage distribué.
}

// Order est la structure principale représentant une commande client complète.
// Elle agrège toutes les informations nécessaires au traitement dans un seul message,
// suivant le principe de transfert d'état porté par événement (ECST).
type Order struct {
	// Identifiants
	OrderID  string `json:"order_id"` // Identifiant unique de la commande (UUID).
	Sequence int    `json:"sequence"` // Numéro de séquence incrémental.
	Status   string `json:"status"`   // État de la commande (ex: "pending").

	// Informations client (dénormalisées pour ECST)
	CustomerInfo CustomerInfo `json:"customer_info"`

	// Articles de la commande
	Items []OrderItem `json:"items"`

	// Instantané de l'inventaire au moment de la commande
	Inventory InventoryStatus `json:"inventory"`

	// Détails financiers
	SubTotal    float64 `json:"subtotal"`     // Somme des articles.
	Tax         float64 `json:"tax"`          // Montant de la taxe.
	ShippingFee float64 `json:"shipping_fee"` // Frais de livraison.
	Total       float64 `json:"total"`        // Montant total.
	Currency    string  `json:"currency"`     // Devise (ex: "EUR").

	// Paiement et livraison
	PaymentMethod string `json:"payment_method"`       // Méthode de paiement utilisée.
	DeliveryNotes string `json:"delivery_notes,omitempty"` // Notes de livraison optionnelles.

	// Métadonnées de l'événement
	Metadata OrderMetadata `json:"metadata"`
}

// Validate vérifie qu'une commande est valide.
// Elle valide tous les champs requis et la cohérence des montants.
//
// Retourne:
//   - error: Une erreur si la commande est invalide.
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
//
// Retourne:
//   - bool: Vrai si la validation réussit, faux sinon.
func (o *Order) IsValid() bool {
	return o.Validate() == nil
}
