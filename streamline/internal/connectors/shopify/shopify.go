package shopify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/savegress/streamline/internal/connectors"
	"github.com/savegress/streamline/pkg/models"
	"github.com/shopspring/decimal"
)

// Config holds Shopify connector configuration
type Config struct {
	connectors.ConnectorConfig
	ShopDomain  string
	AccessToken string
	APIVersion  string
}

// Connector implements the Shopify connector
type Connector struct {
	connectors.BaseConnector
	config     Config
	httpClient *http.Client
	baseURL    string
}

// NewConnector creates a new Shopify connector
func NewConnector(config Config) *Connector {
	if config.APIVersion == "" {
		config.APIVersion = "2024-01"
	}

	c := &Connector{
		BaseConnector: connectors.BaseConnector{
			ID:   config.ID,
			Type: models.ChannelTypeShopify,
			Name: config.Name,
		},
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: fmt.Sprintf("https://%s/admin/api/%s", config.ShopDomain, config.APIVersion),
	}

	return c
}

// Connect establishes connection to Shopify
func (c *Connector) Connect(ctx context.Context) error {
	// Test the connection
	if err := c.TestConnection(ctx); err != nil {
		return err
	}
	c.Connected = true
	return nil
}

// Disconnect disconnects from Shopify
func (c *Connector) Disconnect(ctx context.Context) error {
	c.Connected = false
	return nil
}

// TestConnection tests the Shopify connection
func (c *Connector) TestConnection(ctx context.Context) error {
	_, err := c.doRequest(ctx, "GET", "/shop.json", nil)
	return err
}

// GetProducts retrieves products from Shopify
func (c *Connector) GetProducts(ctx context.Context, limit, offset int) ([]*models.Product, error) {
	if limit == 0 {
		limit = 50
	}

	url := fmt.Sprintf("/products.json?limit=%d", limit)
	if offset > 0 {
		// Shopify uses page_info for pagination, simplified here
		url += fmt.Sprintf("&page=%d", offset/limit+1)
	}

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Products []shopifyProduct `json:"products"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	products := make([]*models.Product, len(result.Products))
	for i, sp := range result.Products {
		products[i] = c.convertToProduct(sp)
	}

	return products, nil
}

// GetProduct retrieves a single product
func (c *Connector) GetProduct(ctx context.Context, externalID string) (*models.Product, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/products/%s.json", externalID), nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Product shopifyProduct `json:"product"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return c.convertToProduct(result.Product), nil
}

// CreateProduct creates a product in Shopify
func (c *Connector) CreateProduct(ctx context.Context, product *models.Product) (string, error) {
	sp := c.convertFromProduct(product)
	body := map[string]interface{}{"product": sp}

	resp, err := c.doRequest(ctx, "POST", "/products.json", body)
	if err != nil {
		return "", err
	}

	var result struct {
		Product struct {
			ID int64 `json:"id"`
		} `json:"product"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}

	return strconv.FormatInt(result.Product.ID, 10), nil
}

// UpdateProduct updates a product in Shopify
func (c *Connector) UpdateProduct(ctx context.Context, externalID string, product *models.Product) error {
	sp := c.convertFromProduct(product)
	body := map[string]interface{}{"product": sp}

	_, err := c.doRequest(ctx, "PUT", fmt.Sprintf("/products/%s.json", externalID), body)
	return err
}

// DeleteProduct deletes a product from Shopify
func (c *Connector) DeleteProduct(ctx context.Context, externalID string) error {
	_, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/products/%s.json", externalID), nil)
	return err
}

// GetInventory retrieves inventory for a SKU
func (c *Connector) GetInventory(ctx context.Context, sku string) (int, error) {
	// First, find the inventory item ID for this SKU
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/variants.json?sku=%s", sku), nil)
	if err != nil {
		return 0, err
	}

	var result struct {
		Variants []struct {
			ID              int64 `json:"id"`
			InventoryItemID int64 `json:"inventory_item_id"`
			InventoryQty    int   `json:"inventory_quantity"`
		} `json:"variants"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, err
	}

	if len(result.Variants) == 0 {
		return 0, connectors.ErrProductNotFound
	}

	return result.Variants[0].InventoryQty, nil
}

