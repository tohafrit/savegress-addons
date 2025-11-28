package connectors

import (
	"context"

	"github.com/savegress/streamline/pkg/models"
	"github.com/shopspring/decimal"
)

// Connector defines the interface for channel connectors
type Connector interface {
	// Identity
	GetID() string
	GetType() models.ChannelType
	GetName() string

	// Connection
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	TestConnection(ctx context.Context) error

	// Products
	GetProducts(ctx context.Context, limit, offset int) ([]*models.Product, error)
	GetProduct(ctx context.Context, externalID string) (*models.Product, error)
	CreateProduct(ctx context.Context, product *models.Product) (string, error)
	UpdateProduct(ctx context.Context, externalID string, product *models.Product) error
	DeleteProduct(ctx context.Context, externalID string) error

	// Inventory
	GetInventory(ctx context.Context, sku string) (int, error)
	UpdateInventory(ctx context.Context, sku string, quantity int) error
	GetInventoryBatch(ctx context.Context, skus []string) (map[string]int, error)
	UpdateInventoryBatch(ctx context.Context, updates map[string]int) error

	// Orders
	GetOrders(ctx context.Context, since *string, limit int) ([]*models.Order, error)
	GetOrder(ctx context.Context, externalID string) (*models.Order, error)
	UpdateOrderStatus(ctx context.Context, externalID string, status models.OrderStatus) error
	FulfillOrder(ctx context.Context, externalID string, fulfillment *models.Fulfillment) error
	CancelOrder(ctx context.Context, externalID string, reason string) error

	// Pricing
	GetPrice(ctx context.Context, sku string) (decimal.Decimal, error)
	UpdatePrice(ctx context.Context, sku string, price decimal.Decimal) error
	UpdatePriceBatch(ctx context.Context, prices map[string]decimal.Decimal) error

	// Webhooks
	RegisterWebhook(ctx context.Context, topic, url string) error
	UnregisterWebhook(ctx context.Context, webhookID string) error
}

// BaseConnector provides common functionality for connectors
type BaseConnector struct {
	ID        string
	Type      models.ChannelType
	Name      string
	Connected bool
}

func (c *BaseConnector) GetID() string {
	return c.ID
}

func (c *BaseConnector) GetType() models.ChannelType {
	return c.Type
}

func (c *BaseConnector) GetName() string {
	return c.Name
}

func (c *BaseConnector) IsConnected() bool {
	return c.Connected
}

// ConnectorConfig holds common connector configuration
type ConnectorConfig struct {
	ID             string
	Name           string
	SyncProducts   bool
	SyncOrders     bool
	SyncInventory  bool
	SyncPricing    bool
	WebhookURL     string
	RateLimitRPS   int
}

// ConnectorFactory creates connectors based on type
type ConnectorFactory struct {
	creators map[models.ChannelType]func(config interface{}) (Connector, error)
}

// NewConnectorFactory creates a new connector factory
func NewConnectorFactory() *ConnectorFactory {
	return &ConnectorFactory{
		creators: make(map[models.ChannelType]func(config interface{}) (Connector, error)),
	}
}

// Register registers a connector creator
func (f *ConnectorFactory) Register(channelType models.ChannelType, creator func(config interface{}) (Connector, error)) {
	f.creators[channelType] = creator
}

// Create creates a connector
func (f *ConnectorFactory) Create(channelType models.ChannelType, config interface{}) (Connector, error) {
	creator, ok := f.creators[channelType]
	if !ok {
		return nil, ErrUnsupportedChannel
	}
	return creator(config)
}

// Errors
var (
	ErrUnsupportedChannel = &ConnectorError{Message: "unsupported channel type"}
	ErrNotConnected       = &ConnectorError{Message: "not connected"}
	ErrRateLimited        = &ConnectorError{Message: "rate limited"}
	ErrProductNotFound    = &ConnectorError{Message: "product not found"}
	ErrOrderNotFound      = &ConnectorError{Message: "order not found"}
)

type ConnectorError struct {
	Message string
	Cause   error
}

func (e *ConnectorError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *ConnectorError) Unwrap() error {
	return e.Cause
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	ChannelID      string   `json:"channel_id"`
	Channel        string   `json:"channel,omitempty"`
	Type           string   `json:"type"` // products, orders, inventory
	Success        bool     `json:"success"`
	ItemsSync      int      `json:"items_synced"`
	ItemsFailed    int      `json:"items_failed"`
	ItemsProcessed int      `json:"items_processed,omitempty"`
	ItemsSucceeded int      `json:"items_succeeded,omitempty"`
	Errors         []string `json:"errors,omitempty"`
	Duration       string   `json:"duration"`
	StartedAt      string   `json:"started_at,omitempty"`
	CompletedAt    string   `json:"completed_at,omitempty"`
}

// ProductParams for querying products
type ProductParams struct {
	Limit        int
	Offset       int
	Query        string
	SKU          string
	UpdatedAfter string
}

