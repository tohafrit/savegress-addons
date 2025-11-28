package inventory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/savegress/streamline/pkg/models"
)

// Engine manages inventory synchronization across channels
type Engine struct {
	allocator  *Allocator
	reconciler *Reconciler

	// In-memory inventory state
	inventory   map[string]*models.Inventory // SKU -> Inventory
	inventoryMu sync.RWMutex

	// Warehouses
	warehouses   map[string]*models.Warehouse
	warehousesMu sync.RWMutex

	// Channel sync handlers
	channelHandlers map[string]ChannelHandler
	handlersMu      sync.RWMutex

	// Event channel
	eventChan chan InventoryEvent
	stopChan  chan struct{}
	wg        sync.WaitGroup

	// Callbacks
	onAlert func(*models.Alert)
}

// ChannelHandler interface for channel-specific inventory operations
type ChannelHandler interface {
	GetInventory(ctx context.Context, sku string) (int, error)
	UpdateInventory(ctx context.Context, sku string, quantity int) error
	GetChannelID() string
}

// InventoryEvent represents an inventory change event
type InventoryEvent struct {
	Type        EventType
	SKU         string
	WarehouseID string
	ChannelID   string
	Quantity    int
	PrevQty     int
	Reason      models.AdjustmentReason
	Reference   string
	Timestamp   time.Time
}

type EventType string

const (
	EventTypeAdjustment EventType = "adjustment"
	EventTypeTransfer   EventType = "transfer"
	EventTypeSale       EventType = "sale"
	EventTypeReturn     EventType = "return"
	EventTypeSync       EventType = "sync"
)

// NewEngine creates a new inventory engine
func NewEngine() *Engine {
	e := &Engine{
		inventory:       make(map[string]*models.Inventory),
		warehouses:      make(map[string]*models.Warehouse),
		channelHandlers: make(map[string]ChannelHandler),
		eventChan:       make(chan InventoryEvent, 1000),
		stopChan:        make(chan struct{}),
	}
	e.allocator = NewAllocator()
	e.reconciler = NewReconciler()
	return e
}

// Start starts the inventory engine
func (e *Engine) Start(ctx context.Context) error {
	e.wg.Add(1)
	go e.processEvents(ctx)

	return nil
}

// Stop stops the inventory engine
func (e *Engine) Stop() {
	close(e.stopChan)
	e.wg.Wait()
}

// SetAlertCallback sets the callback for alerts
func (e *Engine) SetAlertCallback(fn func(*models.Alert)) {
	e.onAlert = fn
}

// RegisterChannel registers a channel handler
func (e *Engine) RegisterChannel(handler ChannelHandler) {
	e.handlersMu.Lock()
	defer e.handlersMu.Unlock()
	e.channelHandlers[handler.GetChannelID()] = handler
}

// AddWarehouse adds a warehouse
func (e *Engine) AddWarehouse(warehouse *models.Warehouse) {
	e.warehousesMu.Lock()
	defer e.warehousesMu.Unlock()
	e.warehouses[warehouse.ID] = warehouse
}

// GetInventory returns inventory for a SKU
func (e *Engine) GetInventory(sku string) (*models.Inventory, bool) {
	e.inventoryMu.RLock()
	defer e.inventoryMu.RUnlock()
	inv, ok := e.inventory[sku]
	return inv, ok
}

// GetAllInventory returns all inventory
func (e *Engine) GetAllInventory() []*models.Inventory {
	e.inventoryMu.RLock()
	defer e.inventoryMu.RUnlock()

	result := make([]*models.Inventory, 0, len(e.inventory))
	for _, inv := range e.inventory {
		result = append(result, inv)
	}
	return result
}

// SetInventory sets the inventory for a SKU
func (e *Engine) SetInventory(inv *models.Inventory) {
	e.inventoryMu.Lock()
	e.inventory[inv.SKU] = inv
	e.inventoryMu.Unlock()

	e.checkAlerts(inv)
}