// UpdateInventory updates inventory for a SKU
func (c *Connector) UpdateInventory(ctx context.Context, sku string, quantity int) error {
	// Find the inventory item ID
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/variants.json?sku=%s", sku), nil)
	if err != nil {
		return err
	}

	var variantResult struct {
		Variants []struct {
			InventoryItemID int64 `json:"inventory_item_id"`
		} `json:"variants"`
	}
	if err := json.Unmarshal(resp, &variantResult); err != nil {
		return err
	}

	if len(variantResult.Variants) == 0 {
		return connectors.ErrProductNotFound
	}

	// Get location ID (simplified - using first location)
	locResp, err := c.doRequest(ctx, "GET", "/locations.json", nil)
	if err != nil {
		return err
	}

	var locResult struct {
		Locations []struct {
			ID int64 `json:"id"`
		} `json:"locations"`
	}
	if err := json.Unmarshal(locResp, &locResult); err != nil {
		return err
	}

	if len(locResult.Locations) == 0 {
		return fmt.Errorf("no locations found")
	}

	// Set inventory level
	body := map[string]interface{}{
		"inventory_item_id": variantResult.Variants[0].InventoryItemID,
		"location_id":       locResult.Locations[0].ID,
		"available":         quantity,
	}

	_, err = c.doRequest(ctx, "POST", "/inventory_levels/set.json", body)
	return err
}

// GetInventoryBatch retrieves inventory for multiple SKUs
func (c *Connector) GetInventoryBatch(ctx context.Context, skus []string) (map[string]int, error) {
	result := make(map[string]int)
	for _, sku := range skus {
		qty, err := c.GetInventory(ctx, sku)
		if err != nil {
			continue
		}
		result[sku] = qty
	}
	return result, nil
}

// UpdateInventoryBatch updates inventory for multiple SKUs
func (c *Connector) UpdateInventoryBatch(ctx context.Context, updates map[string]int) error {
	for sku, qty := range updates {
		if err := c.UpdateInventory(ctx, sku, qty); err != nil {
			return err
		}
	}
	return nil
}

// GetOrders retrieves orders from Shopify
func (c *Connector) GetOrders(ctx context.Context, since *string, limit int) ([]*models.Order, error) {
	if limit == 0 {
		limit = 50
	}

	url := fmt.Sprintf("/orders.json?status=any&limit=%d", limit)
	if since != nil {
		url += fmt.Sprintf("&created_at_min=%s", *since)
	}

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Orders []shopifyOrder `json:"orders"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	orders := make([]*models.Order, len(result.Orders))
	for i, so := range result.Orders {
		orders[i] = c.convertToOrder(so)
	}

	return orders, nil
}

// GetOrder retrieves a single order
func (c *Connector) GetOrder(ctx context.Context, externalID string) (*models.Order, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/orders/%s.json", externalID), nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Order shopifyOrder `json:"order"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return c.convertToOrder(result.Order), nil
}

// UpdateOrderStatus updates an order's status
func (c *Connector) UpdateOrderStatus(ctx context.Context, externalID string, status models.OrderStatus) error {
	// Shopify uses fulfillments for status updates
	// This is simplified
	return nil
}

// FulfillOrder creates a fulfillment for an order
func (c *Connector) FulfillOrder(ctx context.Context, externalID string, fulfillment *models.Fulfillment) error {
	body := map[string]interface{}{
		"fulfillment": map[string]interface{}{
			"tracking_number":  fulfillment.TrackingNum,
			"tracking_company": fulfillment.Carrier,
			"notify_customer":  true,
		},
	}

	_, err := c.doRequest(ctx, "POST", fmt.Sprintf("/orders/%s/fulfillments.json", externalID), body)
	return err
}

// CancelOrder cancels an order
func (c *Connector) CancelOrder(ctx context.Context, externalID string, reason string) error {
	body := map[string]interface{}{
		"reason": reason,
	}

	_, err := c.doRequest(ctx, "POST", fmt.Sprintf("/orders/%s/cancel.json", externalID), body)
	return err
}

