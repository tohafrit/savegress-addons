package ebay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/streamline/streamline/internal/connectors"
)

// Config holds eBay API configuration
type Config struct {
	connectors.ConnectorConfig
	ClientID       string // App ID
	ClientSecret   string // Cert ID
	RefreshToken   string
	MarketplaceID  string // EBAY_US, EBAY_UK, EBAY_DE, etc.
	Environment    string // PRODUCTION or SANDBOX
	FulfillmentProgramID string
}

// Connector implements the eBay connector
type Connector struct {
	config      Config
	httpClient  *http.Client
	accessToken string
	tokenExpiry time.Time
	mu          sync.RWMutex
}

// NewConnector creates a new eBay connector
func NewConnector(config Config) *Connector {
	if config.Environment == "" {
		config.Environment = "PRODUCTION"
	}
	if config.MarketplaceID == "" {
		config.MarketplaceID = "EBAY_US"
	}

	return &Connector{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Type returns the connector type
func (c *Connector) Type() string {
	return "ebay"
}

// Name returns the connector name
func (c *Connector) Name() string {
	return c.config.Name
}

// Connect establishes connection to eBay API
func (c *Connector) Connect(ctx context.Context) error {
	return c.refreshAccessToken(ctx)
}

// Disconnect closes the connection
func (c *Connector) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accessToken = ""
	c.tokenExpiry = time.Time{}
	return nil
}

// IsConnected checks if connected
func (c *Connector) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken != "" && time.Now().Before(c.tokenExpiry)
}

func (c *Connector) getBaseURL() string {
	if c.config.Environment == "SANDBOX" {
		return "https://api.sandbox.ebay.com"
	}
	return "https://api.ebay.com"
}

func (c *Connector) getAuthURL() string {
	if c.config.Environment == "SANDBOX" {
		return "https://api.sandbox.ebay.com/identity/v1/oauth2/token"
	}
	return "https://api.ebay.com/identity/v1/oauth2/token"
}

func (c *Connector) refreshAccessToken(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", c.config.RefreshToken)
	data.Set("scope", "https://api.ebay.com/oauth/api_scope https://api.ebay.com/oauth/api_scope/sell.inventory https://api.ebay.com/oauth/api_scope/sell.fulfillment https://api.ebay.com/oauth/api_scope/sell.analytics.readonly")

	req, err := http.NewRequestWithContext(ctx, "POST", c.getAuthURL(), strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.config.ClientID, c.config.ClientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	return nil
}

func (c *Connector) ensureValidToken(ctx context.Context) error {
	c.mu.RLock()
	valid := c.accessToken != "" && time.Now().Before(c.tokenExpiry)
	c.mu.RUnlock()

	if !valid {
		return c.refreshAccessToken(ctx)
	}
	return nil
}

func (c *Connector) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	if err := c.ensureValidToken(ctx); err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	url := c.getBaseURL() + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	token := c.accessToken
	c.mu.RUnlock()

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-EBAY-C-MARKETPLACE-ID", c.config.MarketplaceID)

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
		return nil, fmt.Errorf("eBay API error: %s - %s", resp.Status, string(respBody))
	}

	return respBody, nil
}