// AdjustStock adjusts stock for a SKU
func (e *Engine) AdjustStock(ctx context.Context, adj *models.InventoryAdjustment) error {
	e.inventoryMu.Lock()
	defer e.inventoryMu.Unlock()

	inv, ok := e.inventory[adj.SKU]
	if !ok {
		inv = &models.Inventory{
			SKU:        adj.SKU,
			Warehouses: make(map[string]models.WarehouseStock),
			Allocated:  make(map[string]int),
		}
		e.inventory[adj.SKU] = inv
	}

	// Update warehouse stock
	ws, ok := inv.Warehouses[adj.WarehouseID]
	if !ok {
		ws = models.WarehouseStock{
			WarehouseID: adj.WarehouseID,
		}
	}

	prevQty := ws.Quantity
	ws.Quantity += adj.Quantity
	if ws.Quantity < 0 {
		ws.Quantity = 0
	}
	ws.Available = ws.Quantity - ws.Reserved
	inv.Warehouses[adj.WarehouseID] = ws

	// Recalculate totals
	e.recalculateTotals(inv)

	// Emit event
	e.eventChan <- InventoryEvent{
		Type:        EventTypeAdjustment,
		SKU:         adj.SKU,
		WarehouseID: adj.WarehouseID,
		Quantity:    adj.Quantity,
		PrevQty:     prevQty,
		Reason:      adj.Reason,
		Reference:   adj.Reference,
		Timestamp:   time.Now(),
	}

	// Check for alerts
	go e.checkAlerts(inv)

	return nil
}

// TransferStock transfers stock between warehouses
func (e *Engine) TransferStock(ctx context.Context, sku, fromWarehouse, toWarehouse string, quantity int) error {
	e.inventoryMu.Lock()
	defer e.inventoryMu.Unlock()

	inv, ok := e.inventory[sku]
	if !ok {
		return fmt.Errorf("SKU not found: %s", sku)
	}

	fromWS, ok := inv.Warehouses[fromWarehouse]
	if !ok || fromWS.Available < quantity {
		return fmt.Errorf("insufficient stock in warehouse %s", fromWarehouse)
	}

	toWS := inv.Warehouses[toWarehouse]

	// Perform transfer
	fromWS.Quantity -= quantity
	fromWS.Available = fromWS.Quantity - fromWS.Reserved
	inv.Warehouses[fromWarehouse] = fromWS

	toWS.Quantity += quantity
	toWS.Available = toWS.Quantity - toWS.Reserved
	inv.Warehouses[toWarehouse] = toWS

	// Emit event
	e.eventChan <- InventoryEvent{
		Type:        EventTypeTransfer,
		SKU:         sku,
		WarehouseID: toWarehouse,
		Quantity:    quantity,
		Reference:   fromWarehouse,
		Timestamp:   time.Now(),
	}

	return nil
}

// ReserveStock reserves stock for an order
func (e *Engine) ReserveStock(ctx context.Context, sku, warehouseID string, quantity int) error {
	e.inventoryMu.Lock()
	defer e.inventoryMu.Unlock()

	inv, ok := e.inventory[sku]
	if !ok {
		return fmt.Errorf("SKU not found: %s", sku)
	}

	ws, ok := inv.Warehouses[warehouseID]
	if !ok {
		return fmt.Errorf("warehouse not found: %s", warehouseID)
	}

	if ws.Available < quantity {
		return fmt.Errorf("insufficient available stock")
	}

	ws.Reserved += quantity
	ws.Available = ws.Quantity - ws.Reserved
	inv.Warehouses[warehouseID] = ws

	e.recalculateTotals(inv)

	return nil
}

// ReleaseReservation releases a stock reservation
func (e *Engine) ReleaseReservation(ctx context.Context, sku, warehouseID string, quantity int) error {
	e.inventoryMu.Lock()
	defer e.inventoryMu.Unlock()

	inv, ok := e.inventory[sku]
	if !ok {
		return fmt.Errorf("SKU not found: %s", sku)
	}

	ws, ok := inv.Warehouses[warehouseID]
	if !ok {
		return fmt.Errorf("warehouse not found: %s", warehouseID)
	}

	ws.Reserved -= quantity
	if ws.Reserved < 0 {
		ws.Reserved = 0
	}
	ws.Available = ws.Quantity - ws.Reserved
	inv.Warehouses[warehouseID] = ws

	e.recalculateTotals(inv)

	return nil
}