// GetPrice retrieves the price for a SKU
func (c *Connector) GetPrice(ctx context.Context, sku string) (decimal.Decimal, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/variants.json?sku=%s", sku), nil)
	if err != nil {
		return decimal.Zero, err
	}

	var result struct {
		Variants []struct {
			Price string `json:"price"`
		} `json:"variants"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return decimal.Zero, err
	}

	if len(result.Variants) == 0 {
		return decimal.Zero, connectors.ErrProductNotFound
	}

	return decimal.NewFromString(result.Variants[0].Price)
}

// UpdatePrice updates the price for a SKU
func (c *Connector) UpdatePrice(ctx context.Context, sku string, price decimal.Decimal) error {
	// Find variant ID
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/variants.json?sku=%s", sku), nil)
	if err != nil {
		return err
	}

	var result struct {
		Variants []struct {
			ID int64 `json:"id"`
		} `json:"variants"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return err
	}

	if len(result.Variants) == 0 {
		return connectors.ErrProductNotFound
	}

	body := map[string]interface{}{
		"variant": map[string]interface{}{
			"id":    result.Variants[0].ID,
			"price": price.String(),
		},
	}

	_, err = c.doRequest(ctx, "PUT", fmt.Sprintf("/variants/%d.json", result.Variants[0].ID), body)
	return err
}

// UpdatePriceBatch updates prices for multiple SKUs
func (c *Connector) UpdatePriceBatch(ctx context.Context, prices map[string]decimal.Decimal) error {
	for sku, price := range prices {
		if err := c.UpdatePrice(ctx, sku, price); err != nil {
			return err
		}
	}
	return nil
}

// RegisterWebhook registers a webhook
func (c *Connector) RegisterWebhook(ctx context.Context, topic, url string) error {
	body := map[string]interface{}{
		"webhook": map[string]interface{}{
			"topic":   topic,
			"address": url,
			"format":  "json",
		},
	}

	_, err := c.doRequest(ctx, "POST", "/webhooks.json", body)
	return err
}

// UnregisterWebhook unregisters a webhook
func (c *Connector) UnregisterWebhook(ctx context.Context, webhookID string) error {
	_, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/webhooks/%s.json", webhookID), nil)
	return err
}

// HTTP helper
func (c *Connector) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Shopify-Access-Token", c.config.AccessToken)

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
		return nil, &connectors.ConnectorError{
			Message: fmt.Sprintf("Shopify API error: %d", resp.StatusCode),
		}
	}

	return respBody, nil
}

// Shopify data types
type shopifyProduct struct {
	ID          int64             `json:"id,omitempty"`
	Title       string            `json:"title"`
	BodyHTML    string            `json:"body_html"`
	Vendor      string            `json:"vendor"`
	ProductType string            `json:"product_type"`
	Status      string            `json:"status"`
	Variants    []shopifyVariant  `json:"variants"`
	Images      []shopifyImage    `json:"images"`
	Tags        string            `json:"tags"`
}

type shopifyVariant struct {
	ID                int64  `json:"id,omitempty"`
	SKU               string `json:"sku"`
	Price             string `json:"price"`
	CompareAtPrice    string `json:"compare_at_price"`
	InventoryQuantity int    `json:"inventory_quantity"`
	Weight            float64 `json:"weight"`
	WeightUnit        string `json:"weight_unit"`
}

type shopifyImage struct {
	Src string `json:"src"`
}

type shopifyOrder struct {
	ID                int64              `json:"id"`
	Name              string             `json:"name"`
	Email             string             `json:"email"`
	FinancialStatus   string             `json:"financial_status"`
	FulfillmentStatus string             `json:"fulfillment_status"`
	TotalPrice        string             `json:"total_price"`
	SubtotalPrice     string             `json:"subtotal_price"`
	TotalTax          string             `json:"total_tax"`
	TotalDiscounts    string             `json:"total_discounts"`
	Currency          string             `json:"currency"`
	LineItems         []shopifyLineItem  `json:"line_items"`
	Customer          shopifyCustomer    `json:"customer"`
	ShippingAddress   shopifyAddress     `json:"shipping_address"`
	BillingAddress    shopifyAddress     `json:"billing_address"`
	CreatedAt         time.Time          `json:"created_at"`
}

