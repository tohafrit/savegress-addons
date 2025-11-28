package orders

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/savegress/streamline/pkg/models"
	"github.com/shopspring/decimal"
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

// Router tests

func TestNewRouter(t *testing.T) {
	r := NewRouter()

	if r == nil {
		t.Fatal("expected non-nil router")
	}
	if r.warehouses == nil {
		t.Error("warehouses map should be initialized")
	}
	if r.optimization != OptimizationCost {
		t.Errorf("default optimization = %s, want cost", r.optimization)
	}
}

func TestRouter_SetOptimization(t *testing.T) {
	r := NewRouter()

	r.SetOptimization(OptimizationSpeed)
	if r.optimization != OptimizationSpeed {
		t.Errorf("optimization = %s, want speed", r.optimization)
	}

	r.SetOptimization(OptimizationBalance)
	if r.optimization != OptimizationBalance {
		t.Errorf("optimization = %s, want balance", r.optimization)
	}
}

func TestRouter_AddWarehouse(t *testing.T) {
	r := NewRouter()

	warehouse := &models.Warehouse{
		ID:     "WH-001",
		Name:   "Main Warehouse",
		Active: true,
	}

	r.AddWarehouse(warehouse)

	if _, ok := r.warehouses["WH-001"]; !ok {
		t.Error("warehouse not added")
	}
}

func TestRouter_SetShippingRate(t *testing.T) {
	r := NewRouter()

	rate := ShippingRate{
		CarrierID: "fedex",
	}

	r.SetShippingRate(rate)

	if _, ok := r.shippingRates["fedex"]; !ok {
		t.Error("shipping rate not added")
	}
}

