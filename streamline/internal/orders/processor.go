package orders

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/savegress/streamline/pkg/models"
)

// Processor handles order processing and status management
type Processor struct {
	orders   map[string]*models.Order
	ordersMu sync.RWMutex

	router *Router

	// Callbacks
	onOrderCreated   func(*models.Order)
	onOrderUpdated   func(*models.Order)
	onOrderFulfilled func(*models.Order)
}

// NewProcessor creates a new order processor
func NewProcessor(router *Router) *Processor {
	return &Processor{
		orders: make(map[string]*models.Order),
		router: router,
	}
}

// SetCallbacks sets event callbacks
func (p *Processor) SetCallbacks(onCreate, onUpdate, onFulfill func(*models.Order)) {
	p.onOrderCreated = onCreate
	p.onOrderUpdated = onUpdate
	p.onOrderFulfilled = onFulfill
}

// CreateOrder creates a new order
func (p *Processor) CreateOrder(ctx context.Context, order *models.Order) error {
	p.ordersMu.Lock()
	defer p.ordersMu.Unlock()

	if order.ID == "" {
		order.ID = generateOrderID()
	}

	order.Status = models.OrderStatusPending
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	p.orders[order.ID] = order

	if p.onOrderCreated != nil {
		p.onOrderCreated(order)
	}

	return nil
}

// GetOrder returns an order by ID
func (p *Processor) GetOrder(id string) (*models.Order, bool) {
	p.ordersMu.RLock()
	defer p.ordersMu.RUnlock()
	order, ok := p.orders[id]
	return order, ok
}

// ListOrders returns orders with optional filters
func (p *Processor) ListOrders(filter OrderFilter) []*models.Order {
	p.ordersMu.RLock()
	defer p.ordersMu.RUnlock()

	result := make([]*models.Order, 0)
	for _, order := range p.orders {
		if filter.matches(order) {
			result = append(result, order)
		}
	}

	// Sort by created_at desc
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].CreatedAt.After(result[i].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	// Apply limit
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}

	return result
}

// OrderFilter defines order filtering options
type OrderFilter struct {
	Status    models.OrderStatus
	Channel   string
	From      *time.Time
	To        *time.Time
	Warehouse string
	Limit     int
}

func (f OrderFilter) matches(order *models.Order) bool {
	if f.Status != "" && order.Status != f.Status {
		return false
	}
	if f.Channel != "" && order.Channel != f.Channel {
		return false
	}
	if f.Warehouse != "" && order.WarehouseID != f.Warehouse {
		return false
	}
	if f.From != nil && order.CreatedAt.Before(*f.From) {
		return false
	}
	if f.To != nil && order.CreatedAt.After(*f.To) {
		return false
	}
	return true
}

// UpdateStatus updates an order's status
func (p *Processor) UpdateStatus(ctx context.Context, orderID string, status models.OrderStatus) error {
	p.ordersMu.Lock()
	defer p.ordersMu.Unlock()

	order, ok := p.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found: %s", orderID)
	}

	// Validate status transition
	if !isValidTransition(order.Status, status) {
		return fmt.Errorf("invalid status transition: %s -> %s", order.Status, status)
	}

	order.Status = status
	order.UpdatedAt = time.Now()

	if p.onOrderUpdated != nil {
		p.onOrderUpdated(order)
	}

	return nil
}

// RouteOrder routes an order to a warehouse
func (p *Processor) RouteOrder(ctx context.Context, orderID string, inventory map[string]*InventoryInfo) (*RoutingDecision, error) {
	order, ok := p.GetOrder(orderID)
	if !ok {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	return p.router.RouteOrder(ctx, order, inventory)
}

// AssignWarehouse assigns an order to a warehouse
func (p *Processor) AssignWarehouse(ctx context.Context, orderID, warehouseID string) error {
	p.ordersMu.Lock()
	defer p.ordersMu.Unlock()

	order, ok := p.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found: %s", orderID)
	}

	order.WarehouseID = warehouseID
	order.Status = models.OrderStatusProcessing
	order.UpdatedAt = time.Now()

	if p.onOrderUpdated != nil {
		p.onOrderUpdated(order)
	}

	return nil
}

