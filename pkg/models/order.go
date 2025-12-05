/*
Package models defines shared data structures for the PubSub system.

This package contains Order and associated types used for Event-Carried State Transfer (ECST).
Each order message contains all information necessary for processing, eliminating the need
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
	ErrEmptyOrderID        = errors.New("order_id is required")
	ErrInvalidSequence     = errors.New("sequence must be positive")
	ErrEmptyStatus         = errors.New("status is required")
	ErrNoItems             = errors.New("order must contain at least one item")
	ErrInvalidCustomerID   = errors.New("customer_id is required")
	ErrInvalidCustomerName = errors.New("customer name is required")
	ErrInvalidEmail        = errors.New("invalid email format")
	ErrInvalidItemID       = errors.New("item_id is required")
	ErrInvalidItemName     = errors.New("item_name is required")
	ErrInvalidQuantity     = errors.New("quantity must be positive")
	ErrInvalidUnitPrice    = errors.New("unit_price must be positive")
	ErrInvalidTotalPrice   = errors.New("total_price does not match quantity * unit_price")
	ErrInvalidSubtotal     = errors.New("subtotal does not match sum of items")
	ErrInvalidTax          = errors.New("tax must be positive or zero")
	ErrInvalidTotal        = errors.New("total is inconsistent with subtotal + tax + shipping fee")
)

// emailRegex verifies the email format
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// CustomerInfo contains detailed information about the customer.
// These data are embedded in every order message.
type CustomerInfo struct {
	CustomerID   string `json:"customer_id"`   // Unique identifier of the customer.
	Name         string `json:"name"`          // Full name of the customer.
	Email        string `json:"email"`         // Email address of the customer.
	Phone        string `json:"phone"`         // Phone number of the customer.
	Address      string `json:"address"`       // Physical address of the customer.
	LoyaltyLevel string `json:"loyalty_level"` // Loyalty level (e.g., "silver", "gold").
}

// Validate checks that the customer information is valid.
//
// Returns:
//   - error: An error if a required field is missing or invalid.
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

// InventoryStatus represents the inventory state for a specific item at the time of the order.
// This information allows understanding the stock context without querying an inventory service.
type InventoryStatus struct {
	ItemID       string  `json:"item_id"`       // Identifier of the item in stock.
	ItemName     string  `json:"item_name"`     // Name of the item.
	AvailableQty int     `json:"available_qty"` // Quantity available before the order.
	ReservedQty  int     `json:"reserved_qty"`  // Quantity reserved by this order.
	UnitPrice    float64 `json:"unit_price"`    // Unit price.
	InStock      bool    `json:"in_stock"`      // Availability indicator (true if stock > 0).
	Warehouse    string  `json:"warehouse"`     // Origin warehouse.
}

// OrderItem represents an individual item within an order.
type OrderItem struct {
	ItemID     string  `json:"item_id"`     // Unique identifier of the item.
	ItemName   string  `json:"item_name"`   // Name of the item.
	Quantity   int     `json:"quantity"`    // Ordered quantity.
	UnitPrice  float64 `json:"unit_price"`  // Unit price.
	TotalPrice float64 `json:"total_price"` // Total price for this item (Quantity * Price).
}

// Validate checks that an order item is valid.
//
// Returns:
//   - error: An error if the item data is invalid or inconsistent.
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
		return fmt.Errorf("%w: expected %.2f, got %.2f", ErrInvalidTotalPrice, expectedTotal, item.TotalPrice)
	}
	return nil
}

// OrderMetadata contains technical and contextual metadata for the order event.
// This information is essential for tracking, debugging, and analyzing the message flow.
type OrderMetadata struct {
	Timestamp     string `json:"timestamp"`      // Event creation timestamp (RFC3339).
	Version       string `json:"version"`        // Data schema version.
	EventType     string `json:"event_type"`     // Event type (e.g., "order.created").
	Source        string `json:"source"`         // Event source (e.g., "producer-service").
	CorrelationID string `json:"correlation_id"` // Correlation identifier for distributed tracing.
}

// Order is the main structure representing a complete customer order.
// It aggregates all necessary information for processing into a single message,
// following the Event-Carried State Transfer (ECST) pattern.
type Order struct {
	// Identifiers
	OrderID  string `json:"order_id"` // Unique identifier of the order (UUID).
	Sequence int    `json:"sequence"` // Incremental sequence number.
	Status   string `json:"status"`   // Status of the order (e.g., "pending").

	// Customer Information (denormalized for ECST)
	CustomerInfo CustomerInfo `json:"customer_info"`

	// Order Items
	Items []OrderItem `json:"items"`

	// Inventory Snapshot at the time of order
	Inventory InventoryStatus `json:"inventory"`

	// Financial Details
	SubTotal    float64 `json:"subtotal"`     // Sum of items.
	Tax         float64 `json:"tax"`          // Tax amount.
	ShippingFee float64 `json:"shipping_fee"` // Shipping fee.
	Total       float64 `json:"total"`        // Total amount.
	Currency    string  `json:"currency"`     // Currency (e.g., "EUR").

	// Payment and Delivery
	PaymentMethod string `json:"payment_method"`           // Payment method used.
	DeliveryNotes string `json:"delivery_notes,omitempty"` // Optional delivery notes.

	// Event Metadata
	Metadata OrderMetadata `json:"metadata"`
}

// Validate checks that an order is valid.
// It validates all required fields and the consistency of amounts.
//
// Returns:
//   - error: An error if the order is invalid.
func (o *Order) Validate() error {
	// Basic Validations
	if strings.TrimSpace(o.OrderID) == "" {
		return ErrEmptyOrderID
	}
	if o.Sequence <= 0 {
		return ErrInvalidSequence
	}
	if strings.TrimSpace(o.Status) == "" {
		return ErrEmptyStatus
	}

	// Customer Validation
	if err := o.CustomerInfo.Validate(); err != nil {
		return err
	}

	// Items Validation
	if len(o.Items) == 0 {
		return ErrNoItems
	}

	var calculatedSubtotal float64
	for i, item := range o.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("item %d: %w", i+1, err)
		}
		calculatedSubtotal += item.TotalPrice
	}

	// Financial Validations
	if math.Abs(o.SubTotal-calculatedSubtotal) > 0.01 {
		return fmt.Errorf("%w: expected %.2f, got %.2f", ErrInvalidSubtotal, calculatedSubtotal, o.SubTotal)
	}

	if o.Tax < 0 {
		return ErrInvalidTax
	}

	expectedTotal := o.SubTotal + o.Tax + o.ShippingFee
	if math.Abs(o.Total-expectedTotal) > 0.01 {
		return fmt.Errorf("%w: expected %.2f, got %.2f", ErrInvalidTotal, expectedTotal, o.Total)
	}

	return nil
}

// IsValid returns true if the order is valid.
//
// Returns:
//   - bool: True if validation succeeds, false otherwise.
func (o *Order) IsValid() bool {
	return o.Validate() == nil
}
