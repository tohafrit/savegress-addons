package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Product represents a product in the catalog
type Product struct {
	ID          string            `json:"id"`
	ExternalID  string            `json:"external_id,omitempty"` // ID in external channel (e.g., ASIN for Amazon)
	SKU         string            `json:"sku"`
	Name        string            `json:"name"`
	Title       string            `json:"title,omitempty"`       // Product title (for channels like Amazon)
	Description string            `json:"description,omitempty"`
	ProductType string            `json:"product_type,omitempty"` // Product type/category
	Channel     ChannelType       `json:"channel,omitempty"`      // Source channel
	ChannelID   string            `json:"channel_id,omitempty"`   // ID of the channel
	BasePrice   decimal.Decimal   `json:"base_price"`
	Cost        decimal.Decimal   `json:"cost"`
	Weight      float64           `json:"weight,omitempty"`
	Dimensions  *Dimensions       `json:"dimensions,omitempty"`
	Images      []string          `json:"images,omitempty"`
	Variants    []Variant         `json:"variants,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Status      ProductStatus     `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type ProductStatus string

const (
	ProductStatusActive   ProductStatus = "active"
	ProductStatusInactive ProductStatus = "inactive"
	ProductStatusDraft    ProductStatus = "draft"
)

type Dimensions struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Unit   string  `json:"unit"` // cm, in
}

type Variant struct {
	ID         string            `json:"id"`
	SKU        string            `json:"sku"`
	Name       string            `json:"name"`
	Price      decimal.Decimal   `json:"price"`
	Cost       decimal.Decimal   `json:"cost"`
	Weight     float64           `json:"weight,omitempty"`
	Attributes map[string]string `json:"attributes"` // color, size, etc.
}

// Inventory represents stock levels for a product
type Inventory struct {
	ProductID   string                    `json:"product_id"`
	SKU         string                    `json:"sku"`
	TotalStock  int                       `json:"total_stock"`
	Allocated   map[string]int            `json:"allocated"`   // channel -> qty
	Reserved    int                       `json:"reserved"`    // pending orders
	Available   int                       `json:"available"`   // total - reserved
	Warehouses  map[string]WarehouseStock `json:"warehouses"`
	ReorderPoint int                      `json:"reorder_point"`
	LastUpdated time.Time                 `json:"last_updated"`
}

type WarehouseStock struct {
	WarehouseID string `json:"warehouse_id"`
	Quantity    int    `json:"quantity"`
	Reserved    int    `json:"reserved"`
	Available   int    `json:"available"`
	Location    string `json:"location,omitempty"` // bin/shelf location
}