func TestRouter_RouteOrder_NoWarehouses(t *testing.T) {
	r := NewRouter()
	ctx := context.Background()

	order := &models.Order{
		ID:      "ORD-001",
		Channel: "shopify",
		Items: []models.OrderItem{
			{SKU: "SKU-001", Quantity: 1},
		},
		Shipping: models.ShippingInfo{
			Address: models.Address{State: "CA"},
		},
	}

	decision, err := r.RouteOrder(ctx, order, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(decision.Options) != 0 {
		t.Errorf("expected 0 options, got %d", len(decision.Options))
	}
}

func TestRouter_RouteOrder_WithWarehouses(t *testing.T) {
	r := NewRouter()
	ctx := context.Background()

	// Add warehouses
	r.AddWarehouse(&models.Warehouse{
		ID:      "WH-001",
		Name:    "West Warehouse",
		Active:  true,
		Address: models.Address{State: "CA"},
	})
	r.AddWarehouse(&models.Warehouse{
		ID:      "WH-002",
		Name:    "East Warehouse",
		Active:  true,
		Address: models.Address{State: "NY"},
	})
	r.AddWarehouse(&models.Warehouse{
		ID:      "WH-003",
		Name:    "Inactive Warehouse",
		Active:  false,
		Address: models.Address{State: "TX"},
	})

	order := &models.Order{
		ID:      "ORD-001",
		Channel: "shopify",
		Items: []models.OrderItem{
			{SKU: "SKU-001", Quantity: 2},
		},
		Shipping: models.ShippingInfo{
			Address: models.Address{State: "CA"},
			Method:  "standard",
		},
	}

	inventory := map[string]*InventoryInfo{
		"SKU-001:WH-001": {SKU: "SKU-001", WarehouseID: "WH-001", Available: 10},
		"SKU-001:WH-002": {SKU: "SKU-001", WarehouseID: "WH-002", Available: 5},
	}

	decision, err := r.RouteOrder(ctx, order, inventory)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only 2 active warehouses
	if len(decision.Options) != 2 {
		t.Errorf("expected 2 options, got %d", len(decision.Options))
	}
	if decision.SelectedOption == nil {
		t.Error("expected a selected option")
	}
}

func TestRouter_RouteOrder_NoStockAnywhere(t *testing.T) {
	r := NewRouter()
	ctx := context.Background()

	r.AddWarehouse(&models.Warehouse{
		ID:      "WH-001",
		Name:    "Warehouse",
		Active:  true,
		Address: models.Address{State: "CA"},
	})

	order := &models.Order{
		ID:      "ORD-001",
		Channel: "shopify",
		Items: []models.OrderItem{
			{SKU: "SKU-001", Quantity: 2},
		},
		Shipping: models.ShippingInfo{
			Address: models.Address{State: "CA"},
		},
	}

	// Empty inventory
	inventory := map[string]*InventoryInfo{}

	decision, err := r.RouteOrder(ctx, order, inventory)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.SelectedOption == nil {
		t.Error("should still select best available option")
	}
	if decision.SelectedOption.AllItemsInStock {
		t.Error("AllItemsInStock should be false")
	}
}

func TestRouter_RouteOrder_PartialStock(t *testing.T) {
	r := NewRouter()
	ctx := context.Background()

	r.AddWarehouse(&models.Warehouse{
		ID:      "WH-001",
		Name:    "Warehouse 1",
		Active:  true,
		Address: models.Address{State: "CA"},
	})
	r.AddWarehouse(&models.Warehouse{
		ID:      "WH-002",
		Name:    "Warehouse 2",
		Active:  true,
		Address: models.Address{State: "NY"},
	})

	order := &models.Order{
		ID:      "ORD-001",
		Channel: "shopify",
		Items: []models.OrderItem{
			{SKU: "SKU-001", Quantity: 5},
			{SKU: "SKU-002", Quantity: 3},
		},
		Shipping: models.ShippingInfo{
			Address: models.Address{State: "CA"},
		},
	}

	// WH-001 has SKU-001, WH-002 has SKU-002
	inventory := map[string]*InventoryInfo{
		"SKU-001:WH-001": {SKU: "SKU-001", WarehouseID: "WH-001", Available: 10},
		"SKU-002:WH-002": {SKU: "SKU-002", WarehouseID: "WH-002", Available: 10},
	}

	decision, err := r.RouteOrder(ctx, order, inventory)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.IsSplitShipment == false {
		// Could be either split or best available depending on coverage
		if decision.SelectedOption != nil && decision.SelectedOption.AllItemsInStock {
			t.Error("no single warehouse should have all items")
		}
	}
}

func TestRouter_RouteOrder_InsufficientQuantity(t *testing.T) {
	r := NewRouter()
	ctx := context.Background()

	r.AddWarehouse(&models.Warehouse{
		ID:      "WH-001",
		Name:    "Warehouse",
		Active:  true,
		Address: models.Address{State: "CA"},
	})

	order := &models.Order{
		ID:      "ORD-001",
		Channel: "shopify",
		Items: []models.OrderItem{
			{SKU: "SKU-001", Quantity: 10}, // Want 10
		},
		Shipping: models.ShippingInfo{
			Address: models.Address{State: "CA"},
		},
	}

	// Only 5 available
	inventory := map[string]*InventoryInfo{
		"SKU-001:WH-001": {SKU: "SKU-001", WarehouseID: "WH-001", Available: 5},
	}

	decision, err := r.RouteOrder(ctx, order, inventory)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.SelectedOption.AllItemsInStock {
		t.Error("should not have all items in stock")
	}
}

func TestRouter_CalculateDistance(t *testing.T) {
	r := NewRouter()

	// Same state
	d := r.calculateDistance(models.Address{State: "CA"}, models.Address{State: "CA"})
	if d != 50 {
		t.Errorf("same state distance = %f, want 50", d)
	}

	// Different states
	d = r.calculateDistance(models.Address{State: "CA"}, models.Address{State: "NY"})
	if d != 500 {
		t.Errorf("different state distance = %f, want 500", d)
	}
}

func TestRouter_EstimateDeliveryDays(t *testing.T) {
	r := NewRouter()

	tests := []struct {
		distance float64
		method   string
		expected int
	}{
		{100, "express", 1},
		{100, "overnight", 1},
		{100, "expedited", 2},
		{100, "2day", 2},
		{50, "standard", 2},
		{200, "standard", 3},
		{600, "standard", 5},
		{1500, "standard", 7},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%fmi", tt.method, tt.distance), func(t *testing.T) {
			days := r.estimateDeliveryDays(tt.distance, tt.method)
			if days != tt.expected {
				t.Errorf("days = %d, want %d", days, tt.expected)
			}
		})
	}
}

