package inventory

import (
	"context"
	"testing"
	"time"

	"github.com/savegress/streamline/pkg/models"
)

// mockChannelHandler implements ChannelHandler for testing
type mockChannelHandler struct {
	channelID string
	inventory map[string]int
	updateErr error
	getErr    error
}

func (m *mockChannelHandler) GetInventory(ctx context.Context, sku string) (int, error) {
	if m.getErr != nil {
		return 0, m.getErr
	}
	return m.inventory[sku], nil
}

func (m *mockChannelHandler) UpdateInventory(ctx context.Context, sku string, quantity int) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if m.inventory == nil {
		m.inventory = make(map[string]int)
	}
	m.inventory[sku] = quantity
	return nil
}

func (m *mockChannelHandler) GetChannelID() string {
	return m.channelID
}

func TestNewEngine(t *testing.T) {
	e := NewEngine()
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
	if e.inventory == nil {
		t.Error("expected inventory map to be initialized")
	}
	if e.warehouses == nil {
		t.Error("expected warehouses map to be initialized")
	}
	if e.channelHandlers == nil {
		t.Error("expected channelHandlers map to be initialized")
	}
	if e.allocator == nil {
		t.Error("expected allocator to be initialized")
	}
	if e.reconciler == nil {
		t.Error("expected reconciler to be initialized")
	}
}

func TestEngine_StartStop(t *testing.T) {
	e := NewEngine()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := e.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	e.Stop()
}

func TestEngine_SetAlertCallback(t *testing.T) {
	e := NewEngine()

	var receivedAlert *models.Alert
	e.SetAlertCallback(func(a *models.Alert) {
		receivedAlert = a
	})

	if e.onAlert == nil {
		t.Error("expected callback to be set")
	}

	// Trigger an alert
	inv := &models.Inventory{
		SKU:          "TEST-001",
		Available:    0,
		ReorderPoint: 10,
		Warehouses:   make(map[string]models.WarehouseStock),
	}
	e.SetInventory(inv)

	time.Sleep(10 * time.Millisecond)

	if receivedAlert == nil {
		t.Error("expected alert to be received")
	}
}

func TestEngine_RegisterChannel(t *testing.T) {
	e := NewEngine()

	handler := &mockChannelHandler{
		channelID: "shopify",
		inventory: make(map[string]int),
	}

	e.RegisterChannel(handler)

	e.handlersMu.RLock()
	defer e.handlersMu.RUnlock()

	if _, ok := e.channelHandlers["shopify"]; !ok {
		t.Error("expected channel handler to be registered")
	}
}

func TestEngine_AddWarehouse(t *testing.T) {
	e := NewEngine()

	warehouse := &models.Warehouse{
		ID:     "WH-001",
		Name:   "Main Warehouse",
		Active: true,
	}

	e.AddWarehouse(warehouse)

	e.warehousesMu.RLock()
	defer e.warehousesMu.RUnlock()

	if _, ok := e.warehouses["WH-001"]; !ok {
		t.Error("expected warehouse to be added")
	}
}

func TestEngine_GetInventory(t *testing.T) {
	e := NewEngine()

	// Not found
	_, ok := e.GetInventory("NONEXISTENT")
	if ok {
		t.Error("expected not to find nonexistent inventory")
	}

	// Set and get
	inv := &models.Inventory{
		SKU:        "TEST-001",
		TotalStock: 100,
		Available:  100,
	}
	e.SetInventory(inv)

	got, ok := e.GetInventory("TEST-001")
	if !ok {
		t.Fatal("expected to find inventory")
	}
	if got.SKU != "TEST-001" {
		t.Errorf("SKU = %s, want TEST-001", got.SKU)
	}
}

func TestEngine_GetAllInventory(t *testing.T) {
	e := NewEngine()

	e.SetInventory(&models.Inventory{SKU: "SKU-001", Warehouses: make(map[string]models.WarehouseStock)})
	e.SetInventory(&models.Inventory{SKU: "SKU-002", Warehouses: make(map[string]models.WarehouseStock)})
	e.SetInventory(&models.Inventory{SKU: "SKU-003", Warehouses: make(map[string]models.WarehouseStock)})

	all := e.GetAllInventory()
	if len(all) != 3 {
		t.Errorf("expected 3 inventory items, got %d", len(all))
	}
}

