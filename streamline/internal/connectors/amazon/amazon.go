package amazon

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/savegress/streamline/internal/connectors"
	"github.com/savegress/streamline/pkg/models"
	"github.com/shopspring/decimal"
)

// Config holds Amazon Seller API configuration
type Config struct {
	connectors.ConnectorConfig
	SellerID        string
	MarketplaceID   string
	AWSAccessKey    string
	AWSSecretKey    string
	RefreshToken    string
	Region          string // NA, EU, FE
	Endpoint        string
	RoleARN         string
}

// Connector implements the Amazon Seller connector
type Connector struct {
	connectors.BaseConnector
	config      *Config
	httpClient  *http.Client
	accessToken string
	tokenExpiry time.Time
	mu          sync.RWMutex
}

// NewConnector creates a new Amazon connector
func NewConnector(config *Config) (*Connector, error) {
	if config.SellerID == "" {
		return nil, fmt.Errorf("seller ID is required")
	}

	c := &Connector{
		BaseConnector: connectors.BaseConnector{
			ID:   config.ID,
			Type: models.ChannelTypeAmazon,
			Name: config.Name,
		},
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Set default endpoint based on region
	if config.Endpoint == "" {
		switch config.Region {
		case "EU":
			config.Endpoint = "https://sellingpartnerapi-eu.amazon.com"
		case "FE":
			config.Endpoint = "https://sellingpartnerapi-fe.amazon.com"
		default:
			config.Endpoint = "https://sellingpartnerapi-na.amazon.com"
		}
	}

	return c, nil
}

// Connect establishes connection to Amazon API
func (c *Connector) Connect(ctx context.Context) error {
	if err := c.refreshAccessToken(ctx); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	c.Connected = true
	return nil
}

// Disconnect disconnects from Amazon API
func (c *Connector) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accessToken = ""
	c.Connected = false
	return nil
}

// TestConnection tests the connection
func (c *Connector) TestConnection(ctx context.Context) error {
	_, err := c.makeRequest(ctx, "GET", "/sellers/v1/marketplaceParticipations", nil)
	return err
}

func (c *Connector) refreshAccessToken(ctx context.Context) error {
	// LWA (Login with Amazon) token refresh
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", c.config.RefreshToken)
	data.Set("client_id", c.config.AWSAccessKey)
	data.Set("client_secret", c.config.AWSSecretKey)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.amazon.com/auth/o2/token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	c.mu.Lock()
	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	c.mu.Unlock()

	return nil
}

func (c *Connector) makeRequest(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	c.mu.RLock()
	if time.Now().After(c.tokenExpiry) {
		c.mu.RUnlock()
		if err := c.refreshAccessToken(ctx); err != nil {
			return nil, err
		}
		c.mu.RLock()
	}
	token := c.accessToken
	c.mu.RUnlock()

	reqURL := c.config.Endpoint + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}

	// Sign request with AWS Signature Version 4
	c.signRequest(req)
	req.Header.Set("x-amz-access-token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *Connector) signRequest(req *http.Request) {
	// Simplified AWS Signature V4 signing
	// In production, use proper AWS SDK signing
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	date := timestamp[:8]

	req.Header.Set("x-amz-date", timestamp)
	req.Header.Set("host", req.URL.Host)

	// Create canonical request and sign
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-date:%s\n", req.URL.Host, timestamp)
	signedHeaders := "host;x-amz-date"

	hasher := sha256.New()
	hasher.Write([]byte("")) // empty body for GET requests
	payloadHash := hex.EncodeToString(hasher.Sum(nil))

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method,
		req.URL.Path,
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash)

	hasher = sha256.New()
	hasher.Write([]byte(canonicalRequest))
	canonicalRequestHash := hex.EncodeToString(hasher.Sum(nil))

	scope := fmt.Sprintf("%s/us-east-1/execute-api/aws4_request", date)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		timestamp, scope, canonicalRequestHash)

	// Calculate signature
	signingKey := c.getSignatureKey(c.config.AWSSecretKey, date, "us-east-1", "execute-api")
	signature := hex.EncodeToString(c.hmacSHA256(signingKey, stringToSign))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.config.AWSAccessKey, scope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