func TestRouter_CalculateShippingCost(t *testing.T) {
	r := NewRouter()

	order := &models.Order{
		Items: []models.OrderItem{
			{SKU: "SKU-001", Quantity: 1},
			{SKU: "SKU-002", Quantity: 1},
		},
	}

	cost := r.calculateShippingCost(100, order)

	// Base: 5.00 + PerMile: 0.01*100 = 1.00 + PerItem: 0.50*2 = 1.00 = 7.00
	expected := 7.00
	costFloat, _ := cost.Float64()
	if costFloat != expected {
		t.Errorf("cost = %f, want %f", costFloat, expected)
	}
}

func TestRouter_CalculateScore_Cost(t *testing.T) {
	r := NewRouter()
	r.SetOptimization(OptimizationCost)

	option := RoutingOption{
		ShippingCost:    newDecimal(10.00),
		EstimatedDays:   3,
		AllItemsInStock: true,
		Distance:        100,
	}

	score := r.calculateScore(option)

	// 100 - cost*2(20) + allInStock(50) - distance*0.05(5) = 125
	if score != 125 {
		t.Errorf("cost score = %f, want 125", score)
	}
}

func TestRouter_CalculateScore_Speed(t *testing.T) {
	r := NewRouter()
	r.SetOptimization(OptimizationSpeed)

	option := RoutingOption{
		ShippingCost:    newDecimal(10.00),
		EstimatedDays:   2,
		AllItemsInStock: true,
		Distance:        100,
	}

	score := r.calculateScore(option)

	// 100 - days*10(20) + allInStock(50) - distance*0.05(5) = 125
	if score != 125 {
		t.Errorf("speed score = %f, want 125", score)
	}
}

func TestRouter_CalculateScore_Balance(t *testing.T) {
	r := NewRouter()
	r.SetOptimization(OptimizationBalance)

	option := RoutingOption{
		ShippingCost:    newDecimal(10.00),
		EstimatedDays:   3,
		AllItemsInStock: true,
		Distance:        100,
		ItemsAvailable:  map[string]int{"SKU-001": 20, "SKU-002": 10},
	}

	score := r.calculateScore(option)

	// 100 + totalAvailable*0.5(15) + allInStock(50) - distance*0.05(5) = 160
	if score != 160 {
		t.Errorf("balance score = %f, want 160", score)
	}
}

func TestRouter_CalculateScore_NegativeBecomesZero(t *testing.T) {
	r := NewRouter()
	r.SetOptimization(OptimizationCost)

	option := RoutingOption{
		ShippingCost:    newDecimal(100.00), // Very high cost
		EstimatedDays:   3,
		AllItemsInStock: false,
		Distance:        1000,
	}

	score := r.calculateScore(option)

	// 100 - cost*2(200) - distance*0.05(50) = -150 -> 0
	if score != 0 {
		t.Errorf("negative score should be 0, got %f", score)
	}
}

func TestRouter_GenerateReason(t *testing.T) {
	r := NewRouter()

	option := RoutingOption{
		ShippingCost:  newDecimal(10.50),
		EstimatedDays: 3,
	}

	r.SetOptimization(OptimizationCost)
	reason := r.generateReason(option)
	if reason != "Lowest shipping cost: $10.50" {
		t.Errorf("cost reason = %s", reason)
	}

	r.SetOptimization(OptimizationSpeed)
	reason = r.generateReason(option)
	if reason != "Fastest delivery: 3 days" {
		t.Errorf("speed reason = %s", reason)
	}

	r.SetOptimization(OptimizationBalance)
	reason = r.generateReason(option)
	if reason != "Best stock balance" {
		t.Errorf("balance reason = %s", reason)
	}
}

func TestRouter_AutoRoute(t *testing.T) {
	r := NewRouter()
	ctx := context.Background()

	r.AddWarehouse(&models.Warehouse{
		ID:      "WH-001",
		Name:    "Main Warehouse",
		Active:  true,
		Address: models.Address{State: "CA"},
	})

	order := &models.Order{
		ID:      "ORD-001",
		Channel: "shopify",
		Items: []models.OrderItem{
			{SKU: "SKU-001", Quantity: 2},
		},
		Shipping: models.ShippingInfo{
			Address: models.Address{State: "CA"},
		},
	}

	inventory := map[string]*InventoryInfo{
		"SKU-001:WH-001": {SKU: "SKU-001", WarehouseID: "WH-001", Available: 10},
	}

	result, err := r.AutoRoute(ctx, order, inventory)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.WarehouseID != "WH-001" {
		t.Errorf("WarehouseID = %s, want WH-001", result.WarehouseID)
	}
}