// FulfillOrder marks an order as fulfilled
func (p *Processor) FulfillOrder(ctx context.Context, orderID string, fulfillment *models.Fulfillment) error {
	p.ordersMu.Lock()
	defer p.ordersMu.Unlock()

	order, ok := p.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found: %s", orderID)
	}

	if order.Status == models.OrderStatusCancelled || order.Status == models.OrderStatusRefunded {
		return fmt.Errorf("cannot fulfill cancelled/refunded order")
	}

	now := time.Now()
	fulfillment.ShippedAt = &now
	fulfillment.Status = models.FulfillmentShipped

	order.Fulfillment = fulfillment
	order.Status = models.OrderStatusShipped
	order.UpdatedAt = now

	if p.onOrderFulfilled != nil {
		p.onOrderFulfilled(order)
	}

	return nil
}

// CancelOrder cancels an order
func (p *Processor) CancelOrder(ctx context.Context, orderID, reason string) error {
	p.ordersMu.Lock()
	defer p.ordersMu.Unlock()

	order, ok := p.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found: %s", orderID)
	}

	if order.Status == models.OrderStatusShipped || order.Status == models.OrderStatusDelivered {
		return fmt.Errorf("cannot cancel shipped/delivered order")
	}

	order.Status = models.OrderStatusCancelled
	order.Notes = reason
	order.UpdatedAt = time.Now()

	if p.onOrderUpdated != nil {
		p.onOrderUpdated(order)
	}

	return nil
}

// HoldOrder puts an order on hold
func (p *Processor) HoldOrder(ctx context.Context, orderID, reason string) error {
	p.ordersMu.Lock()
	defer p.ordersMu.Unlock()

	order, ok := p.orders[orderID]
	if !ok {
		return fmt.Errorf("order not found: %s", orderID)
	}

	order.Status = models.OrderStatusOnHold
	order.Notes = reason
	order.UpdatedAt = time.Now()

	if p.onOrderUpdated != nil {
		p.onOrderUpdated(order)
	}

	return nil
}

// GetOrdersByChannel returns orders for a specific channel
func (p *Processor) GetOrdersByChannel(channel string, limit int) []*models.Order {
	return p.ListOrders(OrderFilter{Channel: channel, Limit: limit})
}

// GetPendingOrders returns all pending orders
func (p *Processor) GetPendingOrders() []*models.Order {
	return p.ListOrders(OrderFilter{Status: models.OrderStatusPending})
}

// OrderStats contains order statistics
type OrderStats struct {
	Total       int            `json:"total"`
	ByStatus    map[string]int `json:"by_status"`
	ByChannel   map[string]int `json:"by_channel"`
	TodayCount  int            `json:"today_count"`
	PendingQty  int            `json:"pending_qty"`
}

// GetStats returns order statistics
func (p *Processor) GetStats() *OrderStats {
	p.ordersMu.RLock()
	defer p.ordersMu.RUnlock()

	stats := &OrderStats{
		Total:     len(p.orders),
		ByStatus:  make(map[string]int),
		ByChannel: make(map[string]int),
	}

	today := time.Now().Truncate(24 * time.Hour)

	for _, order := range p.orders {
		stats.ByStatus[string(order.Status)]++
		stats.ByChannel[order.Channel]++

		if order.CreatedAt.After(today) {
			stats.TodayCount++
		}

		if order.Status == models.OrderStatusPending {
			for _, item := range order.Items {
				stats.PendingQty += item.Quantity
			}
		}
	}

	return stats
}

func isValidTransition(from, to models.OrderStatus) bool {
	validTransitions := map[models.OrderStatus][]models.OrderStatus{
		models.OrderStatusPending:    {models.OrderStatusProcessing, models.OrderStatusCancelled, models.OrderStatusOnHold},
		models.OrderStatusProcessing: {models.OrderStatusShipped, models.OrderStatusCancelled, models.OrderStatusOnHold},
		models.OrderStatusShipped:    {models.OrderStatusDelivered, models.OrderStatusRefunded},
		models.OrderStatusDelivered:  {models.OrderStatusRefunded},
		models.OrderStatusOnHold:     {models.OrderStatusPending, models.OrderStatusProcessing, models.OrderStatusCancelled},
		models.OrderStatusCancelled:  {}, // Terminal state
		models.OrderStatusRefunded:   {}, // Terminal state
	}

	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}

	for _, status := range allowed {
		if status == to {
			return true
		}
	}

	return false
}

func generateOrderID() string {
	return fmt.Sprintf("ORD-%d", time.Now().UnixNano())
}