// SyncToChannels syncs inventory to all connected channels
func (e *Engine) SyncToChannels(ctx context.Context, sku string) error {
	inv, ok := e.GetInventory(sku)
	if !ok {
		return fmt.Errorf("SKU not found: %s", sku)
	}

	e.handlersMu.RLock()
	handlers := make([]ChannelHandler, 0, len(e.channelHandlers))
	for _, h := range e.channelHandlers {
		handlers = append(handlers, h)
	}
	e.handlersMu.RUnlock()

	// Allocate inventory across channels
	allocations := e.allocator.Allocate(inv, handlers)

	// Update each channel
	var lastErr error
	for channelID, qty := range allocations {
		if handler, ok := e.channelHandlers[channelID]; ok {
			if err := handler.UpdateInventory(ctx, sku, qty); err != nil {
				lastErr = err
				e.createAlert(models.AlertTypeSyncError, models.AlertSeverityWarning, sku, channelID,
					fmt.Sprintf("Failed to sync inventory: %v", err))
			}
		}
	}

	// Update allocated map
	e.inventoryMu.Lock()
	inv.Allocated = allocations
	inv.LastUpdated = time.Now()
	e.inventoryMu.Unlock()

	return lastErr
}

// GetLowStockItems returns items below reorder point
func (e *Engine) GetLowStockItems(threshold int) []*models.Inventory {
	e.inventoryMu.RLock()
	defer e.inventoryMu.RUnlock()

	var result []*models.Inventory
	for _, inv := range e.inventory {
		if inv.Available <= threshold || inv.Available <= inv.ReorderPoint {
			result = append(result, inv)
		}
	}
	return result
}

func (e *Engine) recalculateTotals(inv *models.Inventory) {
	var total, reserved int
	for _, ws := range inv.Warehouses {
		total += ws.Quantity
		reserved += ws.Reserved
	}
	inv.TotalStock = total
	inv.Reserved = reserved
	inv.Available = total - reserved
	inv.LastUpdated = time.Now()
}

func (e *Engine) processEvents(ctx context.Context) {
	defer e.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case event := <-e.eventChan:
			e.handleEvent(event)
		}
	}
}

func (e *Engine) handleEvent(event InventoryEvent) {
	// Log event, update metrics, trigger webhooks, etc.
	// For now, just check if we need to sync channels
	if event.Type == EventTypeSale || event.Type == EventTypeAdjustment {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			e.SyncToChannels(ctx, event.SKU)
		}()
	}
}

func (e *Engine) checkAlerts(inv *models.Inventory) {
	if e.onAlert == nil {
		return
	}

	// Out of stock
	if inv.Available <= 0 {
		e.createAlert(models.AlertTypeOutOfStock, models.AlertSeverityCritical, inv.SKU, "",
			fmt.Sprintf("SKU %s is out of stock", inv.SKU))
	} else if inv.Available <= inv.ReorderPoint {
		// Low stock
		e.createAlert(models.AlertTypeLowStock, models.AlertSeverityWarning, inv.SKU, "",
			fmt.Sprintf("SKU %s is low on stock (%d available, reorder point: %d)", inv.SKU, inv.Available, inv.ReorderPoint))
	}
}

func (e *Engine) createAlert(alertType models.AlertType, severity models.AlertSeverity, sku, channel, message string) {
	if e.onAlert == nil {
		return
	}

	alert := &models.Alert{
		ID:        fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		Type:      alertType,
		Severity:  severity,
		SKU:       sku,
		Channel:   channel,
		Message:   message,
		CreatedAt: time.Now(),
	}

	e.onAlert(alert)
}