func (c *Connector) hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func (c *Connector) getSignatureKey(key, dateStamp, regionName, serviceName string) []byte {
	kDate := c.hmacSHA256([]byte("AWS4"+key), dateStamp)
	kRegion := c.hmacSHA256(kDate, regionName)
	kService := c.hmacSHA256(kRegion, serviceName)
	kSigning := c.hmacSHA256(kService, "aws4_request")
	return kSigning
}

// GetProducts returns products from Amazon
func (c *Connector) GetProducts(ctx context.Context, limit, offset int) ([]*models.Product, error) {
	path := fmt.Sprintf("/catalog/2022-04-01/items?marketplaceIds=%s&pageSize=%d",
		c.config.MarketplaceID, limit)

	body, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Items []struct {
			ASIN           string `json:"asin"`
			Title          string `json:"title,omitempty"`
			ProductType    string `json:"productType,omitempty"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var products []*models.Product
	for _, item := range resp.Items {
		products = append(products, &models.Product{
			ExternalID:  item.ASIN,
			Title:       item.Title,
			ProductType: item.ProductType,
			Channel:     models.ChannelTypeAmazon,
			ChannelID:   c.ID,
		})
	}

	return products, nil
}

// GetProduct returns a single product
func (c *Connector) GetProduct(ctx context.Context, externalID string) (*models.Product, error) {
	path := fmt.Sprintf("/catalog/2022-04-01/items/%s?marketplaceIds=%s",
		externalID, c.config.MarketplaceID)

	body, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var item struct {
		ASIN           string `json:"asin"`
		Title          string `json:"title,omitempty"`
		ProductType    string `json:"productType,omitempty"`
	}
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, err
	}

	return &models.Product{
		ExternalID:  item.ASIN,
		Title:       item.Title,
		ProductType: item.ProductType,
		Channel:     models.ChannelTypeAmazon,
		ChannelID:   c.ID,
	}, nil
}

// CreateProduct creates a product on Amazon
func (c *Connector) CreateProduct(ctx context.Context, product *models.Product) (string, error) {
	// Amazon uses Listings API for creating products
	return "", fmt.Errorf("product creation requires Listings API integration")
}

// UpdateProduct updates a product on Amazon
func (c *Connector) UpdateProduct(ctx context.Context, externalID string, product *models.Product) error {
	return fmt.Errorf("product update requires Listings API integration")
}

// DeleteProduct deletes a product from Amazon
func (c *Connector) DeleteProduct(ctx context.Context, externalID string) error {
	return fmt.Errorf("product deletion requires Listings API integration")
}

// GetInventory returns inventory for a SKU
func (c *Connector) GetInventory(ctx context.Context, sku string) (int, error) {
	path := fmt.Sprintf("/fba/inventory/v1/summaries?sellerSkus=%s&marketplaceIds=%s&granularityType=Marketplace&granularityId=%s",
		url.QueryEscape(sku), c.config.MarketplaceID, c.config.MarketplaceID)

	body, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return 0, err
	}

	var resp struct {
		Payload struct {
			InventorySummaries []struct {
				SellerSku                  string `json:"sellerSku"`
				FnSku                      string `json:"fnSku"`
				ASIN                       string `json:"asin"`
				TotalFulfillableQuantity   int    `json:"totalFulfillableQuantity"`
			} `json:"inventorySummaries"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, err
	}

	for _, summary := range resp.Payload.InventorySummaries {
		if summary.SellerSku == sku {
			return summary.TotalFulfillableQuantity, nil
		}
	}

	return 0, nil
}

// UpdateInventory updates inventory for a SKU
func (c *Connector) UpdateInventory(ctx context.Context, sku string, quantity int) error {
	// For FBA, inventory is managed by Amazon
	// For MFN (Merchant Fulfilled), use Listings API
	return fmt.Errorf("inventory update requires Listings API integration")
}

