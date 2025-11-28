package inventory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/savegress/streamline/pkg/models"
)

func TestNewReconciler(t *testing.T) {
	r := NewReconciler()

	if r == nil {
		t.Fatal("expected non-nil reconciler")
	}
	if r.discrepancyThreshold != 5.0 {
		t.Errorf("default threshold = %v, want 5.0", r.discrepancyThreshold)
	}
}

func TestReconciler_SetDiscrepancyThreshold(t *testing.T) {
	r := NewReconciler()

	r.SetDiscrepancyThreshold(10.0)

	if r.discrepancyThreshold != 10.0 {
		t.Errorf("threshold = %v, want 10.0", r.discrepancyThreshold)
	}
}

func TestReconciler_ReconcileChannel_NoDiscrepancy(t *testing.T) {
	r := NewReconciler()
	ctx := context.Background()

	handler := &mockChannelHandler{
		channelID: "shopify",
		inventory: map[string]int{"SKU-001": 100},
	}

	inv := &models.Inventory{
		SKU:       "SKU-001",
		Allocated: map[string]int{"shopify": 100},
	}

	result, err := r.ReconcileChannel(ctx, inv, handler)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Discrepancy != nil {
		t.Error("expected no discrepancy")
	}
	if result.Action != "no_action_needed" {
		t.Errorf("action = %s, want no_action_needed", result.Action)
	}
}

func TestReconciler_ReconcileChannel_WithDiscrepancy(t *testing.T) {
	r := NewReconciler()
	r.SetDiscrepancyThreshold(5.0) // 5% threshold
	ctx := context.Background()

	handler := &mockChannelHandler{
		channelID: "shopify",
		inventory: map[string]int{"SKU-001": 80}, // 20% different from expected 100
	}

	inv := &models.Inventory{
		SKU:       "SKU-001",
		Allocated: map[string]int{"shopify": 100},
	}

	result, err := r.ReconcileChannel(ctx, inv, handler)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Discrepancy == nil {
		t.Fatal("expected discrepancy to be detected")
	}
	if result.Discrepancy.ExpectedQty != 100 {
		t.Errorf("ExpectedQty = %d, want 100", result.Discrepancy.ExpectedQty)
	}
	if result.Discrepancy.ActualQty != 80 {
		t.Errorf("ActualQty = %d, want 80", result.Discrepancy.ActualQty)
	}
	if result.Discrepancy.Difference != -20 {
		t.Errorf("Difference = %d, want -20", result.Discrepancy.Difference)
	}
	if result.Action != "discrepancy_detected" {
		t.Errorf("action = %s, want discrepancy_detected", result.Action)
	}
}

func TestReconciler_ReconcileChannel_Error(t *testing.T) {
	r := NewReconciler()
	ctx := context.Background()

	handler := &mockChannelHandler{
		channelID: "shopify",
		getErr:    errors.New("api error"),
	}

	inv := &models.Inventory{
		SKU:       "SKU-001",
		Allocated: map[string]int{"shopify": 100},
	}

	result, err := r.ReconcileChannel(ctx, inv, handler)

	if err == nil {
		t.Error("expected error")
	}
	if result.Success {
		t.Error("expected failure")
	}
	if result.Error != "api error" {
		t.Errorf("Error = %s, want 'api error'", result.Error)
	}
}

func TestReconciler_ReconcileChannel_ZeroExpected(t *testing.T) {
	r := NewReconciler()
	ctx := context.Background()

	handler := &mockChannelHandler{
		channelID: "shopify",
		inventory: map[string]int{"SKU-001": 50}, // 100% different since expected is 0
	}

	inv := &models.Inventory{
		SKU:       "SKU-001",
		Allocated: map[string]int{"shopify": 0},
	}

	result, err := r.ReconcileChannel(ctx, inv, handler)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Discrepancy == nil {
		t.Fatal("expected discrepancy when expected is 0 but actual is not")
	}
	if result.Discrepancy.PercentageDiff != 100 {
		t.Errorf("PercentageDiff = %v, want 100", result.Discrepancy.PercentageDiff)
	}
}

func TestReconciler_ReconcileChannel_BothZero(t *testing.T) {
	r := NewReconciler()
	ctx := context.Background()

	handler := &mockChannelHandler{
		channelID: "shopify",
		inventory: map[string]int{"SKU-001": 0},
	}

	inv := &models.Inventory{
		SKU:       "SKU-001",
		Allocated: map[string]int{"shopify": 0},
	}

	result, err := r.ReconcileChannel(ctx, inv, handler)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Discrepancy != nil {
		t.Error("expected no discrepancy when both are zero")
	}
}