// GetProducts retrieves products from eBay
func (c *Connector) GetProducts(ctx context.Context, params connectors.ProductParams) ([]*connectors.Product, error) {
	// Use Inventory API to get inventory items
	path := "/sell/inventory/v1/inventory_item"
	if params.Limit > 0 {
		path += fmt.Sprintf("?limit=%d", params.Limit)
	}
	if params.Offset > 0 {
		path += fmt.Sprintf("&offset=%d", params.Offset)
	}

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		InventoryItems []struct {
			SKU     string `json:"sku"`
			Product struct {
				Title       string   `json:"title"`
				Description string   `json:"description"`
				Brand       string   `json:"brand"`
				ImageURLs   []string `json:"imageUrls"`
				EAN         []string `json:"ean"`
				UPC         []string `json:"upc"`
				ISBN        []string `json:"isbn"`
			} `json:"product"`
			Condition    string `json:"condition"`
			Availability struct {
				ShipToLocationAvailability struct {
					Quantity int `json:"quantity"`
				} `json:"shipToLocationAvailability"`
			} `json:"availability"`
		} `json:"inventoryItems"`
		Total int `json:"total"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, err
	}

	products := make([]*connectors.Product, 0, len(response.InventoryItems))
	for _, item := range response.InventoryItems {
		var imageURL string
		if len(item.Product.ImageURLs) > 0 {
			imageURL = item.Product.ImageURLs[0]
		}

		var identifiers []connectors.ProductIdentifier
		for _, ean := range item.Product.EAN {
			identifiers = append(identifiers, connectors.ProductIdentifier{Type: "EAN", Value: ean})
		}
		for _, upc := range item.Product.UPC {
			identifiers = append(identifiers, connectors.ProductIdentifier{Type: "UPC", Value: upc})
		}
		for _, isbn := range item.Product.ISBN {
			identifiers = append(identifiers, connectors.ProductIdentifier{Type: "ISBN", Value: isbn})
		}

		products = append(products, &connectors.Product{
			ID:          item.SKU,
			SKU:         item.SKU,
			Title:       item.Product.Title,
			Description: item.Product.Description,
			Brand:       item.Product.Brand,
			ImageURL:    imageURL,
			Identifiers: identifiers,
			Attributes: map[string]string{
				"condition": item.Condition,
			},
			ChannelID: "ebay",
			Active:    true,
		})
	}

	return products, nil
}

// GetProduct retrieves a single product by SKU
func (c *Connector) GetProduct(ctx context.Context, productID string) (*connectors.Product, error) {
	path := fmt.Sprintf("/sell/inventory/v1/inventory_item/%s", url.PathEscape(productID))

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var item struct {
		SKU     string `json:"sku"`
		Product struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Brand       string   `json:"brand"`
			ImageURLs   []string `json:"imageUrls"`
			EAN         []string `json:"ean"`
			UPC         []string `json:"upc"`
		} `json:"product"`
		Condition    string `json:"condition"`
		Availability struct {
			ShipToLocationAvailability struct {
				Quantity int `json:"quantity"`
			} `json:"shipToLocationAvailability"`
		} `json:"availability"`
	}

	if err := json.Unmarshal(respBody, &item); err != nil {
		return nil, err
	}

	var imageURL string
	if len(item.Product.ImageURLs) > 0 {
		imageURL = item.Product.ImageURLs[0]
	}

	var identifiers []connectors.ProductIdentifier
	for _, ean := range item.Product.EAN {
		identifiers = append(identifiers, connectors.ProductIdentifier{Type: "EAN", Value: ean})
	}
	for _, upc := range item.Product.UPC {
		identifiers = append(identifiers, connectors.ProductIdentifier{Type: "UPC", Value: upc})
	}

	return &connectors.Product{
		ID:          item.SKU,
		SKU:         item.SKU,
		Title:       item.Product.Title,
		Description: item.Product.Description,
		Brand:       item.Product.Brand,
		ImageURL:    imageURL,
		Identifiers: identifiers,
		Attributes: map[string]string{
			"condition": item.Condition,
		},
		ChannelID: "ebay",
		Active:    true,
	}, nil
}

// GetInventory retrieves inventory for a product
func (c *Connector) GetInventory(ctx context.Context, productID string) (*connectors.InventoryLevel, error) {
	path := fmt.Sprintf("/sell/inventory/v1/inventory_item/%s", url.PathEscape(productID))

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var item struct {
		SKU          string `json:"sku"`
		Availability struct {
			ShipToLocationAvailability struct {
				Quantity int `json:"quantity"`
			} `json:"shipToLocationAvailability"`
		} `json:"availability"`
	}

	if err := json.Unmarshal(respBody, &item); err != nil {
		return nil, err
	}

	return &connectors.InventoryLevel{
		ProductID:   productID,
		SKU:         item.SKU,
		Quantity:    item.Availability.ShipToLocationAvailability.Quantity,
		Available:   item.Availability.ShipToLocationAvailability.Quantity,
		ChannelID:   "ebay",
		LocationID:  c.config.MarketplaceID,
		LastUpdated: time.Now(),
	}, nil
}

// GetInventoryBatch retrieves inventory for multiple products
func (c *Connector) GetInventoryBatch(ctx context.Context, productIDs []string) ([]*connectors.InventoryLevel, error) {
	levels := make([]*connectors.InventoryLevel, 0, len(productIDs))

	for _, productID := range productIDs {
		level, err := c.GetInventory(ctx, productID)
		if err != nil {
			continue // Skip errors for individual items
		}
		levels = append(levels, level)
	}

	return levels, nil
}

// UpdateInventory updates inventory for a product
func (c *Connector) UpdateInventory(ctx context.Context, update connectors.InventoryUpdate) error {
	// First get the existing item
	path := fmt.Sprintf("/sell/inventory/v1/inventory_item/%s", url.PathEscape(update.ProductID))

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return err
	}

	var item map[string]interface{}
	if err := json.Unmarshal(respBody, &item); err != nil {
		return err
	}

	// Update the quantity
	availability := item["availability"].(map[string]interface{})
	shipToLocation := availability["shipToLocationAvailability"].(map[string]interface{})
	shipToLocation["quantity"] = update.Quantity

	// PUT the updated item back
	_, err = c.doRequest(ctx, "PUT", path, item)
	return err
}

// UpdateInventoryBatch updates inventory for multiple products
func (c *Connector) UpdateInventoryBatch(ctx context.Context, updates []connectors.InventoryUpdate) error {
	var lastErr error
	for _, update := range updates {
		if err := c.UpdateInventory(ctx, update); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// GetOrders retrieves orders from eBay
func (c *Connector) GetOrders(ctx context.Context, params connectors.OrderParams) ([]*connectors.Order, error) {
	path := "/sell/fulfillment/v1/order"

	queryParams := url.Values{}
	if !params.CreatedAfter.IsZero() {
		queryParams.Set("filter", fmt.Sprintf("creationdate:[%s..]", params.CreatedAfter.Format(time.RFC3339)))
	}
	if params.Limit > 0 {
		queryParams.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		queryParams.Set("offset", strconv.Itoa(params.Offset))
	}

	if len(queryParams) > 0 {
		path += "?" + queryParams.Encode()
	}

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Orders []struct {
			OrderID           string `json:"orderId"`
			CreationDate      string `json:"creationDate"`
			OrderFulfillmentStatus string `json:"orderFulfillmentStatus"`
			OrderPaymentStatus     string `json:"orderPaymentStatus"`
			PricingSummary    struct {
				Total struct {
					Value    string `json:"value"`
					Currency string `json:"currency"`
				} `json:"total"`
			} `json:"pricingSummary"`
			Buyer struct {
				Username string `json:"username"`
			} `json:"buyer"`
			LineItems []struct {
				LineItemID string `json:"lineItemId"`
				SKU        string `json:"sku"`
				Title      string `json:"title"`
				Quantity   int    `json:"quantity"`
				LineItemCost struct {
					Value    string `json:"value"`
					Currency string `json:"currency"`
				} `json:"lineItemCost"`
			} `json:"lineItems"`
			FulfillmentStartInstructions []struct {
				ShippingStep struct {
					ShipTo struct {
						FullName       string `json:"fullName"`
						ContactAddress struct {
							AddressLine1 string `json:"addressLine1"`
							AddressLine2 string `json:"addressLine2"`
							City         string `json:"city"`
							StateOrProvince string `json:"stateOrProvince"`
							PostalCode   string `json:"postalCode"`
							CountryCode  string `json:"countryCode"`
						} `json:"contactAddress"`
						PrimaryPhone struct {
							PhoneNumber string `json:"phoneNumber"`
						} `json:"primaryPhone"`
						Email string `json:"email"`
					} `json:"shipTo"`
				} `json:"shippingStep"`
			} `json:"fulfillmentStartInstructions"`
		} `json:"orders"`
		Total int `json:"total"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, err
	}

	orders := make([]*connectors.Order, 0, len(response.Orders))
	for _, o := range response.Orders {
		createdAt, _ := time.Parse(time.RFC3339, o.CreationDate)

		total, _ := strconv.ParseFloat(o.PricingSummary.Total.Value, 64)

		var items []connectors.OrderItem
		for _, li := range o.LineItems {
			price, _ := strconv.ParseFloat(li.LineItemCost.Value, 64)
			items = append(items, connectors.OrderItem{
				ID:        li.LineItemID,
				ProductID: li.SKU,
				SKU:       li.SKU,
				Title:     li.Title,
				Quantity:  li.Quantity,
				Price:     price,
				Currency:  li.LineItemCost.Currency,
			})
		}

		status := c.mapOrderStatus(o.OrderFulfillmentStatus)

		var shippingAddress connectors.Address
		if len(o.FulfillmentStartInstructions) > 0 {
			ship := o.FulfillmentStartInstructions[0].ShippingStep.ShipTo
			shippingAddress = connectors.Address{
				Name:        ship.FullName,
				Line1:       ship.ContactAddress.AddressLine1,
				Line2:       ship.ContactAddress.AddressLine2,
				City:        ship.ContactAddress.City,
				State:       ship.ContactAddress.StateOrProvince,
				PostalCode:  ship.ContactAddress.PostalCode,
				CountryCode: ship.ContactAddress.CountryCode,
				Phone:       ship.PrimaryPhone.PhoneNumber,
				Email:       ship.Email,
			}
		}

		orders = append(orders, &connectors.Order{
			ID:              o.OrderID,
			ChannelOrderID:  o.OrderID,
			ChannelID:       "ebay",
			Status:          status,
			PaymentStatus:   o.OrderPaymentStatus,
			Total:           total,
			Currency:        o.PricingSummary.Total.Currency,
			Items:           items,
			ShippingAddress: shippingAddress,
			CustomerID:      o.Buyer.Username,
			CreatedAt:       createdAt,
		})
	}

	return orders, nil
}

