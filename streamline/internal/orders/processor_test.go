package orders

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/savegress/streamline/pkg/models"
)

func TestNewProcessor(t *testing.T) {
	router := NewRouter()
	p := NewProcessor(router)

	if p == nil {
		t.Fatal("expected non-nil processor")
	}
	if p.orders == nil {
		t.Error("orders map should be initialized")
	}
	if p.router != router {
		t.Error("router not set correctly")
	}
}

func TestProcessor_SetCallbacks(t *testing.T) {
	p := NewProcessor(nil)

	var createCalled, updateCalled, fulfillCalled bool

	p.SetCallbacks(
		func(o *models.Order) { createCalled = true },
		func(o *models.Order) { updateCalled = true },
		func(o *models.Order) { fulfillCalled = true },
	)

	if p.onOrderCreated == nil {
		t.Error("onCreate callback not set")
	}
	if p.onOrderUpdated == nil {
		t.Error("onUpdate callback not set")
	}
	if p.onOrderFulfilled == nil {
		t.Error("onFulfill callback not set")
	}

	// Trigger callbacks
	p.onOrderCreated(&models.Order{})
	p.onOrderUpdated(&models.Order{})
	p.onOrderFulfilled(&models.Order{})

	if !createCalled {
		t.Error("onCreate callback not triggered")
	}
	if !updateCalled {
		t.Error("onUpdate callback not triggered")
	}
	if !fulfillCalled {
		t.Error("onFulfill callback not triggered")
	}
}

func TestProcessor_CreateOrder(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	var createdOrder *models.Order
	p.SetCallbacks(func(o *models.Order) { createdOrder = o }, nil, nil)

	order := &models.Order{
		Channel: "shopify",
		Items: []models.OrderItem{
			{SKU: "SKU-001", Quantity: 2},
		},
	}

	err := p.CreateOrder(ctx, order)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.ID == "" {
		t.Error("order ID should be generated")
	}
	if order.Status != models.OrderStatusPending {
		t.Errorf("status = %s, want pending", order.Status)
	}
	if order.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if order.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
	if createdOrder == nil {
		t.Error("onCreate callback not called")
	}
}

func TestProcessor_CreateOrder_WithID(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	order := &models.Order{
		ID:      "EXISTING-001",
		Channel: "shopify",
	}

	err := p.CreateOrder(ctx, order)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.ID != "EXISTING-001" {
		t.Errorf("ID = %s, want EXISTING-001", order.ID)
	}
}

func TestProcessor_GetOrder(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	order := &models.Order{ID: "ORD-001", Channel: "shopify"}
	p.CreateOrder(ctx, order)

	got, ok := p.GetOrder("ORD-001")
	if !ok {
		t.Fatal("expected to find order")
	}
	if got.ID != "ORD-001" {
		t.Errorf("ID = %s, want ORD-001", got.ID)
	}

	_, ok = p.GetOrder("NONEXISTENT")
	if ok {
		t.Error("should not find nonexistent order")
	}
}

func TestProcessor_ListOrders(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	// Create orders with slight time gaps
	for i := 0; i < 5; i++ {
		p.CreateOrder(ctx, &models.Order{
			Channel: "shopify",
			Status:  models.OrderStatusPending,
		})
		time.Sleep(time.Millisecond)
	}

	orders := p.ListOrders(OrderFilter{})
	if len(orders) != 5 {
		t.Errorf("expected 5 orders, got %d", len(orders))
	}

	// Check sorted by created_at desc
	for i := 0; i < len(orders)-1; i++ {
		if orders[i].CreatedAt.Before(orders[i+1].CreatedAt) {
			t.Error("orders not sorted by created_at desc")
		}
	}
}

func TestProcessor_ListOrders_WithFilters(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	p.CreateOrder(ctx, &models.Order{ID: "ORD-001", Channel: "shopify"})
	p.CreateOrder(ctx, &models.Order{ID: "ORD-002", Channel: "amazon"})
	p.CreateOrder(ctx, &models.Order{ID: "ORD-003", Channel: "shopify"})

	// Update ORD-003 to processing (CreateOrder always sets to pending)
	p.UpdateStatus(ctx, "ORD-003", models.OrderStatusProcessing)

	// Filter by channel
	shopifyOrders := p.ListOrders(OrderFilter{Channel: "shopify"})
	if len(shopifyOrders) != 2 {
		t.Errorf("expected 2 shopify orders, got %d", len(shopifyOrders))
	}

	// Filter by status
	pendingOrders := p.ListOrders(OrderFilter{Status: models.OrderStatusPending})
	if len(pendingOrders) != 2 {
		t.Errorf("expected 2 pending orders, got %d", len(pendingOrders))
	}
}