// InventoryAdjustment represents a stock adjustment
type InventoryAdjustment struct {
	ID          string          `json:"id"`
	SKU         string          `json:"sku"`
	WarehouseID string          `json:"warehouse_id"`
	Quantity    int             `json:"quantity"` // positive or negative
	Reason      AdjustmentReason `json:"reason"`
	Reference   string          `json:"reference,omitempty"` // order ID, PO number, etc.
	Notes       string          `json:"notes,omitempty"`
	CreatedBy   string          `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
}

type AdjustmentReason string

const (
	AdjustmentReasonReceived    AdjustmentReason = "received"
	AdjustmentReasonSold        AdjustmentReason = "sold"
	AdjustmentReasonReturned    AdjustmentReason = "returned"
	AdjustmentReasonDamaged     AdjustmentReason = "damaged"
	AdjustmentReasonLost        AdjustmentReason = "lost"
	AdjustmentReasonCount       AdjustmentReason = "count"
	AdjustmentReasonTransfer    AdjustmentReason = "transfer"
	AdjustmentReasonCorrection  AdjustmentReason = "correction"
)

// Order represents a sales order
type Order struct {
	ID              string                 `json:"id"`
	ExternalID      string                 `json:"external_id,omitempty"` // order ID in the channel
	Channel         ChannelType            `json:"channel"`
	ChannelID       string                 `json:"channel_id,omitempty"`
	ChannelRef      string                 `json:"channel_ref,omitempty"` // legacy field
	Status          OrderStatus            `json:"status"`
	Total           decimal.Decimal        `json:"total,omitempty"`    // simplified total field
	Currency        string                 `json:"currency,omitempty"` // currency code
	Items           []OrderItem            `json:"items"`
	Customer        Customer               `json:"customer"`
	Shipping        ShippingInfo           `json:"shipping"`
	ShippingAddress Address                `json:"shipping_address,omitempty"` // direct address field
	Billing         *BillingInfo           `json:"billing,omitempty"`
	Totals          OrderTotals            `json:"totals"`
	WarehouseID     string                 `json:"warehouse_id,omitempty"`
	Fulfillment     *Fulfillment           `json:"fulfillment,omitempty"`
	Notes           string                 `json:"notes,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type OrderStatus string

const (
	OrderStatusPending          OrderStatus = "pending"
	OrderStatusProcessing       OrderStatus = "processing"
	OrderStatusShipped          OrderStatus = "shipped"
	OrderStatusPartiallyShipped OrderStatus = "partially_shipped"
	OrderStatusDelivered        OrderStatus = "delivered"
	OrderStatusCancelled        OrderStatus = "cancelled"
	OrderStatusCanceled         OrderStatus = "canceled" // alternate spelling
	OrderStatusRefunded         OrderStatus = "refunded"
	OrderStatusOnHold           OrderStatus = "on_hold"
	OrderStatusFailed           OrderStatus = "failed"
)

type OrderItem struct {
	ID        string          `json:"id"`
	ProductID string          `json:"product_id"`
	SKU       string          `json:"sku"`
	Name      string          `json:"name"`
	Quantity  int             `json:"quantity"`
	Price     decimal.Decimal `json:"price"`
	Discount  decimal.Decimal `json:"discount"`
	Tax       decimal.Decimal `json:"tax"`
	Total     decimal.Decimal `json:"total"`
}

type Customer struct {
	ID        string `json:"id,omitempty"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone,omitempty"`
}

type ShippingInfo struct {
	Method    string  `json:"method"`
	Carrier   string  `json:"carrier,omitempty"`
	Address   Address `json:"address"`
	Cost      decimal.Decimal `json:"cost"`
	Estimated string  `json:"estimated,omitempty"` // estimated delivery date
}

type BillingInfo struct {
	Address Address `json:"address"`
}

type Address struct {
	Name       string `json:"name,omitempty"`
	Line1      string `json:"line1"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
	Phone      string `json:"phone,omitempty"`
}

type OrderTotals struct {
	Subtotal decimal.Decimal `json:"subtotal"`
	Shipping decimal.Decimal `json:"shipping"`
	Tax      decimal.Decimal `json:"tax"`
	Discount decimal.Decimal `json:"discount"`
	Total    decimal.Decimal `json:"total"`
	Currency string          `json:"currency"`
}

type Fulfillment struct {
	ID          string            `json:"id"`
	Status      FulfillmentStatus `json:"status"`
	WarehouseID string            `json:"warehouse_id"`
	TrackingNum string            `json:"tracking_number,omitempty"`
	Carrier     string            `json:"carrier,omitempty"`
	ShippedAt   *time.Time        `json:"shipped_at,omitempty"`
	DeliveredAt *time.Time        `json:"delivered_at,omitempty"`
}

type FulfillmentStatus string

const (
	FulfillmentPending   FulfillmentStatus = "pending"
	FulfillmentPicking   FulfillmentStatus = "picking"
	FulfillmentPacked    FulfillmentStatus = "packed"
	FulfillmentShipped   FulfillmentStatus = "shipped"
	FulfillmentDelivered FulfillmentStatus = "delivered"
)

// Warehouse represents a fulfillment location
type Warehouse struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Code         string   `json:"code"` // short code
	Address      Address  `json:"address"`
	Priority     int      `json:"priority"` // for routing
	Capabilities []string `json:"capabilities"` // standard, express, fragile, cold_chain
	Active       bool     `json:"active"`
}

// Channel represents a connected sales channel
type Channel struct {
	ID           string        `json:"id"`
	Type         ChannelType   `json:"type"`
	Name         string        `json:"name"`
	Status       ChannelStatus `json:"status"`
	Config       map[string]interface{} `json:"config"`
	SyncProducts bool          `json:"sync_products"`
	SyncOrders   bool          `json:"sync_orders"`
	SyncInventory bool         `json:"sync_inventory"`
	LastSyncAt   *time.Time    `json:"last_sync_at,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
}

type ChannelType string

const (
	ChannelTypeShopify     ChannelType = "shopify"
	ChannelTypeAmazon      ChannelType = "amazon"
	ChannelTypeWooCommerce ChannelType = "woocommerce"
	ChannelTypeEbay        ChannelType = "ebay"
	ChannelTypeWalmart     ChannelType = "walmart"
	ChannelTypeEtsy        ChannelType = "etsy"
)

type ChannelStatus string

const (
	ChannelStatusActive       ChannelStatus = "active"
	ChannelStatusInactive     ChannelStatus = "inactive"
	ChannelStatusError        ChannelStatus = "error"
	ChannelStatusPendingSetup ChannelStatus = "pending_setup"
)

// ChannelListing represents a product listing on a channel
type ChannelListing struct {
	ChannelID  string          `json:"channel_id"`
	ProductID  string          `json:"product_id"`
	ExternalID string          `json:"external_id"` // ID in the channel
	Price      decimal.Decimal `json:"price"`
	Stock      int             `json:"stock"`
	Status     string          `json:"status"` // active, paused, out_of_stock
	URL        string          `json:"url,omitempty"`
	LastSyncAt time.Time       `json:"last_sync_at"`
}

// PricingRule represents a pricing rule
type PricingRule struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Type        PricingRuleType `json:"type"`
	Channels    []string        `json:"channels,omitempty"` // empty = all channels
	Products    []string        `json:"products,omitempty"` // SKUs, empty = all products
	Condition   string          `json:"condition,omitempty"`
	Action      PricingAction   `json:"action"`
	Priority    int             `json:"priority"`
	StartDate   *time.Time      `json:"start_date,omitempty"`
	EndDate     *time.Time      `json:"end_date,omitempty"`
	Active      bool            `json:"active"`
}