func (c *Connector) mapOrderStatus(ebayStatus string) string {
	switch ebayStatus {
	case "NOT_STARTED":
		return connectors.OrderStatusPending
	case "IN_PROGRESS":
		return connectors.OrderStatusProcessing
	case "FULFILLED":
		return connectors.OrderStatusShipped
	default:
		return connectors.OrderStatusPending
	}
}

// GetOrder retrieves a single order
func (c *Connector) GetOrder(ctx context.Context, orderID string) (*connectors.Order, error) {
	path := fmt.Sprintf("/sell/fulfillment/v1/order/%s", url.PathEscape(orderID))

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var o struct {
		OrderID           string `json:"orderId"`
		CreationDate      string `json:"creationDate"`
		OrderFulfillmentStatus string `json:"orderFulfillmentStatus"`
		OrderPaymentStatus     string `json:"orderPaymentStatus"`
		PricingSummary    struct {
			Total struct {
				Value    string `json:"value"`
				Currency string `json:"currency"`
			} `json:"total"`
		} `json:"pricingSummary"`
		Buyer struct {
			Username string `json:"username"`
		} `json:"buyer"`
		LineItems []struct {
			LineItemID string `json:"lineItemId"`
			SKU        string `json:"sku"`
			Title      string `json:"title"`
			Quantity   int    `json:"quantity"`
			LineItemCost struct {
				Value    string `json:"value"`
				Currency string `json:"currency"`
			} `json:"lineItemCost"`
		} `json:"lineItems"`
	}

	if err := json.Unmarshal(respBody, &o); err != nil {
		return nil, err
	}

	createdAt, _ := time.Parse(time.RFC3339, o.CreationDate)
	total, _ := strconv.ParseFloat(o.PricingSummary.Total.Value, 64)

	var items []connectors.OrderItem
	for _, li := range o.LineItems {
		price, _ := strconv.ParseFloat(li.LineItemCost.Value, 64)
		items = append(items, connectors.OrderItem{
			ID:        li.LineItemID,
			ProductID: li.SKU,
			SKU:       li.SKU,
			Title:     li.Title,
			Quantity:  li.Quantity,
			Price:     price,
			Currency:  li.LineItemCost.Currency,
		})
	}

	return &connectors.Order{
		ID:             o.OrderID,
		ChannelOrderID: o.OrderID,
		ChannelID:      "ebay",
		Status:         c.mapOrderStatus(o.OrderFulfillmentStatus),
		PaymentStatus:  o.OrderPaymentStatus,
		Total:          total,
		Currency:       o.PricingSummary.Total.Currency,
		Items:          items,
		CustomerID:     o.Buyer.Username,
		CreatedAt:      createdAt,
	}, nil
}

