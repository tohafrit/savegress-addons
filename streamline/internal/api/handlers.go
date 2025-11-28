package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/savegress/streamline/internal/inventory"
	"github.com/savegress/streamline/internal/orders"
	"github.com/savegress/streamline/internal/pricing"
	"github.com/savegress/streamline/pkg/models"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	inventory *inventory.Engine
	orders    *orders.Processor
	pricing   *pricing.Engine
}

// NewHandlers creates new handlers
func NewHandlers(inv *inventory.Engine, ord *orders.Processor, pr *pricing.Engine) *Handlers {
	return &Handlers{
		inventory: inv,
		orders:    ord,
		pricing:   pr,
	}
}

// Response helpers
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Success: true, Data: data})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Success: false, Error: message})
}

// Health check
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// Inventory handlers

func (h *Handlers) GetAllInventory(w http.ResponseWriter, r *http.Request) {
	inventory := h.inventory.GetAllInventory()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"inventory": inventory,
		"count":     len(inventory),
	})
}

func (h *Handlers) GetInventory(w http.ResponseWriter, r *http.Request) {
	sku := chi.URLParam(r, "sku")

	inv, ok := h.inventory.GetInventory(sku)
	if !ok {
		writeError(w, http.StatusNotFound, "SKU not found")
		return
	}

	writeJSON(w, http.StatusOK, inv)
}

func (h *Handlers) UpdateInventory(w http.ResponseWriter, r *http.Request) {
	sku := chi.URLParam(r, "sku")

	var req models.Inventory
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.SKU = sku
	h.inventory.SetInventory(&req)

	writeJSON(w, http.StatusOK, req)
}

func (h *Handlers) AdjustInventory(w http.ResponseWriter, r *http.Request) {
	sku := chi.URLParam(r, "sku")

	var adj models.InventoryAdjustment
	if err := json.NewDecoder(r.Body).Decode(&adj); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	adj.SKU = sku
	if err := h.inventory.AdjustStock(r.Context(), &adj); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "adjusted"})
}

func (h *Handlers) TransferInventory(w http.ResponseWriter, r *http.Request) {
	sku := chi.URLParam(r, "sku")

	var req struct {
		FromWarehouse string `json:"from_warehouse"`
		ToWarehouse   string `json:"to_warehouse"`
		Quantity      int    `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.inventory.TransferStock(r.Context(), sku, req.FromWarehouse, req.ToWarehouse, req.Quantity); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "transferred"})
}

func (h *Handlers) GetLowStock(w http.ResponseWriter, r *http.Request) {
	thresholdStr := r.URL.Query().Get("threshold")
	threshold := 15
	if thresholdStr != "" {
		if t, err := strconv.Atoi(thresholdStr); err == nil {
			threshold = t
		}
	}

	items := h.inventory.GetLowStockItems(threshold)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"count": len(items),
	})
}

// Order handlers

func (h *Handlers) ListOrders(w http.ResponseWriter, r *http.Request) {
	filter := orders.OrderFilter{}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = models.OrderStatus(status)
	}
	if channel := r.URL.Query().Get("channel"); channel != "" {
		filter.Channel = channel
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = l
		}
	}

	orderList := h.orders.ListOrders(filter)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"orders": orderList,
		"count":  len(orderList),
	})
}

func (h *Handlers) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	order, ok := h.orders.GetOrder(id)
	if !ok {
		writeError(w, http.StatusNotFound, "Order not found")
		return
	}

	writeJSON(w, http.StatusOK, order)
}

func (h *Handlers) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var order models.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.orders.CreateOrder(r.Context(), &order); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, order)
}

func (h *Handlers) RouteOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Get inventory info (simplified - would come from actual inventory)
	invInfo := make(map[string]*orders.InventoryInfo)

	decision, err := h.orders.RouteOrder(r.Context(), id, invInfo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, decision)
}

func (h *Handlers) AssignWarehouse(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		WarehouseID string `json:"warehouse_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.orders.AssignWarehouse(r.Context(), id, req.WarehouseID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "assigned"})
}

func (h *Handlers) FulfillOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var fulfillment models.Fulfillment
	if err := json.NewDecoder(r.Body).Decode(&fulfillment); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.orders.FulfillOrder(r.Context(), id, &fulfillment); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "fulfilled"})
}

func (h *Handlers) CancelOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.orders.CancelOrder(r.Context(), id, req.Reason); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (h *Handlers) GetOrderStats(w http.ResponseWriter, r *http.Request) {
	stats := h.orders.GetStats()
	writeJSON(w, http.StatusOK, stats)
}

// Pricing handlers

func (h *Handlers) GetPricingRules(w http.ResponseWriter, r *http.Request) {
	rules := h.pricing.GetRules()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rules": rules,
		"count": len(rules),
	})
}

func (h *Handlers) CreatePricingRule(w http.ResponseWriter, r *http.Request) {
	var rule models.PricingRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	h.pricing.AddRule(&rule)
	writeJSON(w, http.StatusCreated, rule)
}

func (h *Handlers) DeletePricingRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.pricing.RemoveRule(id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Stats/Overview
func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	orderStats := h.orders.GetStats()
	inventoryList := h.inventory.GetAllInventory()
	lowStock := h.inventory.GetLowStockItems(15)

	stats := map[string]interface{}{
		"orders":        orderStats,
		"inventory": map[string]interface{}{
			"total_skus": len(inventoryList),
			"low_stock":  len(lowStock),
		},
		"pricing_rules": len(h.pricing.GetRules()),
	}

	writeJSON(w, http.StatusOK, stats)
}
