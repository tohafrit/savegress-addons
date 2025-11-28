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
	ChannelID    string    `json:"channel_id"`
	Type         string    `json:"type"` // products, orders, inventory
	Success      bool      `json:"success"`
	ItemsSync    int       `json:"items_synced"`
	ItemsFailed  int       `json:"items_failed"`
	Errors       []string  `json:"errors,omitempty"`
	Duration     string    `json:"duration"`
}
