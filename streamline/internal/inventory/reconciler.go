package inventory

import (
	"context"
	"fmt"
	"time"

	"github.com/savegress/streamline/pkg/models"
)

// Reconciler handles inventory discrepancy detection and resolution
type Reconciler struct {
	discrepancyThreshold float64 // Percentage threshold for flagging discrepancies
}

// NewReconciler creates a new reconciler
func NewReconciler() *Reconciler {
	return &Reconciler{
		discrepancyThreshold: 5.0, // 5% default threshold
	}
}

// SetDiscrepancyThreshold sets the threshold for flagging discrepancies
func (r *Reconciler) SetDiscrepancyThreshold(threshold float64) {
	r.discrepancyThreshold = threshold
}

// Discrepancy represents an inventory discrepancy
type Discrepancy struct {
	SKU           string    `json:"sku"`
	ChannelID     string    `json:"channel_id"`
	ExpectedQty   int       `json:"expected_qty"`
	ActualQty     int       `json:"actual_qty"`
	Difference    int       `json:"difference"`
	PercentageDiff float64  `json:"percentage_diff"`
	DetectedAt    time.Time `json:"detected_at"`
	Resolved      bool      `json:"resolved"`
	Resolution    string    `json:"resolution,omitempty"`
}

// ReconciliationResult represents the result of a reconciliation
type ReconciliationResult struct {
	SKU           string        `json:"sku"`
	ChannelID     string        `json:"channel_id"`
	Success       bool          `json:"success"`
	Discrepancy   *Discrepancy  `json:"discrepancy,omitempty"`
	Action        string        `json:"action"`
	Error         string        `json:"error,omitempty"`
	ReconcileTime time.Time     `json:"reconcile_time"`
}

// ReconcileChannel reconciles inventory with a channel
func (r *Reconciler) ReconcileChannel(ctx context.Context, inv *models.Inventory, handler ChannelHandler) (*ReconciliationResult, error) {
	channelID := handler.GetChannelID()

	result := &ReconciliationResult{
		SKU:           inv.SKU,
		ChannelID:     channelID,
		ReconcileTime: time.Now(),
	}

	// Get expected quantity (what we think the channel has)
	expectedQty := inv.Allocated[channelID]

	// Get actual quantity from channel
	actualQty, err := handler.GetInventory(ctx, inv.SKU)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, err
	}

	// Check for discrepancy
	diff := actualQty - expectedQty
	var percentDiff float64
	if expectedQty > 0 {
		percentDiff = float64(abs(diff)) / float64(expectedQty) * 100
	} else if actualQty > 0 {
		percentDiff = 100 // 100% difference if expected is 0 but actual isn't
	}

	if percentDiff > r.discrepancyThreshold {
		result.Discrepancy = &Discrepancy{
			SKU:            inv.SKU,
			ChannelID:      channelID,
			ExpectedQty:    expectedQty,
			ActualQty:      actualQty,
			Difference:     diff,
			PercentageDiff: percentDiff,
			DetectedAt:     time.Now(),
		}
		result.Action = "discrepancy_detected"
		result.Success = true
	} else {
		result.Success = true
		result.Action = "no_action_needed"
	}

	return result, nil
}

// ResolveDiscrepancy resolves a discrepancy
func (r *Reconciler) ResolveDiscrepancy(ctx context.Context, disc *Discrepancy, resolution ResolutionStrategy, handler ChannelHandler) error {
	switch resolution {
	case ResolutionUseSystem:
		// Push system value to channel
		return handler.UpdateInventory(ctx, disc.SKU, disc.ExpectedQty)

	case ResolutionUseChannel:
		// Accept channel value (will need to update system)
		disc.Resolved = true
		disc.Resolution = fmt.Sprintf("Accepted channel value: %d", disc.ActualQty)
		return nil

	case ResolutionManual:
		// Flag for manual resolution
		disc.Resolution = "Flagged for manual resolution"
		return nil

	default:
		return fmt.Errorf("unknown resolution strategy: %s", resolution)
	}
}

// ResolutionStrategy defines how to resolve discrepancies
type ResolutionStrategy string

const (
	ResolutionUseSystem  ResolutionStrategy = "use_system"  // Push system value to channel
	ResolutionUseChannel ResolutionStrategy = "use_channel" // Accept channel value
	ResolutionManual     ResolutionStrategy = "manual"      // Flag for manual resolution
)

// BatchReconcile reconciles inventory across multiple channels
func (r *Reconciler) BatchReconcile(ctx context.Context, inventory []*models.Inventory, handlers map[string]ChannelHandler) ([]*ReconciliationResult, error) {
	var results []*ReconciliationResult

	for _, inv := range inventory {
		for channelID, handler := range handlers {
			// Only reconcile if we have allocation for this channel
			if _, ok := inv.Allocated[channelID]; !ok {
				continue
			}

			result, err := r.ReconcileChannel(ctx, inv, handler)
			if err != nil {
				result.Error = err.Error()
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// DetectOversold detects oversold situations
func (r *Reconciler) DetectOversold(inv *models.Inventory, pendingOrders []PendingOrder) *OversoldResult {
	result := &OversoldResult{
		SKU:          inv.SKU,
		TotalStock:   inv.Available,
		TotalPending: 0,
	}

	for _, order := range pendingOrders {
		result.TotalPending += order.Quantity
		result.PendingOrders = append(result.PendingOrders, order)
	}

	if result.TotalPending > result.TotalStock {
		result.IsOversold = true
		result.OversoldQty = result.TotalPending - result.TotalStock
	}

	return result
}

// PendingOrder represents a pending order for oversold detection
type PendingOrder struct {
	OrderID   string `json:"order_id"`
	ChannelID string `json:"channel_id"`
	SKU       string `json:"sku"`
	Quantity  int    `json:"quantity"`
	OrderTime time.Time `json:"order_time"`
}

// OversoldResult represents the result of oversold detection
type OversoldResult struct {
	SKU           string         `json:"sku"`
	IsOversold    bool           `json:"is_oversold"`
	TotalStock    int            `json:"total_stock"`
	TotalPending  int            `json:"total_pending"`
	OversoldQty   int            `json:"oversold_qty"`
	PendingOrders []PendingOrder `json:"pending_orders"`
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