func TestRouter_AutoRoute_SplitShipment(t *testing.T) {
	r := NewRouter()
	ctx := context.Background()

	r.AddWarehouse(&models.Warehouse{
		ID:      "WH-001",
		Name:    "Warehouse 1",
		Active:  true,
		Address: models.Address{State: "CA"},
	})
	r.AddWarehouse(&models.Warehouse{
		ID:      "WH-002",
		Name:    "Warehouse 2",
		Active:  true,
		Address: models.Address{State: "NY"},
	})

	order := &models.Order{
		ID:      "ORD-001",
		Channel: "shopify",
		Items: []models.OrderItem{
			{SKU: "SKU-001", Quantity: 2},
			{SKU: "SKU-002", Quantity: 2},
		},
		Shipping: models.ShippingInfo{
			Address: models.Address{State: "CA"},
		},
	}

	// Each warehouse has only one SKU
	inventory := map[string]*InventoryInfo{
		"SKU-001:WH-001": {SKU: "SKU-001", WarehouseID: "WH-001", Available: 10},
		"SKU-002:WH-002": {SKU: "SKU-002", WarehouseID: "WH-002", Available: 10},
	}

	result, err := r.AutoRoute(ctx, order, inventory)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should assign to first split warehouse
	if result.WarehouseID == "" {
		t.Error("WarehouseID should be assigned")
	}
}

func TestProcessor_RouteOrder(t *testing.T) {
	router := NewRouter()
	router.AddWarehouse(&models.Warehouse{
		ID:      "WH-001",
		Name:    "Warehouse",
		Active:  true,
		Address: models.Address{State: "CA"},
	})

	p := NewProcessor(router)
	ctx := context.Background()

	order := &models.Order{
		ID:      "ORD-001",
		Channel: "shopify",
		Items: []models.OrderItem{
			{SKU: "SKU-001", Quantity: 1},
		},
		Shipping: models.ShippingInfo{
			Address: models.Address{State: "CA"},
		},
	}
	p.CreateOrder(ctx, order)

	inventory := map[string]*InventoryInfo{
		"SKU-001:WH-001": {SKU: "SKU-001", WarehouseID: "WH-001", Available: 10},
	}

	decision, err := p.RouteOrder(ctx, "ORD-001", inventory)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil {
		t.Fatal("expected routing decision")
	}
}

func TestProcessor_RouteOrder_NotFound(t *testing.T) {
	router := NewRouter()
	p := NewProcessor(router)
	ctx := context.Background()

	_, err := p.RouteOrder(ctx, "NONEXISTENT", nil)

	if err == nil {
		t.Error("expected error for nonexistent order")
	}
}

func TestProcessor_AssignWarehouse_WithCallback(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	var updatedOrder *models.Order
	p.SetCallbacks(nil, func(o *models.Order) { updatedOrder = o }, nil)

	order := &models.Order{ID: "ORD-001"}
	p.CreateOrder(ctx, order)

	err := p.AssignWarehouse(ctx, "ORD-001", "WH-001")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedOrder == nil {
		t.Error("onUpdate callback not called")
	}
}

func TestProcessor_HoldOrder_WithCallback(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	var updatedOrder *models.Order
	p.SetCallbacks(nil, func(o *models.Order) { updatedOrder = o }, nil)

	order := &models.Order{ID: "ORD-001"}
	p.CreateOrder(ctx, order)

	err := p.HoldOrder(ctx, "ORD-001", "Payment check")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedOrder == nil {
		t.Error("onUpdate callback not called")
	}
}

func TestProcessor_CancelOrder_WithCallback(t *testing.T) {
	p := NewProcessor(nil)
	ctx := context.Background()

	var updatedOrder *models.Order
	p.SetCallbacks(nil, func(o *models.Order) { updatedOrder = o }, nil)

	order := &models.Order{ID: "ORD-001"}
	p.CreateOrder(ctx, order)

	err := p.CancelOrder(ctx, "ORD-001", "Customer request")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedOrder == nil {
		t.Error("onUpdate callback not called")
	}
}

// Helper function
func newDecimal(f float64) decimal.Decimal {
	return decimal.NewFromFloat(f)
}