type shopifyLineItem struct {
	ID        int64   `json:"id"`
	SKU       string  `json:"sku"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     string  `json:"price"`
}

type shopifyCustomer struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
}

type shopifyAddress struct {
	Address1 string `json:"address1"`
	Address2 string `json:"address2"`
	City     string `json:"city"`
	Province string `json:"province"`
	Zip      string `json:"zip"`
	Country  string `json:"country"`
}

// Conversion functions
func (c *Connector) convertToProduct(sp shopifyProduct) *models.Product {
	product := &models.Product{
		ID:          strconv.FormatInt(sp.ID, 10),
		Name:        sp.Title,
		Description: sp.BodyHTML,
		Status:      models.ProductStatusActive,
	}

	if sp.Status == "draft" {
		product.Status = models.ProductStatusDraft
	} else if sp.Status == "archived" {
		product.Status = models.ProductStatusInactive
	}

	if len(sp.Variants) > 0 {
		product.SKU = sp.Variants[0].SKU
		product.BasePrice, _ = decimal.NewFromString(sp.Variants[0].Price)
	}

	for _, img := range sp.Images {
		product.Images = append(product.Images, img.Src)
	}

	return product
}

func (c *Connector) convertFromProduct(product *models.Product) map[string]interface{} {
	return map[string]interface{}{
		"title":     product.Name,
		"body_html": product.Description,
		"variants": []map[string]interface{}{
			{
				"sku":   product.SKU,
				"price": product.BasePrice.String(),
			},
		},
	}
}

func (c *Connector) convertToOrder(so shopifyOrder) *models.Order {
	order := &models.Order{
		ID:         strconv.FormatInt(so.ID, 10),
		Channel:    "shopify",
		ChannelRef: so.Name,
		CreatedAt:  so.CreatedAt,
		UpdatedAt:  so.CreatedAt,
	}

	// Status mapping
	switch so.FulfillmentStatus {
	case "fulfilled":
		order.Status = models.OrderStatusShipped
	case "partial":
		order.Status = models.OrderStatusProcessing
	default:
		if so.FinancialStatus == "paid" {
			order.Status = models.OrderStatusProcessing
		} else {
			order.Status = models.OrderStatusPending
		}
	}

	// Customer
	order.Customer = models.Customer{
		ID:        strconv.FormatInt(so.Customer.ID, 10),
		Email:     so.Email,
		FirstName: so.Customer.FirstName,
		LastName:  so.Customer.LastName,
		Phone:     so.Customer.Phone,
	}

	// Items
	for _, li := range so.LineItems {
		price, _ := decimal.NewFromString(li.Price)
		order.Items = append(order.Items, models.OrderItem{
			ID:       strconv.FormatInt(li.ID, 10),
			SKU:      li.SKU,
			Name:     li.Name,
			Quantity: li.Quantity,
			Price:    price,
			Total:    price.Mul(decimal.NewFromInt(int64(li.Quantity))),
		})
	}

	// Shipping address
	order.Shipping = models.ShippingInfo{
		Address: models.Address{
			Line1:      so.ShippingAddress.Address1,
			Line2:      so.ShippingAddress.Address2,
			City:       so.ShippingAddress.City,
			State:      so.ShippingAddress.Province,
			PostalCode: so.ShippingAddress.Zip,
			Country:    so.ShippingAddress.Country,
		},
	}

	// Totals
	subtotal, _ := decimal.NewFromString(so.SubtotalPrice)
	tax, _ := decimal.NewFromString(so.TotalTax)
	discount, _ := decimal.NewFromString(so.TotalDiscounts)
	total, _ := decimal.NewFromString(so.TotalPrice)

	order.Totals = models.OrderTotals{
		Subtotal: subtotal,
		Tax:      tax,
		Discount: discount,
		Total:    total,
		Currency: so.Currency,
	}

	return order
}