func TestReconciler_ResolveDiscrepancy_UseSystem(t *testing.T) {
	r := NewReconciler()
	ctx := context.Background()

	handler := &mockChannelHandler{
		channelID: "shopify",
		inventory: make(map[string]int),
	}

	disc := &Discrepancy{
		SKU:         "SKU-001",
		ChannelID:   "shopify",
		ExpectedQty: 100,
		ActualQty:   80,
	}

	err := r.ResolveDiscrepancy(ctx, disc, ResolutionUseSystem, handler)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if handler.inventory["SKU-001"] != 100 {
		t.Errorf("inventory = %d, want 100", handler.inventory["SKU-001"])
	}
}

func TestReconciler_ResolveDiscrepancy_UseChannel(t *testing.T) {
	r := NewReconciler()
	ctx := context.Background()

	handler := &mockChannelHandler{
		channelID: "shopify",
	}

	disc := &Discrepancy{
		SKU:         "SKU-001",
		ChannelID:   "shopify",
		ExpectedQty: 100,
		ActualQty:   80,
	}

	err := r.ResolveDiscrepancy(ctx, disc, ResolutionUseChannel, handler)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !disc.Resolved {
		t.Error("discrepancy should be marked as resolved")
	}
	if disc.Resolution == "" {
		t.Error("resolution should be set")
	}
}

func TestReconciler_ResolveDiscrepancy_Manual(t *testing.T) {
	r := NewReconciler()
	ctx := context.Background()

	handler := &mockChannelHandler{
		channelID: "shopify",
	}

	disc := &Discrepancy{
		SKU:         "SKU-001",
		ChannelID:   "shopify",
		ExpectedQty: 100,
		ActualQty:   80,
	}

	err := r.ResolveDiscrepancy(ctx, disc, ResolutionManual, handler)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if disc.Resolution != "Flagged for manual resolution" {
		t.Errorf("Resolution = %s, want 'Flagged for manual resolution'", disc.Resolution)
	}
}

func TestReconciler_ResolveDiscrepancy_Unknown(t *testing.T) {
	r := NewReconciler()
	ctx := context.Background()

	handler := &mockChannelHandler{
		channelID: "shopify",
	}

	disc := &Discrepancy{
		SKU: "SKU-001",
	}

	err := r.ResolveDiscrepancy(ctx, disc, "invalid", handler)

	if err == nil {
		t.Error("expected error for unknown resolution strategy")
	}
}

func TestReconciler_ResolveDiscrepancy_UpdateError(t *testing.T) {
	r := NewReconciler()
	ctx := context.Background()

	handler := &mockChannelHandler{
		channelID: "shopify",
		updateErr: errors.New("update failed"),
	}

	disc := &Discrepancy{
		SKU:         "SKU-001",
		ExpectedQty: 100,
	}

	err := r.ResolveDiscrepancy(ctx, disc, ResolutionUseSystem, handler)

	if err == nil {
		t.Error("expected error when update fails")
	}
}

func TestReconciler_BatchReconcile(t *testing.T) {
	r := NewReconciler()
	ctx := context.Background()

	handlers := map[string]ChannelHandler{
		"shopify": &mockChannelHandler{
			channelID: "shopify",
			inventory: map[string]int{"SKU-001": 100, "SKU-002": 50},
		},
		"amazon": &mockChannelHandler{
			channelID: "amazon",
			inventory: map[string]int{"SKU-001": 80, "SKU-002": 50},
		},
	}

	inventory := []*models.Inventory{
		{
			SKU:       "SKU-001",
			Allocated: map[string]int{"shopify": 100, "amazon": 100},
		},
		{
			SKU:       "SKU-002",
			Allocated: map[string]int{"shopify": 50, "amazon": 50},
		},
	}

	results, err := r.BatchReconcile(ctx, inventory, handlers)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 4 { // 2 SKUs x 2 channels
		t.Errorf("expected 4 results, got %d", len(results))
	}
}

