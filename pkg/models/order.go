/*
Package models defines the shared data structures for the PubSub system.

This package contains the Order and related types used for Event Carried State Transfer (ECST).
Each order message contains all information needed for processing, eliminating the need
for consumers to query external services.
*/
package models

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
)

// Validation errors
var (
	ErrEmptyOrderID       = errors.New("order_id est requis")
	ErrInvalidSequence    = errors.New("sequence doit être positif")
	ErrEmptyStatus        = errors.New("status est requis")
	ErrNoItems            = errors.New("la commande doit contenir au moins un article")
	ErrInvalidCustomerID  = errors.New("customer_id est requis")
	ErrInvalidCustomerName = errors.New("le nom du client est requis")
	ErrInvalidEmail       = errors.New("email invalide")
	ErrInvalidItemID      = errors.New("item_id est requis")
	ErrInvalidItemName    = errors.New("item_name est requis")
	ErrInvalidQuantity    = errors.New("quantity doit être positif")
	ErrInvalidUnitPrice   = errors.New("unit_price doit être positif")
	ErrInvalidTotalPrice  = errors.New("total_price ne correspond pas à quantity * unit_price")
	ErrInvalidSubtotal    = errors.New("subtotal ne correspond pas à la somme des articles")
	ErrInvalidTax         = errors.New("tax doit être positif ou nul")
	ErrInvalidTotal       = errors.New("total est incohérent avec subtotal + tax + shipping")
)

// emailRegex validates email format
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// CustomerInfo contains detailed customer information.
// This data is embedded in each order message.
type CustomerInfo struct {
	CustomerID   string `json:"customer_id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	LoyaltyLevel string `json:"loyalty_level"`
}

// Validate checks that the customer information is valid.
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

// InventoryStatus represents inventory state for a specific item at order time.
// This information allows understanding stock context without querying an inventory service.
type InventoryStatus struct {
	ItemID       string  `json:"item_id"`
	ItemName     string  `json:"item_name"`
	AvailableQty int     `json:"available_qty"`
	ReservedQty  int     `json:"reserved_qty"`
	UnitPrice    float64 `json:"unit_price"`
	InStock      bool    `json:"in_stock"`
	Warehouse    string  `json:"warehouse"`
}

// OrderItem represents an individual item within an order.
type OrderItem struct {
	ItemID     string  `json:"item_id"`
	ItemName   string  `json:"item_name"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	TotalPrice float64 `json:"total_price"`
}

// Validate checks that an order item is valid.
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
		return fmt.Errorf("%w: attendu %.2f, obtenu %.2f", ErrInvalidTotalPrice, expectedTotal, item.TotalPrice)
	}
	return nil
}

// OrderMetadata contains technical and contextual metadata for the order event.
// This information is essential for tracking, debugging, and message flow analysis.
type OrderMetadata struct {
	Timestamp     string `json:"timestamp"`
	Version       string `json:"version"`
	EventType     string `json:"event_type"`
	Source        string `json:"source"`
	CorrelationID string `json:"correlation_id"`
}

// Order is the main structure representing a complete customer order.
// It aggregates all information needed for processing in a single message,
// following the Event Carried State Transfer principle.
type Order struct {
	// Identifiers
	OrderID  string `json:"order_id"`
	Sequence int    `json:"sequence"`
	Status   string `json:"status"`

	// Customer information (denormalized for ECST)
	CustomerInfo CustomerInfo `json:"customer_info"`

	// Order items
	Items []OrderItem `json:"items"`

	// Inventory snapshot at order time
	Inventory InventoryStatus `json:"inventory"`

	// Financial details
	SubTotal    float64 `json:"subtotal"`
	Tax         float64 `json:"tax"`
	ShippingFee float64 `json:"shipping_fee"`
	Total       float64 `json:"total"`
	Currency    string  `json:"currency"`

	// Payment and delivery
	PaymentMethod string `json:"payment_method"`
	DeliveryNotes string `json:"delivery_notes,omitempty"`

	// Event metadata
	Metadata OrderMetadata `json:"metadata"`
}

// Validate checks that an order is valid.
// It validates all required fields and amount consistency.
func (o *Order) Validate() error {
	// Basic validations
	if strings.TrimSpace(o.OrderID) == "" {
		return ErrEmptyOrderID
	}
	if o.Sequence <= 0 {
		return ErrInvalidSequence
	}
	if strings.TrimSpace(o.Status) == "" {
		return ErrEmptyStatus
	}

	// Customer validation
	if err := o.CustomerInfo.Validate(); err != nil {
		return err
	}

	// Items validation
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

	// Financial validations
	if math.Abs(o.SubTotal-calculatedSubtotal) > 0.01 {
		return fmt.Errorf("%w: attendu %.2f, obtenu %.2f", ErrInvalidSubtotal, calculatedSubtotal, o.SubTotal)
	}

	if o.Tax < 0 {
		return ErrInvalidTax
	}

	expectedTotal := o.SubTotal + o.Tax + o.ShippingFee
	if math.Abs(o.Total-expectedTotal) > 0.01 {
		return fmt.Errorf("%w: attendu %.2f, obtenu %.2f", ErrInvalidTotal, expectedTotal, o.Total)
	}

	return nil
}

// IsValid returns true if the order is valid.
func (o *Order) IsValid() bool {
	return o.Validate() == nil
}