// GetInventoryBatch returns inventory for multiple SKUs
func (c *Connector) GetInventoryBatch(ctx context.Context, skus []string) (map[string]int, error) {
	result := make(map[string]int)

	// Amazon API allows up to 50 SKUs per request
	batchSize := 50
	for i := 0; i < len(skus); i += batchSize {
		end := i + batchSize
		if end > len(skus) {
			end = len(skus)
		}
		batch := skus[i:end]

		skuList := strings.Join(batch, ",")
		path := fmt.Sprintf("/fba/inventory/v1/summaries?sellerSkus=%s&marketplaceIds=%s&granularityType=Marketplace&granularityId=%s",
			url.QueryEscape(skuList), c.config.MarketplaceID, c.config.MarketplaceID)

		body, err := c.makeRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Payload struct {
				InventorySummaries []struct {
					SellerSku                string `json:"sellerSku"`
					TotalFulfillableQuantity int    `json:"totalFulfillableQuantity"`
				} `json:"inventorySummaries"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, err
		}

		for _, summary := range resp.Payload.InventorySummaries {
			result[summary.SellerSku] = summary.TotalFulfillableQuantity
		}
	}

	return result, nil
}

// UpdateInventoryBatch updates inventory for multiple SKUs
func (c *Connector) UpdateInventoryBatch(ctx context.Context, updates map[string]int) error {
	return fmt.Errorf("batch inventory update requires Listings API integration")
}

// GetOrders returns orders from Amazon
func (c *Connector) GetOrders(ctx context.Context, since *string, limit int) ([]*models.Order, error) {
	path := fmt.Sprintf("/orders/v0/orders?MarketplaceIds=%s&MaxResultsPerPage=%d",
		c.config.MarketplaceID, limit)

	if since != nil {
		path += "&CreatedAfter=" + url.QueryEscape(*since)
	}

	body, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Payload struct {
			Orders []struct {
				AmazonOrderId string `json:"AmazonOrderId"`
				PurchaseDate  string `json:"PurchaseDate"`
				OrderStatus   string `json:"OrderStatus"`
				OrderTotal    struct {
					Amount       string `json:"Amount"`
					CurrencyCode string `json:"CurrencyCode"`
				} `json:"OrderTotal"`
				ShippingAddress struct {
					Name          string `json:"Name"`
					AddressLine1  string `json:"AddressLine1"`
					City          string `json:"City"`
					StateOrRegion string `json:"StateOrRegion"`
					PostalCode    string `json:"PostalCode"`
					CountryCode   string `json:"CountryCode"`
				} `json:"ShippingAddress"`
			} `json:"Orders"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var orders []*models.Order
	for _, o := range resp.Payload.Orders {
		total, _ := decimal.NewFromString(o.OrderTotal.Amount)
		purchaseDate, _ := time.Parse(time.RFC3339, o.PurchaseDate)

		orders = append(orders, &models.Order{
			ExternalID: o.AmazonOrderId,
			Channel:    models.ChannelTypeAmazon,
			ChannelID:  c.ID,
			Status:     mapAmazonOrderStatus(o.OrderStatus),
			Total:      total,
			Currency:   o.OrderTotal.CurrencyCode,
			ShippingAddress: models.Address{
				Name:       o.ShippingAddress.Name,
				Line1:      o.ShippingAddress.AddressLine1,
				City:       o.ShippingAddress.City,
				State:      o.ShippingAddress.StateOrRegion,
				PostalCode: o.ShippingAddress.PostalCode,
				Country:    o.ShippingAddress.CountryCode,
			},
			CreatedAt: purchaseDate,
		})
	}

	return orders, nil
}

// GetOrder returns a single order
func (c *Connector) GetOrder(ctx context.Context, externalID string) (*models.Order, error) {
	path := fmt.Sprintf("/orders/v0/orders/%s", externalID)

	body, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Payload struct {
			AmazonOrderId string `json:"AmazonOrderId"`
			PurchaseDate  string `json:"PurchaseDate"`
			OrderStatus   string `json:"OrderStatus"`
			OrderTotal    struct {
				Amount       string `json:"Amount"`
				CurrencyCode string `json:"CurrencyCode"`
			} `json:"OrderTotal"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	total, _ := decimal.NewFromString(resp.Payload.OrderTotal.Amount)
	purchaseDate, _ := time.Parse(time.RFC3339, resp.Payload.PurchaseDate)

	return &models.Order{
		ExternalID: resp.Payload.AmazonOrderId,
		Channel:    models.ChannelTypeAmazon,
		ChannelID:  c.ID,
		Status:     mapAmazonOrderStatus(resp.Payload.OrderStatus),
		Total:      total,
		Currency:   resp.Payload.OrderTotal.CurrencyCode,
		CreatedAt:  purchaseDate,
	}, nil
}

// UpdateOrderStatus updates order status
func (c *Connector) UpdateOrderStatus(ctx context.Context, externalID string, status models.OrderStatus) error {
	// Amazon manages order status internally
	return fmt.Errorf("order status is managed by Amazon")
}

// FulfillOrder fulfills an order
func (c *Connector) FulfillOrder(ctx context.Context, externalID string, fulfillment *models.Fulfillment) error {
	// Use Merchant Fulfillment API for MFN orders
	// FBA orders are fulfilled by Amazon
	return fmt.Errorf("order fulfillment requires Merchant Fulfillment API")
}

// CancelOrder cancels an order
func (c *Connector) CancelOrder(ctx context.Context, externalID string, reason string) error {
	return fmt.Errorf("order cancellation not supported via API")
}

// GetPrice returns price for a SKU
func (c *Connector) GetPrice(ctx context.Context, sku string) (decimal.Decimal, error) {
	path := fmt.Sprintf("/products/pricing/v0/price?MarketplaceId=%s&Skus=%s&ItemType=Sku",
		c.config.MarketplaceID, url.QueryEscape(sku))

	body, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return decimal.Zero, err
	}

	var resp struct {
		Payload []struct {
			SKU   string `json:"SKU"`
			Price struct {
				LandedPrice struct {
					Amount string `json:"Amount"`
				} `json:"LandedPrice"`
			} `json:"Price"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return decimal.Zero, err
	}

	if len(resp.Payload) > 0 {
		return decimal.NewFromString(resp.Payload[0].Price.LandedPrice.Amount)
	}

	return decimal.Zero, nil
}

// UpdatePrice updates price for a SKU
func (c *Connector) UpdatePrice(ctx context.Context, sku string, price decimal.Decimal) error {
	return fmt.Errorf("price update requires Listings API integration")
}

// UpdatePriceBatch updates prices for multiple SKUs
func (c *Connector) UpdatePriceBatch(ctx context.Context, prices map[string]decimal.Decimal) error {
	return fmt.Errorf("batch price update requires Listings API integration")
}

// RegisterWebhook registers a webhook (Amazon uses push notifications differently)
func (c *Connector) RegisterWebhook(ctx context.Context, topic, url string) error {
	// Amazon uses Notifications API with Amazon EventBridge
	return fmt.Errorf("webhooks require Amazon EventBridge configuration")
}

// UnregisterWebhook unregisters a webhook
func (c *Connector) UnregisterWebhook(ctx context.Context, webhookID string) error {
	return fmt.Errorf("webhooks require Amazon EventBridge configuration")
}

func mapAmazonOrderStatus(status string) models.OrderStatus {
	switch status {
	case "Pending":
		return models.OrderStatusPending
	case "Unshipped":
		return models.OrderStatusProcessing
	case "PartiallyShipped":
		return models.OrderStatusPartiallyShipped
	case "Shipped":
		return models.OrderStatusShipped
	case "Canceled":
		return models.OrderStatusCanceled
	case "Unfulfillable":
		return models.OrderStatusFailed
	default:
		return models.OrderStatusPending
	}
}