type PricingRuleType string

const (
	PricingRuleTypeMarkup       PricingRuleType = "markup"
	PricingRuleTypeDiscount     PricingRuleType = "discount"
	PricingRuleTypeFixed        PricingRuleType = "fixed"
	PricingRuleTypeCompetitive  PricingRuleType = "competitive"
	PricingRuleTypeMap          PricingRuleType = "map" // minimum advertised price
)

type PricingAction struct {
	Type       string          `json:"type"`   // percent, fixed, match
	Value      decimal.Decimal `json:"value,omitempty"`
	MinPrice   decimal.Decimal `json:"min_price,omitempty"`
	MaxPrice   decimal.Decimal `json:"max_price,omitempty"`
	RoundTo    string          `json:"round_to,omitempty"` // 0.99, 0.95, etc.
}

// Alert represents a stock or sync alert
type Alert struct {
	ID          string       `json:"id"`
	Type        AlertType    `json:"type"`
	Severity    AlertSeverity `json:"severity"`
	SKU         string       `json:"sku,omitempty"`
	Channel     string       `json:"channel,omitempty"`
	Message     string       `json:"message"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Acknowledged bool        `json:"acknowledged"`
	AcknowledgedBy string   `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

type AlertType string

const (
	AlertTypeOutOfStock    AlertType = "out_of_stock"
	AlertTypeLowStock      AlertType = "low_stock"
	AlertTypeOversold      AlertType = "oversold"
	AlertTypeDiscrepancy   AlertType = "discrepancy"
	AlertTypeSyncError     AlertType = "sync_error"
	AlertTypeHighDemand    AlertType = "high_demand"
	AlertTypeSlowMoving    AlertType = "slow_moving"
	AlertTypePriceChange   AlertType = "price_change"
)

type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityInfo     AlertSeverity = "info"
)