func TestReconciler_BatchReconcile_SkipUnallocated(t *testing.T) {
	r := NewReconciler()
	ctx := context.Background()

	handlers := map[string]ChannelHandler{
		"shopify": &mockChannelHandler{
			channelID: "shopify",
			inventory: map[string]int{"SKU-001": 100},
		},
		"amazon": &mockChannelHandler{
			channelID: "amazon",
			inventory: map[string]int{},
		},
	}

	inventory := []*models.Inventory{
		{
			SKU:       "SKU-001",
			Allocated: map[string]int{"shopify": 100}, // Only allocated to shopify
		},
	}

	results, err := r.BatchReconcile(ctx, inventory, handlers)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 { // Only shopify should be reconciled
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestReconciler_DetectOversold_NotOversold(t *testing.T) {
	r := NewReconciler()

	inv := &models.Inventory{
		SKU:       "SKU-001",
		Available: 100,
	}

	orders := []PendingOrder{
		{OrderID: "ORD-001", SKU: "SKU-001", Quantity: 30},
		{OrderID: "ORD-002", SKU: "SKU-001", Quantity: 20},
	}

	result := r.DetectOversold(inv, orders)

	if result.IsOversold {
		t.Error("should not be oversold")
	}
	if result.TotalStock != 100 {
		t.Errorf("TotalStock = %d, want 100", result.TotalStock)
	}
	if result.TotalPending != 50 {
		t.Errorf("TotalPending = %d, want 50", result.TotalPending)
	}
	if result.OversoldQty != 0 {
		t.Errorf("OversoldQty = %d, want 0", result.OversoldQty)
	}
}

func TestReconciler_DetectOversold_IsOversold(t *testing.T) {
	r := NewReconciler()

	inv := &models.Inventory{
		SKU:       "SKU-001",
		Available: 50,
	}

	orders := []PendingOrder{
		{OrderID: "ORD-001", SKU: "SKU-001", Quantity: 40},
		{OrderID: "ORD-002", SKU: "SKU-001", Quantity: 30},
	}

	result := r.DetectOversold(inv, orders)

	if !result.IsOversold {
		t.Error("should be oversold")
	}
	if result.TotalPending != 70 {
		t.Errorf("TotalPending = %d, want 70", result.TotalPending)
	}
	if result.OversoldQty != 20 {
		t.Errorf("OversoldQty = %d, want 20", result.OversoldQty)
	}
	if len(result.PendingOrders) != 2 {
		t.Errorf("PendingOrders length = %d, want 2", len(result.PendingOrders))
	}
}

func TestReconciler_DetectOversold_NoPendingOrders(t *testing.T) {
	r := NewReconciler()

	inv := &models.Inventory{
		SKU:       "SKU-001",
		Available: 100,
	}

	result := r.DetectOversold(inv, nil)

	if result.IsOversold {
		t.Error("should not be oversold with no orders")
	}
	if result.TotalPending != 0 {
		t.Errorf("TotalPending = %d, want 0", result.TotalPending)
	}
}

func TestResolutionStrategy_Constants(t *testing.T) {
	if ResolutionUseSystem != "use_system" {
		t.Errorf("ResolutionUseSystem = %s, want use_system", ResolutionUseSystem)
	}
	if ResolutionUseChannel != "use_channel" {
		t.Errorf("ResolutionUseChannel = %s, want use_channel", ResolutionUseChannel)
	}
	if ResolutionManual != "manual" {
		t.Errorf("ResolutionManual = %s, want manual", ResolutionManual)
	}
}

func TestDiscrepancy_Struct(t *testing.T) {
	now := time.Now()
	disc := &Discrepancy{
		SKU:            "SKU-001",
		ChannelID:      "shopify",
		ExpectedQty:    100,
		ActualQty:      80,
		Difference:     -20,
		PercentageDiff: 20.0,
		DetectedAt:     now,
		Resolved:       false,
		Resolution:     "",
	}

	if disc.SKU != "SKU-001" {
		t.Error("SKU not set correctly")
	}
	if disc.Difference != -20 {
		t.Error("Difference not set correctly")
	}
}

func TestPendingOrder_Struct(t *testing.T) {
	now := time.Now()
	order := PendingOrder{
		OrderID:   "ORD-001",
		ChannelID: "shopify",
		SKU:       "SKU-001",
		Quantity:  5,
		OrderTime: now,
	}

	if order.OrderID != "ORD-001" {
		t.Error("OrderID not set correctly")
	}
	if order.Quantity != 5 {
		t.Error("Quantity not set correctly")
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{-100, 100},
	}

	for _, tt := range tests {
		got := abs(tt.input)
		if got != tt.want {
			t.Errorf("abs(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// mockChannelHandler is defined in engine_test.go