func TestEngine_AdjustStock(t *testing.T) {
	e := NewEngine()
	ctx := context.Background()

	// Adjust for new SKU
	adj := &models.InventoryAdjustment{
		SKU:         "TEST-001",
		WarehouseID: "WH-001",
		Quantity:    100,
		Reason:      models.AdjustmentReasonReceived,
	}

	err := e.AdjustStock(ctx, adj)
	if err != nil {
		t.Fatalf("AdjustStock failed: %v", err)
	}

	inv, ok := e.GetInventory("TEST-001")
	if !ok {
		t.Fatal("expected inventory to be created")
	}
	if inv.TotalStock != 100 {
		t.Errorf("TotalStock = %d, want 100", inv.TotalStock)
	}

	// Adjust negative
	adj2 := &models.InventoryAdjustment{
		SKU:         "TEST-001",
		WarehouseID: "WH-001",
		Quantity:    -30,
		Reason:      models.AdjustmentReasonSold,
	}

	err = e.AdjustStock(ctx, adj2)
	if err != nil {
		t.Fatalf("AdjustStock failed: %v", err)
	}

	inv, _ = e.GetInventory("TEST-001")
	if inv.TotalStock != 70 {
		t.Errorf("TotalStock = %d, want 70", inv.TotalStock)
	}

	// Adjust below zero (should cap at 0)
	adj3 := &models.InventoryAdjustment{
		SKU:         "TEST-001",
		WarehouseID: "WH-001",
		Quantity:    -200,
		Reason:      models.AdjustmentReasonCorrection,
	}

	err = e.AdjustStock(ctx, adj3)
	if err != nil {
		t.Fatalf("AdjustStock failed: %v", err)
	}

	inv, _ = e.GetInventory("TEST-001")
	if inv.TotalStock != 0 {
		t.Errorf("TotalStock = %d, want 0", inv.TotalStock)
	}
}

func TestEngine_TransferStock(t *testing.T) {
	e := NewEngine()
	ctx := context.Background()

	// Setup inventory
	inv := &models.Inventory{
		SKU:        "TEST-001",
		TotalStock: 100,
		Available:  100,
		Warehouses: map[string]models.WarehouseStock{
			"WH-001": {WarehouseID: "WH-001", Quantity: 100, Available: 100},
			"WH-002": {WarehouseID: "WH-002", Quantity: 0, Available: 0},
		},
	}
	e.SetInventory(inv)

	// Transfer
	err := e.TransferStock(ctx, "TEST-001", "WH-001", "WH-002", 30)
	if err != nil {
		t.Fatalf("TransferStock failed: %v", err)
	}

	inv, _ = e.GetInventory("TEST-001")
	if inv.Warehouses["WH-001"].Quantity != 70 {
		t.Errorf("WH-001 quantity = %d, want 70", inv.Warehouses["WH-001"].Quantity)
	}
	if inv.Warehouses["WH-002"].Quantity != 30 {
		t.Errorf("WH-002 quantity = %d, want 30", inv.Warehouses["WH-002"].Quantity)
	}

	// Transfer non-existent SKU
	err = e.TransferStock(ctx, "NONEXISTENT", "WH-001", "WH-002", 10)
	if err == nil {
		t.Error("expected error for non-existent SKU")
	}

	// Transfer insufficient stock
	err = e.TransferStock(ctx, "TEST-001", "WH-001", "WH-002", 1000)
	if err == nil {
		t.Error("expected error for insufficient stock")
	}
}

func TestEngine_ReserveStock(t *testing.T) {
	e := NewEngine()
	ctx := context.Background()

	// Setup inventory
	inv := &models.Inventory{
		SKU:        "TEST-001",
		TotalStock: 100,
		Available:  100,
		Reserved:   0,
		Warehouses: map[string]models.WarehouseStock{
			"WH-001": {WarehouseID: "WH-001", Quantity: 100, Available: 100, Reserved: 0},
		},
	}
	e.SetInventory(inv)

	// Reserve stock
	err := e.ReserveStock(ctx, "TEST-001", "WH-001", 20)
	if err != nil {
		t.Fatalf("ReserveStock failed: %v", err)
	}

	inv, _ = e.GetInventory("TEST-001")
	if inv.Reserved != 20 {
		t.Errorf("Reserved = %d, want 20", inv.Reserved)
	}
	if inv.Available != 80 {
		t.Errorf("Available = %d, want 80", inv.Available)
	}

	// Reserve non-existent SKU
	err = e.ReserveStock(ctx, "NONEXISTENT", "WH-001", 10)
	if err == nil {
		t.Error("expected error for non-existent SKU")
	}

	// Reserve non-existent warehouse
	err = e.ReserveStock(ctx, "TEST-001", "WH-NONEXISTENT", 10)
	if err == nil {
		t.Error("expected error for non-existent warehouse")
	}

	// Reserve more than available
	err = e.ReserveStock(ctx, "TEST-001", "WH-001", 1000)
	if err == nil {
		t.Error("expected error for insufficient available stock")
	}
}