// AcknowledgeOrder marks an order as acknowledged
func (c *Connector) AcknowledgeOrder(ctx context.Context, orderID string) error {
	// eBay doesn't have a direct acknowledge endpoint
	// Orders are acknowledged by creating a shipment
	return nil
}

// GetPrice retrieves pricing for a product
func (c *Connector) GetPrice(ctx context.Context, productID string) (*connectors.Price, error) {
	// Use Inventory API offer to get price
	path := "/sell/inventory/v1/offer"
	queryParams := url.Values{}
	queryParams.Set("sku", productID)
	path += "?" + queryParams.Encode()

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Offers []struct {
			OfferID string `json:"offerId"`
			SKU     string `json:"sku"`
			PricingSummary struct {
				Price struct {
					Value    string `json:"value"`
					Currency string `json:"currency"`
				} `json:"price"`
				OriginalRetailPrice struct {
					Value    string `json:"value"`
					Currency string `json:"currency"`
				} `json:"originalRetailPrice"`
			} `json:"pricingSummary"`
			ListingStatus string `json:"status"`
		} `json:"offers"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, err
	}

	if len(response.Offers) == 0 {
		return nil, fmt.Errorf("no offers found for SKU: %s", productID)
	}

	offer := response.Offers[0]
	price, _ := strconv.ParseFloat(offer.PricingSummary.Price.Value, 64)
	listPrice, _ := strconv.ParseFloat(offer.PricingSummary.OriginalRetailPrice.Value, 64)

	return &connectors.Price{
		ProductID:   productID,
		SKU:         offer.SKU,
		Price:       price,
		ListPrice:   listPrice,
		Currency:    offer.PricingSummary.Price.Currency,
		ChannelID:   "ebay",
		LastUpdated: time.Now(),
	}, nil
}

// UpdatePrice updates the price for a product
func (c *Connector) UpdatePrice(ctx context.Context, update connectors.PriceUpdate) error {
	// First get the existing offer
	path := "/sell/inventory/v1/offer"
	queryParams := url.Values{}
	queryParams.Set("sku", update.ProductID)
	path += "?" + queryParams.Encode()

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return err
	}

	var response struct {
		Offers []struct {
			OfferID string `json:"offerId"`
		} `json:"offers"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return err
	}

	if len(response.Offers) == 0 {
		return fmt.Errorf("no offers found for SKU: %s", update.ProductID)
	}

	// Update the offer
	offerID := response.Offers[0].OfferID
	updatePath := fmt.Sprintf("/sell/inventory/v1/offer/%s", offerID)

	// Get existing offer details
	offerBody, err := c.doRequest(ctx, "GET", updatePath, nil)
	if err != nil {
		return err
	}

	var offer map[string]interface{}
	if err := json.Unmarshal(offerBody, &offer); err != nil {
		return err
	}

	// Update pricing
	pricingSummary := offer["pricingSummary"].(map[string]interface{})
	priceObj := pricingSummary["price"].(map[string]interface{})
	priceObj["value"] = fmt.Sprintf("%.2f", update.Price)

	_, err = c.doRequest(ctx, "PUT", updatePath, offer)
	return err
}