// Product represents a product in the connector
type Product struct {
	ID          string              `json:"id"`
	SKU         string              `json:"sku"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Brand       string              `json:"brand,omitempty"`
	ImageURL    string              `json:"image_url,omitempty"`
	Price       float64             `json:"price,omitempty"`
	ListPrice   float64             `json:"list_price,omitempty"`
	Weight      float64             `json:"weight,omitempty"`
	Quantity    int                 `json:"quantity,omitempty"`
	Categories  []string            `json:"categories,omitempty"`
	Identifiers []ProductIdentifier `json:"identifiers,omitempty"`
	Attributes  map[string]string   `json:"attributes,omitempty"`
	ChannelID   string              `json:"channel_id"`
	Active      bool                `json:"active"`
}

// ProductIdentifier represents a product identifier (UPC, EAN, ISBN)
type ProductIdentifier struct {
	Type  string `json:"type"`  // UPC, EAN, ISBN, etc.
	Value string `json:"value"`
}

// InventoryLevel represents inventory level for a product
type InventoryLevel struct {
	SKU         string    `json:"sku"`
	ProductID   string    `json:"product_id,omitempty"`
	Quantity    int       `json:"quantity"`
	Available   int       `json:"available"`
	Reserved    int       `json:"reserved,omitempty"`
	ChannelID   string    `json:"channel_id,omitempty"`
	LocationID  string    `json:"location_id,omitempty"`
	LastUpdated string    `json:"last_updated,omitempty"`
}

// InventoryUpdate represents an inventory update request
type InventoryUpdate struct {
	SKU       string `json:"sku"`
	ProductID string `json:"product_id,omitempty"`
	Quantity  int    `json:"quantity"`
}

// OrderParams for querying orders
type OrderParams struct {
	Limit        int
	Offset       int
	Since        string
	Status       string
	CreatedAfter string
}

// Order represents an order from a channel
type Order struct {
	ID                string      `json:"id"`
	ExternalID        string      `json:"external_id"`
	ChannelID         string      `json:"channel_id"`
	ChannelOrderID    string      `json:"channel_order_id,omitempty"`
	Status            string      `json:"status"`
	PaymentStatus     string      `json:"payment_status,omitempty"`
	FulfillmentStatus string      `json:"fulfillment_status,omitempty"`
	Total             float64     `json:"total"`
	Tax               float64     `json:"tax,omitempty"`
	Shipping          float64     `json:"shipping,omitempty"`
	Discount          float64     `json:"discount,omitempty"`
	Currency          string      `json:"currency"`
	Items             []OrderItem `json:"items"`
	ShippingAddress   *Address    `json:"shipping_address,omitempty"`
	BillingAddress    *Address    `json:"billing_address,omitempty"`
	ShippingMethod    string      `json:"shipping_method,omitempty"`
	CustomerID        string      `json:"customer_id,omitempty"`
	CustomerEmail     string      `json:"customer_email,omitempty"`
	CustomerName      string      `json:"customer_name,omitempty"`
	CustomerNote      string      `json:"customer_note,omitempty"`
	Notes             string      `json:"notes,omitempty"`
	CreatedAt         string      `json:"created_at"`
	UpdatedAt         string      `json:"updated_at,omitempty"`
}

// OrderStatus constants
const (
	OrderStatusPending    = "pending"
	OrderStatusProcessing = "processing"
	OrderStatusShipped    = "shipped"
	OrderStatusDelivered  = "delivered"
	OrderStatusCancelled  = "cancelled"
	OrderStatusRefunded   = "refunded"
)

// OrderItem represents an item in an order
type OrderItem struct {
	ID        string  `json:"id,omitempty"`
	ProductID string  `json:"product_id,omitempty"`
	SKU       string  `json:"sku"`
	Title     string  `json:"title"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
	Total     float64 `json:"total"`
	Currency  string  `json:"currency,omitempty"`
}

// Address represents a shipping/billing address
type Address struct {
	Name        string `json:"name,omitempty"`
	Line1       string `json:"line1"`
	Line2       string `json:"line2,omitempty"`
	City        string `json:"city"`
	State       string `json:"state"`
	PostalCode  string `json:"postal_code"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
}

// Price represents product pricing
type Price struct {
	SKU         string  `json:"sku"`
	ProductID   string  `json:"product_id,omitempty"`
	Price       float64 `json:"price"`
	ListPrice   float64 `json:"list_price,omitempty"`
	SalePrice   float64 `json:"sale_price,omitempty"`
	OnSale      bool    `json:"on_sale,omitempty"`
	Currency    string  `json:"currency,omitempty"`
	ChannelID   string  `json:"channel_id,omitempty"`
	LastUpdated string  `json:"last_updated,omitempty"`
}

// PriceUpdate represents a price update request
type PriceUpdate struct {
	SKU       string  `json:"sku"`
	ProductID string  `json:"product_id,omitempty"`
	Price     float64 `json:"price"`
	ListPrice float64 `json:"list_price,omitempty"`
	SalePrice float64 `json:"sale_price,omitempty"`
	Currency  string  `json:"currency,omitempty"`
}

// Webhook represents a webhook subscription
type Webhook struct {
	ID        string   `json:"id"`
	Topic     string   `json:"topic"`
	URL       string   `json:"url"`
	Events    []string `json:"events,omitempty"`
	Active    bool     `json:"active"`
	CreatedAt string   `json:"created_at,omitempty"`
}

// Shipment represents shipping information
type Shipment struct {
	OrderID        string `json:"order_id"`
	TrackingNumber string `json:"tracking_number"`
	Carrier        string `json:"carrier"`
	ShippedAt      string `json:"shipped_at,omitempty"`
}