func TestProcessor_ListOrders_WithLimit(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		p.CreateOrder(ctx, &models.Order{ID: fmt.Sprintf("ORD-LIM-%03d", i), Channel: "shopify"})
	}

	orders := p.ListOrders(OrderFilter{Limit: 5})
	if len(orders) != 5 {
		t.Errorf("expected 5 orders with limit, got %d", len(orders))
	}
}

func TestProcessor_ListOrders_TimeFilters(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	// Create an order
	p.CreateOrder(ctx, &models.Order{ID: "ORD-001", Channel: "shopify"})

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	// Filter from past should include order
	orders := p.ListOrders(OrderFilter{From: &past})
	if len(orders) != 1 {
		t.Errorf("expected 1 order from past, got %d", len(orders))
	}

	// Filter from future should exclude order
	orders = p.ListOrders(OrderFilter{From: &future})
	if len(orders) != 0 {
		t.Errorf("expected 0 orders from future, got %d", len(orders))
	}

	// Filter to future should include order
	orders = p.ListOrders(OrderFilter{To: &future})
	if len(orders) != 1 {
		t.Errorf("expected 1 order to future, got %d", len(orders))
	}

	// Filter to past should exclude order
	orders = p.ListOrders(OrderFilter{To: &past})
	if len(orders) != 0 {
		t.Errorf("expected 0 orders to past, got %d", len(orders))
	}
}

func TestProcessor_UpdateStatus(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	var updatedOrder *models.Order
	p.SetCallbacks(nil, func(o *models.Order) { updatedOrder = o }, nil)

	order := &models.Order{ID: "ORD-001", Status: models.OrderStatusPending}
	p.CreateOrder(ctx, order)

	err := p.UpdateStatus(ctx, "ORD-001", models.OrderStatusProcessing)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := p.GetOrder("ORD-001")
	if got.Status != models.OrderStatusProcessing {
		t.Errorf("status = %s, want processing", got.Status)
	}
	if updatedOrder == nil {
		t.Error("onUpdate callback not called")
	}
}

func TestProcessor_UpdateStatus_NotFound(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	err := p.UpdateStatus(ctx, "NONEXISTENT", models.OrderStatusProcessing)

	if err == nil {
		t.Error("expected error for nonexistent order")
	}
}

func TestProcessor_UpdateStatus_InvalidTransition(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	order := &models.Order{ID: "ORD-001", Status: models.OrderStatusCancelled}
	p.CreateOrder(ctx, order)
	// Manually set to cancelled (terminal state)
	p.orders["ORD-001"].Status = models.OrderStatusCancelled

	err := p.UpdateStatus(ctx, "ORD-001", models.OrderStatusProcessing)

	if err == nil {
		t.Error("expected error for invalid transition from cancelled")
	}
}

func TestProcessor_AssignWarehouse(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	order := &models.Order{ID: "ORD-001"}
	p.CreateOrder(ctx, order)

	err := p.AssignWarehouse(ctx, "ORD-001", "WH-001")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := p.GetOrder("ORD-001")
	if got.WarehouseID != "WH-001" {
		t.Errorf("WarehouseID = %s, want WH-001", got.WarehouseID)
	}
	if got.Status != models.OrderStatusProcessing {
		t.Errorf("Status = %s, want processing", got.Status)
	}
}

func TestProcessor_AssignWarehouse_NotFound(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	err := p.AssignWarehouse(ctx, "NONEXISTENT", "WH-001")

	if err == nil {
		t.Error("expected error for nonexistent order")
	}
}

func TestProcessor_FulfillOrder(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	var fulfilledOrder *models.Order
	p.SetCallbacks(nil, nil, func(o *models.Order) { fulfilledOrder = o })

	order := &models.Order{ID: "ORD-001", Status: models.OrderStatusProcessing}
	p.CreateOrder(ctx, order)
	p.orders["ORD-001"].Status = models.OrderStatusProcessing

	fulfillment := &models.Fulfillment{
		Carrier:     "FedEx",
		TrackingNum: "TRACK123",
	}

	err := p.FulfillOrder(ctx, "ORD-001", fulfillment)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := p.GetOrder("ORD-001")
	if got.Status != models.OrderStatusShipped {
		t.Errorf("Status = %s, want shipped", got.Status)
	}
	if got.Fulfillment == nil {
		t.Fatal("Fulfillment should be set")
	}
	if got.Fulfillment.Status != models.FulfillmentShipped {
		t.Errorf("Fulfillment.Status = %s, want shipped", got.Fulfillment.Status)
	}
	if got.Fulfillment.ShippedAt == nil {
		t.Error("ShippedAt should be set")
	}
	if fulfilledOrder == nil {
		t.Error("onFulfill callback not called")
	}
}