// SyncInventory performs full inventory synchronization
func (c *Connector) SyncInventory(ctx context.Context) (*connectors.SyncResult, error) {
	result := &connectors.SyncResult{
		StartedAt: time.Now(),
		Channel:   "ebay",
	}

	// Get all inventory items
	products, err := c.GetProducts(ctx, connectors.ProductParams{Limit: 200})
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
	// eBay uses notification preferences instead of webhooks
	return []connectors.Webhook{}, nil
}

// CreateWebhook creates a webhook
func (c *Connector) CreateWebhook(ctx context.Context, webhook connectors.Webhook) error {
	// eBay uses notification preferences API
	return fmt.Errorf("eBay uses notification preferences instead of webhooks")
}

// DeleteWebhook deletes a webhook
func (c *Connector) DeleteWebhook(ctx context.Context, webhookID string) error {
	return fmt.Errorf("eBay uses notification preferences instead of webhooks")
}

// CreateShipment creates a shipment for an order
func (c *Connector) CreateShipment(ctx context.Context, orderID string, shipment *connectors.Shipment) error {
	path := fmt.Sprintf("/sell/fulfillment/v1/order/%s/shipping_fulfillment", url.PathEscape(orderID))

	// Get order to get line item IDs
	order, err := c.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	var lineItems []map[string]interface{}
	for _, item := range order.Items {
		lineItems = append(lineItems, map[string]interface{}{
			"lineItemId": item.ID,
			"quantity":   item.Quantity,
		})
	}

	body := map[string]interface{}{
		"lineItems":       lineItems,
		"shippingCarrierCode": shipment.Carrier,
		"trackingNumber":      shipment.TrackingNumber,
	}

	if !shipment.ShippedAt.IsZero() {
		body["shippedDate"] = shipment.ShippedAt.Format(time.RFC3339)
	}

	_, err = c.doRequest(ctx, "POST", path, body)
	return err
}