func TestEngine_ReleaseReservation(t *testing.T) {
	e := NewEngine()
	ctx := context.Background()

	// Setup inventory with reservation
	inv := &models.Inventory{
		SKU:        "TEST-001",
		TotalStock: 100,
		Available:  80,
		Reserved:   20,
		Warehouses: map[string]models.WarehouseStock{
			"WH-001": {WarehouseID: "WH-001", Quantity: 100, Available: 80, Reserved: 20},
		},
	}
	e.SetInventory(inv)

	// Release reservation
	err := e.ReleaseReservation(ctx, "TEST-001", "WH-001", 10)
	if err != nil {
		t.Fatalf("ReleaseReservation failed: %v", err)
	}

	inv, _ = e.GetInventory("TEST-001")
	if inv.Reserved != 10 {
		t.Errorf("Reserved = %d, want 10", inv.Reserved)
	}
	if inv.Available != 90 {
		t.Errorf("Available = %d, want 90", inv.Available)
	}

	// Release more than reserved (should cap at 0)
	err = e.ReleaseReservation(ctx, "TEST-001", "WH-001", 100)
	if err != nil {
		t.Fatalf("ReleaseReservation failed: %v", err)
	}

	inv, _ = e.GetInventory("TEST-001")
	if inv.Reserved != 0 {
		t.Errorf("Reserved = %d, want 0", inv.Reserved)
	}

	// Release non-existent SKU
	err = e.ReleaseReservation(ctx, "NONEXISTENT", "WH-001", 10)
	if err == nil {
		t.Error("expected error for non-existent SKU")
	}

	// Release non-existent warehouse
	err = e.ReleaseReservation(ctx, "TEST-001", "WH-NONEXISTENT", 10)
	if err == nil {
		t.Error("expected error for non-existent warehouse")
	}
}

func TestEngine_GetLowStockItems(t *testing.T) {
	e := NewEngine()

	e.SetInventory(&models.Inventory{SKU: "SKU-001", Available: 5, ReorderPoint: 10, Warehouses: make(map[string]models.WarehouseStock)})
	e.SetInventory(&models.Inventory{SKU: "SKU-002", Available: 100, ReorderPoint: 10, Warehouses: make(map[string]models.WarehouseStock)})
	e.SetInventory(&models.Inventory{SKU: "SKU-003", Available: 8, ReorderPoint: 20, Warehouses: make(map[string]models.WarehouseStock)})
	e.SetInventory(&models.Inventory{SKU: "SKU-004", Available: 0, ReorderPoint: 5, Warehouses: make(map[string]models.WarehouseStock)})

	lowStock := e.GetLowStockItems(10)

	// Should include SKU-001 (below threshold), SKU-003 (below reorder point), SKU-004 (zero)
	if len(lowStock) < 3 {
		t.Errorf("expected at least 3 low stock items, got %d", len(lowStock))
	}
}

func TestEngine_SyncToChannels_NotFound(t *testing.T) {
	e := NewEngine()
	ctx := context.Background()

	err := e.SyncToChannels(ctx, "NONEXISTENT")
	if err == nil {
		t.Error("expected error for non-existent SKU")
	}
}

func TestEngine_CheckAlerts(t *testing.T) {
	tests := []struct {
		name        string
		inv         *models.Inventory
		wantAlert   bool
		alertType   models.AlertType
	}{
		{
			name: "out of stock",
			inv: &models.Inventory{
				SKU:       "TEST-001",
				Available: 0,
			},
			wantAlert: true,
			alertType: models.AlertTypeOutOfStock,
		},
		{
			name: "low stock",
			inv: &models.Inventory{
				SKU:          "TEST-002",
				Available:    5,
				ReorderPoint: 10,
			},
			wantAlert: true,
			alertType: models.AlertTypeLowStock,
		},
		{
			name: "healthy stock",
			inv: &models.Inventory{
				SKU:          "TEST-003",
				Available:    100,
				ReorderPoint: 10,
			},
			wantAlert: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEngine()

			var receivedAlert *models.Alert
			e.SetAlertCallback(func(a *models.Alert) {
				receivedAlert = a
			})

			tt.inv.Warehouses = make(map[string]models.WarehouseStock)
			e.checkAlerts(tt.inv)

			if tt.wantAlert {
				if receivedAlert == nil {
					t.Error("expected alert to be generated")
				} else if receivedAlert.Type != tt.alertType {
					t.Errorf("alert type = %s, want %s", receivedAlert.Type, tt.alertType)
				}
			} else {
				if receivedAlert != nil {
					t.Error("expected no alert")
				}
			}
		})
	}
}

func TestEventType_Constants(t *testing.T) {
	types := []EventType{
		EventTypeAdjustment,
		EventTypeTransfer,
		EventTypeSale,
		EventTypeReturn,
		EventTypeSync,
	}

	for _, et := range types {
		if et == "" {
			t.Error("event type should not be empty")
		}
	}
}
