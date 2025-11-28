package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Redis     RedisConfig     `yaml:"redis"`
	Inventory InventoryConfig `yaml:"inventory"`
	Orders    OrdersConfig    `yaml:"orders"`
	Pricing   PricingConfig   `yaml:"pricing"`
	Channels  ChannelsConfig  `yaml:"channels"`
	Alerts    AlertsConfig    `yaml:"alerts"`
}

type ServerConfig struct {
	Port        int    `yaml:"port"`
	Environment string `yaml:"environment"`
	JWTSecret   string `yaml:"jwt_secret"`
}

type DatabaseConfig struct {
	URL             string `yaml:"url"`
	MaxConns        int    `yaml:"max_conns"`
	MinConns        int    `yaml:"min_conns"`
}

type RedisConfig struct {
	URL string `yaml:"url"`
}

type InventoryConfig struct {
	SyncInterval       time.Duration `yaml:"sync_interval"`
	AllocationStrategy string        `yaml:"allocation_strategy"` // fixed, weighted, priority, dynamic
	SafetyBuffer       float64       `yaml:"safety_buffer"`       // percentage
	LowStockThreshold  int           `yaml:"low_stock_threshold"`
}

type OrdersConfig struct {
	RoutingOptimization string `yaml:"routing_optimization"` // cost, speed, balance
	AutoRoute           bool   `yaml:"auto_route"`
	HoldOversold        bool   `yaml:"hold_oversold"`
}

type PricingConfig struct {
	AutoSync           bool    `yaml:"auto_sync"`
	MinMargin          float64 `yaml:"min_margin"`      // percentage
	CompetitivePricing bool    `yaml:"competitive_pricing"`
}

type ChannelsConfig struct {
	Shopify     *ShopifyConfig     `yaml:"shopify,omitempty"`
	Amazon      *AmazonConfig      `yaml:"amazon,omitempty"`
	WooCommerce *WooCommerceConfig `yaml:"woocommerce,omitempty"`
	Ebay        *EbayConfig        `yaml:"ebay,omitempty"`
}

type ShopifyConfig struct {
	APIKey        string `yaml:"api_key"`
	APISecret     string `yaml:"api_secret"`
	ShopDomain    string `yaml:"shop_domain"`
	AccessToken   string `yaml:"access_token"`
	SyncProducts  bool   `yaml:"sync_products"`
	SyncOrders    bool   `yaml:"sync_orders"`
	SyncInventory bool   `yaml:"sync_inventory"`
}

type AmazonConfig struct {
	SellerID       string   `yaml:"seller_id"`
	MWSAuthToken   string   `yaml:"mws_auth_token"`
	AccessKeyID    string   `yaml:"access_key_id"`
	SecretKey      string   `yaml:"secret_key"`
	MarketplaceIDs []string `yaml:"marketplace_ids"`
	UseFBA         bool     `yaml:"use_fba"`
}

type WooCommerceConfig struct {
	URL            string `yaml:"url"`
	ConsumerKey    string `yaml:"consumer_key"`
	ConsumerSecret string `yaml:"consumer_secret"`
	SyncProducts   bool   `yaml:"sync_products"`
	SyncOrders     bool   `yaml:"sync_orders"`
	SyncInventory  bool   `yaml:"sync_inventory"`
}

type EbayConfig struct {
	AppID       string `yaml:"app_id"`
	CertID      string `yaml:"cert_id"`
	DevID       string `yaml:"dev_id"`
	AuthToken   string `yaml:"auth_token"`
	Environment string `yaml:"environment"` // sandbox, production
}

type AlertsConfig struct {
	Channels AlertChannelsConfig `yaml:"channels"`
}

type AlertChannelsConfig struct {
	Slack *SlackConfig `yaml:"slack,omitempty"`
	Email *EmailConfig `yaml:"email,omitempty"`
}

type SlackConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
}

type EmailConfig struct {
	SMTPHost   string   `yaml:"smtp_host"`
	SMTPPort   int      `yaml:"smtp_port"`
	From       string   `yaml:"from"`
	Recipients []string `yaml:"recipients"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	data = []byte(os.ExpandEnv(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	setDefaults(&cfg)
	return &cfg, nil
}

func LoadFromEnv() *Config {
	cfg := &Config{}
	setDefaults(cfg)

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.Database.URL = dbURL
	}
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		cfg.Redis.URL = redisURL
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.Server.JWTSecret = jwtSecret
	}

	// Shopify
	if apiKey := os.Getenv("SHOPIFY_API_KEY"); apiKey != "" {
		if cfg.Channels.Shopify == nil {
			cfg.Channels.Shopify = &ShopifyConfig{}
		}
		cfg.Channels.Shopify.APIKey = apiKey
		cfg.Channels.Shopify.APISecret = os.Getenv("SHOPIFY_API_SECRET")
		cfg.Channels.Shopify.ShopDomain = os.Getenv("SHOPIFY_SHOP_DOMAIN")
		cfg.Channels.Shopify.AccessToken = os.Getenv("SHOPIFY_ACCESS_TOKEN")
	}

	// Amazon
	if sellerID := os.Getenv("AMAZON_SELLER_ID"); sellerID != "" {
		if cfg.Channels.Amazon == nil {
			cfg.Channels.Amazon = &AmazonConfig{}
		}
		cfg.Channels.Amazon.SellerID = sellerID
		cfg.Channels.Amazon.MWSAuthToken = os.Getenv("AMAZON_MWS_TOKEN")
		cfg.Channels.Amazon.AccessKeyID = os.Getenv("AMAZON_ACCESS_KEY_ID")
		cfg.Channels.Amazon.SecretKey = os.Getenv("AMAZON_SECRET_KEY")
	}

	return cfg
}

func setDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 3003
	}
	if cfg.Server.Environment == "" {
		cfg.Server.Environment = "development"
	}
	if cfg.Database.MaxConns == 0 {
		cfg.Database.MaxConns = 25
	}
	if cfg.Database.MinConns == 0 {
		cfg.Database.MinConns = 5
	}
	if cfg.Inventory.SyncInterval == 0 {
		cfg.Inventory.SyncInterval = 30 * time.Second
	}
	if cfg.Inventory.AllocationStrategy == "" {
		cfg.Inventory.AllocationStrategy = "weighted"
	}
	if cfg.Inventory.SafetyBuffer == 0 {
		cfg.Inventory.SafetyBuffer = 10
	}
	if cfg.Inventory.LowStockThreshold == 0 {
		cfg.Inventory.LowStockThreshold = 15
	}
	if cfg.Orders.RoutingOptimization == "" {
		cfg.Orders.RoutingOptimization = "cost"
	}
	if cfg.Pricing.MinMargin == 0 {
		cfg.Pricing.MinMargin = 20
	}
}