func TestProcessor_FulfillOrder_NotFound(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	err := p.FulfillOrder(ctx, "NONEXISTENT", &models.Fulfillment{})

	if err == nil {
		t.Error("expected error for nonexistent order")
	}
}

func TestProcessor_FulfillOrder_Cancelled(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	order := &models.Order{ID: "ORD-001"}
	p.CreateOrder(ctx, order)
	p.orders["ORD-001"].Status = models.OrderStatusCancelled

	err := p.FulfillOrder(ctx, "ORD-001", &models.Fulfillment{})

	if err == nil {
		t.Error("expected error for cancelled order")
	}
}

func TestProcessor_FulfillOrder_Refunded(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	order := &models.Order{ID: "ORD-001"}
	p.CreateOrder(ctx, order)
	p.orders["ORD-001"].Status = models.OrderStatusRefunded

	err := p.FulfillOrder(ctx, "ORD-001", &models.Fulfillment{})

	if err == nil {
		t.Error("expected error for refunded order")
	}
}

func TestProcessor_CancelOrder(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	order := &models.Order{ID: "ORD-001"}
	p.CreateOrder(ctx, order)

	err := p.CancelOrder(ctx, "ORD-001", "Customer requested")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := p.GetOrder("ORD-001")
	if got.Status != models.OrderStatusCancelled {
		t.Errorf("Status = %s, want cancelled", got.Status)
	}
	if got.Notes != "Customer requested" {
		t.Errorf("Notes = %s, want 'Customer requested'", got.Notes)
	}
}

func TestProcessor_CancelOrder_NotFound(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	err := p.CancelOrder(ctx, "NONEXISTENT", "reason")

	if err == nil {
		t.Error("expected error for nonexistent order")
	}
}

func TestProcessor_CancelOrder_Shipped(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	order := &models.Order{ID: "ORD-001"}
	p.CreateOrder(ctx, order)
	p.orders["ORD-001"].Status = models.OrderStatusShipped

	err := p.CancelOrder(ctx, "ORD-001", "reason")

	if err == nil {
		t.Error("expected error for shipped order")
	}
}

func TestProcessor_CancelOrder_Delivered(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	order := &models.Order{ID: "ORD-001"}
	p.CreateOrder(ctx, order)
	p.orders["ORD-001"].Status = models.OrderStatusDelivered

	err := p.CancelOrder(ctx, "ORD-001", "reason")

	if err == nil {
		t.Error("expected error for delivered order")
	}
}

func TestProcessor_HoldOrder(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	order := &models.Order{ID: "ORD-001"}
	p.CreateOrder(ctx, order)

	err := p.HoldOrder(ctx, "ORD-001", "Payment verification")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := p.GetOrder("ORD-001")
	if got.Status != models.OrderStatusOnHold {
		t.Errorf("Status = %s, want on_hold", got.Status)
	}
	if got.Notes != "Payment verification" {
		t.Errorf("Notes = %s, want 'Payment verification'", got.Notes)
	}
}

func TestProcessor_HoldOrder_NotFound(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	err := p.HoldOrder(ctx, "NONEXISTENT", "reason")

	if err == nil {
		t.Error("expected error for nonexistent order")
	}
}

func TestProcessor_GetOrdersByChannel(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	p.CreateOrder(ctx, &models.Order{ID: "ORD-CH-001", Channel: "shopify"})
	p.CreateOrder(ctx, &models.Order{ID: "ORD-CH-002", Channel: "shopify"})
	p.CreateOrder(ctx, &models.Order{ID: "ORD-CH-003", Channel: "amazon"})

	orders := p.GetOrdersByChannel("shopify", 10)
	if len(orders) != 2 {
		t.Errorf("expected 2 shopify orders, got %d", len(orders))
	}
}

func TestProcessor_GetPendingOrders(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	p.CreateOrder(ctx, &models.Order{ID: "ORD-001"})
	p.CreateOrder(ctx, &models.Order{ID: "ORD-002"})
	p.CreateOrder(ctx, &models.Order{ID: "ORD-003"})
	p.orders["ORD-003"].Status = models.OrderStatusProcessing

	pending := p.GetPendingOrders()
	if len(pending) != 2 {
		t.Errorf("expected 2 pending orders, got %d", len(pending))
	}
}

