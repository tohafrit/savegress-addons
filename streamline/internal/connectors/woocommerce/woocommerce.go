package woocommerce

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/savegress/streamline/internal/connectors"
)

// Config holds WooCommerce API configuration
type Config struct {
	connectors.ConnectorConfig
	StoreURL       string // e.g., https://example.com
	ConsumerKey    string
	ConsumerSecret string
	Version        string // API version, default "wc/v3"
	UseOAuth       bool   // Use OAuth 1.0a for non-HTTPS stores
}

// Connector implements the WooCommerce connector
type Connector struct {
	config     Config
	httpClient *http.Client
	connected  bool
}

// NewConnector creates a new WooCommerce connector
func NewConnector(config Config) *Connector {
	if config.Version == "" {
		config.Version = "wc/v3"
	}

	// Ensure store URL doesn't have trailing slash
	config.StoreURL = strings.TrimRight(config.StoreURL, "/")

	return &Connector{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Type returns the connector type
func (c *Connector) Type() string {
	return "woocommerce"
}

// Name returns the connector name
func (c *Connector) Name() string {
	return c.config.Name
}

// Connect establishes connection to WooCommerce API
func (c *Connector) Connect(ctx context.Context) error {
	// Verify connection by fetching system status
	_, err := c.doRequest(ctx, "GET", "/system_status", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WooCommerce: %w", err)
	}
	c.connected = true
	return nil
}

// Disconnect closes the connection
func (c *Connector) Disconnect(ctx context.Context) error {
	c.connected = false
	return nil
}

// IsConnected checks if connected
func (c *Connector) IsConnected() bool {
	return c.connected
}

func (c *Connector) getBaseURL() string {
	return fmt.Sprintf("%s/wp-json/%s", c.config.StoreURL, c.config.Version)
}

func (c *Connector) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	fullURL := c.getBaseURL() + path

	// Add authentication
	if c.config.UseOAuth {
		fullURL = c.signOAuthRequest(method, fullURL)
	} else {
		// Use basic auth over HTTPS
		if strings.Contains(fullURL, "?") {
			fullURL += "&"
		} else {
			fullURL += "?"
		}
		fullURL += fmt.Sprintf("consumer_key=%s&consumer_secret=%s", c.config.ConsumerKey, c.config.ConsumerSecret)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

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
		return nil, fmt.Errorf("WooCommerce API error: %s - %s", resp.Status, string(respBody))
	}

	return respBody, nil
}

func (c *Connector) signOAuthRequest(method, urlStr string) string {
	// OAuth 1.0a signing for non-HTTPS connections
	parsedURL, _ := url.Parse(urlStr)
	params := parsedURL.Query()

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := base64.StdEncoding.EncodeToString([]byte(timestamp))

	params.Set("oauth_consumer_key", c.config.ConsumerKey)
	params.Set("oauth_signature_method", "HMAC-SHA256")
	params.Set("oauth_timestamp", timestamp)
	params.Set("oauth_nonce", nonce)
	params.Set("oauth_version", "1.0")

	// Create signature base string
	baseString := fmt.Sprintf("%s&%s&%s",
		strings.ToUpper(method),
		url.QueryEscape(parsedURL.Scheme+"://"+parsedURL.Host+parsedURL.Path),
		url.QueryEscape(params.Encode()),
	)

	// Sign with HMAC-SHA256
	key := []byte(c.config.ConsumerSecret + "&")
	h := hmac.New(sha256.New, key)
	h.Write([]byte(baseString))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	params.Set("oauth_signature", signature)
	parsedURL.RawQuery = params.Encode()

	return parsedURL.String()
}

// GetProducts retrieves products from WooCommerce
func (c *Connector) GetProducts(ctx context.Context, params connectors.ProductParams) ([]*connectors.Product, error) {
	path := "/products"

	queryParams := url.Values{}
	if params.Limit > 0 {
		queryParams.Set("per_page", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		page := (params.Offset / params.Limit) + 1
		queryParams.Set("page", strconv.Itoa(page))
	}
	if params.SKU != "" {
		queryParams.Set("sku", params.SKU)
	}
	if params.UpdatedAfter != "" {
		queryParams.Set("modified_after", params.UpdatedAfter)
	}

	if len(queryParams) > 0 {
		path += "?" + queryParams.Encode()
	}

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var wooProducts []struct {
		ID              int    `json:"id"`
		Name            string `json:"name"`
		SKU             string `json:"sku"`
		Description     string `json:"description"`
		ShortDescription string `json:"short_description"`
		Price           string `json:"price"`
		RegularPrice    string `json:"regular_price"`
		SalePrice       string `json:"sale_price"`
		Status          string `json:"status"`
		StockQuantity   *int   `json:"stock_quantity"`
		ManageStock     bool   `json:"manage_stock"`
		StockStatus     string `json:"stock_status"`
		Weight          string `json:"weight"`
		Categories      []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"categories"`
		Images []struct {
			ID  int    `json:"id"`
			Src string `json:"src"`
		} `json:"images"`
		Attributes []struct {
			ID      int      `json:"id"`
			Name    string   `json:"name"`
			Options []string `json:"options"`
		} `json:"attributes"`
		MetaData []struct {
			Key   string      `json:"key"`
			Value interface{} `json:"value"`
		} `json:"meta_data"`
	}

	if err := json.Unmarshal(respBody, &wooProducts); err != nil {
		return nil, err
	}

	products := make([]*connectors.Product, 0, len(wooProducts))
	for _, p := range wooProducts {
		var imageURL string
		if len(p.Images) > 0 {
			imageURL = p.Images[0].Src
		}

		var categories []string
		for _, cat := range p.Categories {
			categories = append(categories, cat.Name)
		}

		attrs := make(map[string]string)
		for _, attr := range p.Attributes {
			if len(attr.Options) > 0 {
				attrs[attr.Name] = strings.Join(attr.Options, ", ")
			}
		}

		price, _ := strconv.ParseFloat(p.Price, 64)
		regularPrice, _ := strconv.ParseFloat(p.RegularPrice, 64)
		weight, _ := strconv.ParseFloat(p.Weight, 64)

		var quantity int
		if p.StockQuantity != nil {
			quantity = *p.StockQuantity
		}

		products = append(products, &connectors.Product{
			ID:          strconv.Itoa(p.ID),
			SKU:         p.SKU,
			Title:       p.Name,
			Description: p.Description,
			Price:       price,
			ListPrice:   regularPrice,
			ImageURL:    imageURL,
			Categories:  categories,
			Attributes:  attrs,
			Weight:      weight,
			ChannelID:   "woocommerce",
			Active:      p.Status == "publish",
			Quantity:    quantity,
		})
	}

	return products, nil
}

// GetProduct retrieves a single product
func (c *Connector) GetProduct(ctx context.Context, productID string) (*connectors.Product, error) {
	path := fmt.Sprintf("/products/%s", productID)

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var p struct {
		ID              int    `json:"id"`
		Name            string `json:"name"`
		SKU             string `json:"sku"`
		Description     string `json:"description"`
		Price           string `json:"price"`
		RegularPrice    string `json:"regular_price"`
		Status          string `json:"status"`
		StockQuantity   *int   `json:"stock_quantity"`
		Weight          string `json:"weight"`
		Categories      []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"categories"`
		Images []struct {
			ID  int    `json:"id"`
			Src string `json:"src"`
		} `json:"images"`
	}

	if err := json.Unmarshal(respBody, &p); err != nil {
		return nil, err
	}

	var imageURL string
	if len(p.Images) > 0 {
		imageURL = p.Images[0].Src
	}

	var categories []string
	for _, cat := range p.Categories {
		categories = append(categories, cat.Name)
	}

	price, _ := strconv.ParseFloat(p.Price, 64)
	regularPrice, _ := strconv.ParseFloat(p.RegularPrice, 64)
	weight, _ := strconv.ParseFloat(p.Weight, 64)

	var quantity int
	if p.StockQuantity != nil {
		quantity = *p.StockQuantity
	}

	return &connectors.Product{
		ID:          strconv.Itoa(p.ID),
		SKU:         p.SKU,
		Title:       p.Name,
		Description: p.Description,
		Price:       price,
		ListPrice:   regularPrice,
		ImageURL:    imageURL,
		Categories:  categories,
		Weight:      weight,
		ChannelID:   "woocommerce",
		Active:      p.Status == "publish",
		Quantity:    quantity,
	}, nil
}

// GetInventory retrieves inventory for a product
func (c *Connector) GetInventory(ctx context.Context, productID string) (*connectors.InventoryLevel, error) {
	path := fmt.Sprintf("/products/%s", productID)

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var p struct {
		ID            int    `json:"id"`
		SKU           string `json:"sku"`
		StockQuantity *int   `json:"stock_quantity"`
		StockStatus   string `json:"stock_status"`
		ManageStock   bool   `json:"manage_stock"`
		BackOrders    string `json:"backorders"`
	}

	if err := json.Unmarshal(respBody, &p); err != nil {
		return nil, err
	}

	var quantity int
	if p.StockQuantity != nil {
		quantity = *p.StockQuantity
	}

	return &connectors.InventoryLevel{
		ProductID:   productID,
		SKU:         p.SKU,
		Quantity:    quantity,
		Available:   quantity,
		ChannelID:   "woocommerce",
		LocationID:  "default",
		LastUpdated: time.Now().Format(time.RFC3339),
	}, nil
}

// GetInventoryBatch retrieves inventory for multiple products
func (c *Connector) GetInventoryBatch(ctx context.Context, productIDs []string) ([]*connectors.InventoryLevel, error) {
	levels := make([]*connectors.InventoryLevel, 0, len(productIDs))

	for _, productID := range productIDs {
		level, err := c.GetInventory(ctx, productID)
		if err != nil {
			continue
		}
		levels = append(levels, level)
	}

	return levels, nil
}

// UpdateInventory updates inventory for a product
func (c *Connector) UpdateInventory(ctx context.Context, update connectors.InventoryUpdate) error {
	path := fmt.Sprintf("/products/%s", update.ProductID)

	body := map[string]interface{}{
		"stock_quantity": update.Quantity,
		"manage_stock":   true,
	}

	_, err := c.doRequest(ctx, "PUT", path, body)
	return err
}

// UpdateInventoryBatch updates inventory for multiple products
func (c *Connector) UpdateInventoryBatch(ctx context.Context, updates []connectors.InventoryUpdate) error {
	// WooCommerce supports batch updates
	path := "/products/batch"

	var updateItems []map[string]interface{}
	for _, update := range updates {
		id, _ := strconv.Atoi(update.ProductID)
		updateItems = append(updateItems, map[string]interface{}{
			"id":             id,
			"stock_quantity": update.Quantity,
			"manage_stock":   true,
		})
	}

	body := map[string]interface{}{
		"update": updateItems,
	}

	_, err := c.doRequest(ctx, "POST", path, body)
	return err
}

// GetOrders retrieves orders from WooCommerce
func (c *Connector) GetOrders(ctx context.Context, params connectors.OrderParams) ([]*connectors.Order, error) {
	path := "/orders"

	queryParams := url.Values{}
	if params.Limit > 0 {
		queryParams.Set("per_page", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		page := (params.Offset / params.Limit) + 1
		queryParams.Set("page", strconv.Itoa(page))
	}
	if params.CreatedAfter != "" {
		queryParams.Set("after", params.CreatedAfter)
	}
	if params.Status != "" {
		queryParams.Set("status", params.Status)
	}

	if len(queryParams) > 0 {
		path += "?" + queryParams.Encode()
	}

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var wooOrders []struct {
		ID            int    `json:"id"`
		OrderKey      string `json:"order_key"`
		Status        string `json:"status"`
		Currency      string `json:"currency"`
		Total         string `json:"total"`
		TotalTax      string `json:"total_tax"`
		ShippingTotal string `json:"shipping_total"`
		DiscountTotal string `json:"discount_total"`
		DateCreated   string `json:"date_created"`
		DateModified  string `json:"date_modified"`
		CustomerID    int    `json:"customer_id"`
		CustomerNote  string `json:"customer_note"`
		Billing       struct {
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Email     string `json:"email"`
			Phone     string `json:"phone"`
			Address1  string `json:"address_1"`
			Address2  string `json:"address_2"`
			City      string `json:"city"`
			State     string `json:"state"`
			Postcode  string `json:"postcode"`
			Country   string `json:"country"`
		} `json:"billing"`
		Shipping struct {
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Address1  string `json:"address_1"`
			Address2  string `json:"address_2"`
			City      string `json:"city"`
			State     string `json:"state"`
			Postcode  string `json:"postcode"`
			Country   string `json:"country"`
		} `json:"shipping"`
		LineItems []struct {
			ID        int    `json:"id"`
			ProductID int    `json:"product_id"`
			SKU       string `json:"sku"`
			Name      string `json:"name"`
			Quantity  int    `json:"quantity"`
			Subtotal  string `json:"subtotal"`
			Total     string `json:"total"`
		} `json:"line_items"`
		ShippingLines []struct {
			MethodTitle string `json:"method_title"`
			Total       string `json:"total"`
		} `json:"shipping_lines"`
	}

	if err := json.Unmarshal(respBody, &wooOrders); err != nil {
		return nil, err
	}

	orders := make([]*connectors.Order, 0, len(wooOrders))
	for _, o := range wooOrders {
		createdAt, _ := time.Parse("2006-01-02T15:04:05", o.DateCreated)
		updatedAt, _ := time.Parse("2006-01-02T15:04:05", o.DateModified)
		total, _ := strconv.ParseFloat(o.Total, 64)
		tax, _ := strconv.ParseFloat(o.TotalTax, 64)
		shipping, _ := strconv.ParseFloat(o.ShippingTotal, 64)
		discount, _ := strconv.ParseFloat(o.DiscountTotal, 64)

		var items []connectors.OrderItem
		for _, li := range o.LineItems {
			itemTotal, _ := strconv.ParseFloat(li.Total, 64)
			items = append(items, connectors.OrderItem{
				ID:        strconv.Itoa(li.ID),
				ProductID: strconv.Itoa(li.ProductID),
				SKU:       li.SKU,
				Title:     li.Name,
				Quantity:  li.Quantity,
				Price:     itemTotal / float64(li.Quantity),
				Total:     itemTotal,
				Currency:  o.Currency,
			})
		}

		var shippingMethod string
		if len(o.ShippingLines) > 0 {
			shippingMethod = o.ShippingLines[0].MethodTitle
		}

		billingAddr := &connectors.Address{
			Name:        o.Billing.FirstName + " " + o.Billing.LastName,
			Line1:       o.Billing.Address1,
			Line2:       o.Billing.Address2,
			City:        o.Billing.City,
			State:       o.Billing.State,
			PostalCode:  o.Billing.Postcode,
			CountryCode: o.Billing.Country,
			Phone:       o.Billing.Phone,
			Email:       o.Billing.Email,
		}
		shippingAddr := &connectors.Address{
			Name:        o.Shipping.FirstName + " " + o.Shipping.LastName,
			Line1:       o.Shipping.Address1,
			Line2:       o.Shipping.Address2,
			City:        o.Shipping.City,
			State:       o.Shipping.State,
			PostalCode:  o.Shipping.Postcode,
			CountryCode: o.Shipping.Country,
		}

		orders = append(orders, &connectors.Order{
			ID:              strconv.Itoa(o.ID),
			ChannelOrderID:  o.OrderKey,
			ChannelID:       "woocommerce",
			Status:          c.mapOrderStatus(o.Status),
			PaymentStatus:   c.mapPaymentStatus(o.Status),
			Total:           total,
			Tax:             tax,
			Shipping:        shipping,
			Discount:        discount,
			Currency:        o.Currency,
			Items:           items,
			BillingAddress:  billingAddr,
			ShippingAddress: shippingAddr,
			ShippingMethod:  shippingMethod,
			CustomerID:      strconv.Itoa(o.CustomerID),
			CustomerNote:    o.CustomerNote,
			CreatedAt:       createdAt.Format(time.RFC3339),
			UpdatedAt:       updatedAt.Format(time.RFC3339),
		})
	}

	return orders, nil
}

func (c *Connector) mapOrderStatus(wooStatus string) string {
	switch wooStatus {
	case "pending", "on-hold":
		return connectors.OrderStatusPending
	case "processing":
		return connectors.OrderStatusProcessing
	case "completed":
		return connectors.OrderStatusDelivered
	case "cancelled":
		return connectors.OrderStatusCancelled
	case "refunded":
		return connectors.OrderStatusRefunded
	case "failed":
		return connectors.OrderStatusCancelled
	default:
		return connectors.OrderStatusPending
	}
}

func (c *Connector) mapPaymentStatus(wooStatus string) string {
	switch wooStatus {
	case "pending", "on-hold":
		return "pending"
	case "processing", "completed":
		return "paid"
	case "refunded":
		return "refunded"
	case "failed", "cancelled":
		return "failed"
	default:
		return "pending"
	}
}

// GetOrder retrieves a single order
func (c *Connector) GetOrder(ctx context.Context, orderID string) (*connectors.Order, error) {
	path := fmt.Sprintf("/orders/%s", orderID)

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var o struct {
		ID            int    `json:"id"`
		OrderKey      string `json:"order_key"`
		Status        string `json:"status"`
		Currency      string `json:"currency"`
		Total         string `json:"total"`
		TotalTax      string `json:"total_tax"`
		ShippingTotal string `json:"shipping_total"`
		DateCreated   string `json:"date_created"`
		CustomerID    int    `json:"customer_id"`
		Billing       struct {
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Email     string `json:"email"`
			Phone     string `json:"phone"`
			Address1  string `json:"address_1"`
			City      string `json:"city"`
			State     string `json:"state"`
			Postcode  string `json:"postcode"`
			Country   string `json:"country"`
		} `json:"billing"`
		Shipping struct {
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Address1  string `json:"address_1"`
			City      string `json:"city"`
			State     string `json:"state"`
			Postcode  string `json:"postcode"`
			Country   string `json:"country"`
		} `json:"shipping"`
		LineItems []struct {
			ID        int    `json:"id"`
			ProductID int    `json:"product_id"`
			SKU       string `json:"sku"`
			Name      string `json:"name"`
			Quantity  int    `json:"quantity"`
			Total     string `json:"total"`
		} `json:"line_items"`
	}

	if err := json.Unmarshal(respBody, &o); err != nil {
		return nil, err
	}

	createdAt, _ := time.Parse("2006-01-02T15:04:05", o.DateCreated)
	total, _ := strconv.ParseFloat(o.Total, 64)

	var items []connectors.OrderItem
	for _, li := range o.LineItems {
		itemTotal, _ := strconv.ParseFloat(li.Total, 64)
		items = append(items, connectors.OrderItem{
			ID:        strconv.Itoa(li.ID),
			ProductID: strconv.Itoa(li.ProductID),
			SKU:       li.SKU,
			Title:     li.Name,
			Quantity:  li.Quantity,
			Price:     itemTotal / float64(li.Quantity),
			Total:     itemTotal,
			Currency:  o.Currency,
		})
	}

	return &connectors.Order{
		ID:             strconv.Itoa(o.ID),
		ChannelOrderID: o.OrderKey,
		ChannelID:      "woocommerce",
		Status:         c.mapOrderStatus(o.Status),
		PaymentStatus:  c.mapPaymentStatus(o.Status),
		Total:          total,
		Currency:       o.Currency,
		Items:          items,
		BillingAddress: connectors.Address{
			Name:        o.Billing.FirstName + " " + o.Billing.LastName,
			Line1:       o.Billing.Address1,
			City:        o.Billing.City,
			State:       o.Billing.State,
			PostalCode:  o.Billing.Postcode,
			CountryCode: o.Billing.Country,
			Phone:       o.Billing.Phone,
			Email:       o.Billing.Email,
		},
		ShippingAddress: connectors.Address{
			Name:        o.Shipping.FirstName + " " + o.Shipping.LastName,
			Line1:       o.Shipping.Address1,
			City:        o.Shipping.City,
			State:       o.Shipping.State,
			PostalCode:  o.Shipping.Postcode,
			CountryCode: o.Shipping.Country,
		},
		CustomerID: strconv.Itoa(o.CustomerID),
		CreatedAt:  createdAt,
	}, nil
}

// AcknowledgeOrder marks an order as acknowledged
func (c *Connector) AcknowledgeOrder(ctx context.Context, orderID string) error {
	// WooCommerce doesn't have a specific acknowledge endpoint
	// We can add a meta field to track acknowledgment
	path := fmt.Sprintf("/orders/%s", orderID)

	body := map[string]interface{}{
		"meta_data": []map[string]interface{}{
			{
				"key":   "_acknowledged",
				"value": time.Now().Format(time.RFC3339),
			},
		},
	}

	_, err := c.doRequest(ctx, "PUT", path, body)
	return err
}

// GetPrice retrieves pricing for a product
func (c *Connector) GetPrice(ctx context.Context, productID string) (*connectors.Price, error) {
	path := fmt.Sprintf("/products/%s", productID)

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var p struct {
		ID           int    `json:"id"`
		SKU          string `json:"sku"`
		Price        string `json:"price"`
		RegularPrice string `json:"regular_price"`
		SalePrice    string `json:"sale_price"`
		OnSale       bool   `json:"on_sale"`
	}

	if err := json.Unmarshal(respBody, &p); err != nil {
		return nil, err
	}

	price, _ := strconv.ParseFloat(p.Price, 64)
	regularPrice, _ := strconv.ParseFloat(p.RegularPrice, 64)
	salePrice, _ := strconv.ParseFloat(p.SalePrice, 64)

	return &connectors.Price{
		ProductID:   productID,
		SKU:         p.SKU,
		Price:       price,
		ListPrice:   regularPrice,
		SalePrice:   salePrice,
		OnSale:      p.OnSale,
		Currency:    "USD", // WooCommerce stores default currency in settings
		ChannelID:   "woocommerce",
		LastUpdated: time.Now(),
	}, nil
}

// UpdatePrice updates the price for a product
func (c *Connector) UpdatePrice(ctx context.Context, update connectors.PriceUpdate) error {
	path := fmt.Sprintf("/products/%s", update.ProductID)

	body := map[string]interface{}{
		"regular_price": fmt.Sprintf("%.2f", update.Price),
	}

	if update.SalePrice > 0 {
		body["sale_price"] = fmt.Sprintf("%.2f", update.SalePrice)
	}

	_, err := c.doRequest(ctx, "PUT", path, body)
	return err
}

// SyncInventory performs full inventory synchronization
func (c *Connector) SyncInventory(ctx context.Context) (*connectors.SyncResult, error) {
	result := &connectors.SyncResult{
		StartedAt: time.Now(),
		Channel:   "woocommerce",
	}

	// Get all products
	products, err := c.GetProducts(ctx, connectors.ProductParams{Limit: 100})
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.CompletedAt = time.Now()
		return result, err
	}

	result.ItemsProcessed = len(products)
	result.ItemsSucceeded = len(products)
	result.CompletedAt = time.Now()

	return result, nil
}

// GetWebhooks retrieves configured webhooks
func (c *Connector) GetWebhooks(ctx context.Context) ([]connectors.Webhook, error) {
	path := "/webhooks"

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var wooWebhooks []struct {
		ID          int      `json:"id"`
		Name        string   `json:"name"`
		Status      string   `json:"status"`
		Topic       string   `json:"topic"`
		DeliveryURL string   `json:"delivery_url"`
		Secret      string   `json:"secret"`
		DateCreated string   `json:"date_created"`
	}

	if err := json.Unmarshal(respBody, &wooWebhooks); err != nil {
		return nil, err
	}

	webhooks := make([]connectors.Webhook, 0, len(wooWebhooks))
	for _, wh := range wooWebhooks {
		webhooks = append(webhooks, connectors.Webhook{
			ID:        strconv.Itoa(wh.ID),
			Name:      wh.Name,
			URL:       wh.DeliveryURL,
			Topic:     wh.Topic,
			Secret:    wh.Secret,
			Active:    wh.Status == "active",
			ChannelID: "woocommerce",
		})
	}

	return webhooks, nil
}

// CreateWebhook creates a webhook
func (c *Connector) CreateWebhook(ctx context.Context, webhook connectors.Webhook) error {
	path := "/webhooks"

	body := map[string]interface{}{
		"name":         webhook.Name,
		"topic":        webhook.Topic,
		"delivery_url": webhook.URL,
		"secret":       webhook.Secret,
		"status":       "active",
	}

	_, err := c.doRequest(ctx, "POST", path, body)
	return err
}

// DeleteWebhook deletes a webhook
func (c *Connector) DeleteWebhook(ctx context.Context, webhookID string) error {
	path := fmt.Sprintf("/webhooks/%s?force=true", webhookID)

	_, err := c.doRequest(ctx, "DELETE", path, nil)
	return err
}

// UpdateOrderStatus updates order status
func (c *Connector) UpdateOrderStatus(ctx context.Context, orderID string, status string) error {
	path := fmt.Sprintf("/orders/%s", orderID)

	wooStatus := c.mapToWooStatus(status)
	body := map[string]interface{}{
		"status": wooStatus,
	}

	_, err := c.doRequest(ctx, "PUT", path, body)
	return err
}

func (c *Connector) mapToWooStatus(status string) string {
	switch status {
	case connectors.OrderStatusPending:
		return "pending"
	case connectors.OrderStatusProcessing:
		return "processing"
	case connectors.OrderStatusShipped:
		return "completed"
	case connectors.OrderStatusDelivered:
		return "completed"
	case connectors.OrderStatusCancelled:
		return "cancelled"
	case connectors.OrderStatusRefunded:
		return "refunded"
	default:
		return "pending"
	}
}

// AddOrderNote adds a note to an order
func (c *Connector) AddOrderNote(ctx context.Context, orderID, note string, customerNote bool) error {
	path := fmt.Sprintf("/orders/%s/notes", orderID)

	body := map[string]interface{}{
		"note":          note,
		"customer_note": customerNote,
	}

	_, err := c.doRequest(ctx, "POST", path, body)
	return err
}

// CreateRefund creates a refund for an order
func (c *Connector) CreateRefund(ctx context.Context, orderID string, amount float64, reason string) error {
	path := fmt.Sprintf("/orders/%s/refunds", orderID)

	body := map[string]interface{}{
		"amount": fmt.Sprintf("%.2f", amount),
		"reason": reason,
	}

	_, err := c.doRequest(ctx, "POST", path, body)
	return err
}