func TestProcessor_GetStats(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	p.CreateOrder(ctx, &models.Order{
		ID:      "ORD-001",
		Channel: "shopify",
		Items:   []models.OrderItem{{SKU: "SKU-001", Quantity: 5}},
	})
	p.CreateOrder(ctx, &models.Order{
		ID:      "ORD-002",
		Channel: "shopify",
		Items:   []models.OrderItem{{SKU: "SKU-002", Quantity: 3}},
	})
	p.CreateOrder(ctx, &models.Order{
		ID:      "ORD-003",
		Channel: "amazon",
		Items:   []models.OrderItem{{SKU: "SKU-003", Quantity: 2}},
	})
	p.orders["ORD-003"].Status = models.OrderStatusProcessing

	stats := p.GetStats()

	if stats.Total != 3 {
		t.Errorf("Total = %d, want 3", stats.Total)
	}
	if stats.ByChannel["shopify"] != 2 {
		t.Errorf("shopify count = %d, want 2", stats.ByChannel["shopify"])
	}
	if stats.ByChannel["amazon"] != 1 {
		t.Errorf("amazon count = %d, want 1", stats.ByChannel["amazon"])
	}
	if stats.ByStatus[string(models.OrderStatusPending)] != 2 {
		t.Errorf("pending count = %d, want 2", stats.ByStatus[string(models.OrderStatusPending)])
	}
	if stats.PendingQty != 8 { // 5 + 3 from pending orders
		t.Errorf("PendingQty = %d, want 8", stats.PendingQty)
	}
	if stats.TodayCount != 3 {
		t.Errorf("TodayCount = %d, want 3", stats.TodayCount)
	}
}

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		from  models.OrderStatus
		to    models.OrderStatus
		valid bool
	}{
		{models.OrderStatusPending, models.OrderStatusProcessing, true},
		{models.OrderStatusPending, models.OrderStatusCancelled, true},
		{models.OrderStatusPending, models.OrderStatusOnHold, true},
		{models.OrderStatusPending, models.OrderStatusShipped, false},
		{models.OrderStatusProcessing, models.OrderStatusShipped, true},
		{models.OrderStatusProcessing, models.OrderStatusCancelled, true},
		{models.OrderStatusShipped, models.OrderStatusDelivered, true},
		{models.OrderStatusShipped, models.OrderStatusRefunded, true},
		{models.OrderStatusShipped, models.OrderStatusPending, false},
		{models.OrderStatusDelivered, models.OrderStatusRefunded, true},
		{models.OrderStatusDelivered, models.OrderStatusPending, false},
		{models.OrderStatusOnHold, models.OrderStatusPending, true},
		{models.OrderStatusOnHold, models.OrderStatusProcessing, true},
		{models.OrderStatusCancelled, models.OrderStatusPending, false},
		{models.OrderStatusRefunded, models.OrderStatusPending, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			result := isValidTransition(tt.from, tt.to)
			if result != tt.valid {
				t.Errorf("isValidTransition(%s, %s) = %v, want %v", tt.from, tt.to, result, tt.valid)
			}
		})
	}
}

func TestIsValidTransition_UnknownStatus(t *testing.T) {
	result := isValidTransition("unknown", models.OrderStatusPending)
	if result {
		t.Error("should not allow transitions from unknown status")
	}
}

func TestGenerateOrderID(t *testing.T) {
	id1 := generateOrderID()
	time.Sleep(time.Nanosecond)
	id2 := generateOrderID()

	if id1 == "" {
		t.Error("generated ID should not be empty")
	}
	if id1 == id2 {
		t.Error("generated IDs should be unique")
	}
	if len(id1) < 4 || id1[:4] != "ORD-" {
		t.Errorf("ID should start with 'ORD-', got %s", id1)
	}
}

func TestOrderFilter_Matches(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	order := &models.Order{
		Status:      models.OrderStatusPending,
		Channel:     "shopify",
		WarehouseID: "WH-001",
		CreatedAt:   now,
	}

	tests := []struct {
		name    string
		filter  OrderFilter
		matches bool
	}{
		{"empty filter", OrderFilter{}, true},
		{"matching status", OrderFilter{Status: models.OrderStatusPending}, true},
		{"non-matching status", OrderFilter{Status: models.OrderStatusProcessing}, false},
		{"matching channel", OrderFilter{Channel: "shopify"}, true},
		{"non-matching channel", OrderFilter{Channel: "amazon"}, false},
		{"matching warehouse", OrderFilter{Warehouse: "WH-001"}, true},
		{"non-matching warehouse", OrderFilter{Warehouse: "WH-002"}, false},
		{"from past", OrderFilter{From: &past}, true},
		{"from future", OrderFilter{From: &future}, false},
		{"to future", OrderFilter{To: &future}, true},
		{"to past", OrderFilter{To: &past}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.matches(order)
			if result != tt.matches {
				t.Errorf("filter.matches() = %v, want %v", result, tt.matches)
			}
		})
	}
}
