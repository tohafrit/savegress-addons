package inventory

import (
	"testing"

	"github.com/savegress/streamline/pkg/models"
)

func TestNewAllocator(t *testing.T) {
	a := NewAllocator()

	if a == nil {
		t.Fatal("expected non-nil allocator")
	}
	if a.strategy != StrategyWeighted {
		t.Errorf("default strategy = %s, want weighted", a.strategy)
	}
	if a.globalBuffer != 10.0 {
		t.Errorf("default globalBuffer = %v, want 10.0", a.globalBuffer)
	}
	if a.allocations == nil {
		t.Error("allocations map should be initialized")
	}
}

func TestAllocator_SetStrategy(t *testing.T) {
	a := NewAllocator()

	strategies := []AllocationStrategy{
		StrategyFixed,
		StrategyWeighted,
		StrategyPriority,
		StrategyDynamic,
	}

	for _, s := range strategies {
		a.SetStrategy(s)
		if a.strategy != s {
			t.Errorf("strategy = %s, want %s", a.strategy, s)
		}
	}
}

func TestAllocator_SetChannelAllocation(t *testing.T) {
	a := NewAllocator()

	config := &ChannelAllocation{
		ChannelID: "shopify",
		Weight:    50.0,
		Priority:  1,
		MinStock:  10,
		MaxStock:  100,
		Buffer:    5.0,
		FixedQty:  50,
	}

	a.SetChannelAllocation(config)

	got, ok := a.allocations["shopify"]
	if !ok {
		t.Fatal("allocation not found")
	}
	if got.Weight != 50.0 {
		t.Errorf("Weight = %v, want 50.0", got.Weight)
	}
	if got.Priority != 1 {
		t.Errorf("Priority = %d, want 1", got.Priority)
	}
}

func TestAllocator_SetGlobalBuffer(t *testing.T) {
	a := NewAllocator()

	a.SetGlobalBuffer(15.0)

	if a.globalBuffer != 15.0 {
		t.Errorf("globalBuffer = %v, want 15.0", a.globalBuffer)
	}
}

func TestAllocator_Allocate_Weighted(t *testing.T) {
	a := NewAllocator()
	a.SetStrategy(StrategyWeighted)
	a.SetGlobalBuffer(0) // No buffer for easier testing

	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "shopify",
		Weight:    60.0,
	})
	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "amazon",
		Weight:    40.0,
	})

	inv := &models.Inventory{
		SKU:       "TEST-001",
		Available: 100,
	}

	channels := []ChannelHandler{
		&mockChannelHandler{channelID: "shopify"},
		&mockChannelHandler{channelID: "amazon"},
	}

	result := a.Allocate(inv, channels)

	if result["shopify"] != 60 {
		t.Errorf("shopify allocation = %d, want 60", result["shopify"])
	}
	if result["amazon"] != 40 {
		t.Errorf("amazon allocation = %d, want 40", result["amazon"])
	}
}

func TestAllocator_Allocate_Weighted_WithBuffer(t *testing.T) {
	a := NewAllocator()
	a.SetStrategy(StrategyWeighted)
	a.SetGlobalBuffer(10.0) // 10% buffer

	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "shopify",
		Weight:    100.0,
	})

	inv := &models.Inventory{
		SKU:       "TEST-001",
		Available: 100,
	}

	channels := []ChannelHandler{
		&mockChannelHandler{channelID: "shopify"},
	}

	result := a.Allocate(inv, channels)

	// 100 - 10% buffer = 90 available
	if result["shopify"] != 90 {
		t.Errorf("shopify allocation = %d, want 90", result["shopify"])
	}
}

func TestAllocator_Allocate_Weighted_ChannelBuffer(t *testing.T) {
	a := NewAllocator()
	a.SetStrategy(StrategyWeighted)
	a.SetGlobalBuffer(0)

	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "shopify",
		Weight:    100.0,
		Buffer:    20.0, // 20% channel buffer
	})

	inv := &models.Inventory{
		SKU:       "TEST-001",
		Available: 100,
	}

	channels := []ChannelHandler{
		&mockChannelHandler{channelID: "shopify"},
	}

	result := a.Allocate(inv, channels)

	// 100 * (1 - 0.20) = 80
	if result["shopify"] != 80 {
		t.Errorf("shopify allocation = %d, want 80", result["shopify"])
	}
}

func TestAllocator_Allocate_Weighted_MinStock(t *testing.T) {
	a := NewAllocator()
	a.SetStrategy(StrategyWeighted)
	a.SetGlobalBuffer(0)

	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "shopify",
		Weight:    10.0, // Would normally get 10
		MinStock:  50,   // But min is 50
	})

	inv := &models.Inventory{
		SKU:       "TEST-001",
		Available: 100,
	}

	channels := []ChannelHandler{
		&mockChannelHandler{channelID: "shopify"},
	}

	result := a.Allocate(inv, channels)

	if result["shopify"] < 50 {
		t.Errorf("shopify allocation = %d, want >= 50 (minStock)", result["shopify"])
	}
}

func TestAllocator_Allocate_Weighted_MaxStock(t *testing.T) {
	a := NewAllocator()
	a.SetStrategy(StrategyWeighted)
	a.SetGlobalBuffer(0)

	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "shopify",
		Weight:    100.0,
		MaxStock:  30, // Cap at 30
	})

	inv := &models.Inventory{
		SKU:       "TEST-001",
		Available: 100,
	}

	channels := []ChannelHandler{
		&mockChannelHandler{channelID: "shopify"},
	}

	result := a.Allocate(inv, channels)

	if result["shopify"] > 30 {
		t.Errorf("shopify allocation = %d, want <= 30 (maxStock)", result["shopify"])
	}
}

func TestAllocator_Allocate_Fixed(t *testing.T) {
	a := NewAllocator()
	a.SetStrategy(StrategyFixed)
	a.SetGlobalBuffer(0)

	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "shopify",
		FixedQty:  30,
	})
	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "amazon",
		FixedQty:  20,
	})

	inv := &models.Inventory{
		SKU:       "TEST-001",
		Available: 100,
	}

	channels := []ChannelHandler{
		&mockChannelHandler{channelID: "shopify"},
		&mockChannelHandler{channelID: "amazon"},
	}

	result := a.Allocate(inv, channels)

	if result["shopify"] != 30 {
		t.Errorf("shopify allocation = %d, want 30", result["shopify"])
	}
	if result["amazon"] != 20 {
		t.Errorf("amazon allocation = %d, want 20", result["amazon"])
	}
}

func TestAllocator_Allocate_Fixed_InsufficientStock(t *testing.T) {
	a := NewAllocator()
	a.SetStrategy(StrategyFixed)
	a.SetGlobalBuffer(0)

	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "shopify",
		FixedQty:  80,
	})
	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "amazon",
		FixedQty:  50, // Would exceed available
	})

	inv := &models.Inventory{
		SKU:       "TEST-001",
		Available: 100,
	}

	channels := []ChannelHandler{
		&mockChannelHandler{channelID: "shopify"},
		&mockChannelHandler{channelID: "amazon"},
	}

	result := a.Allocate(inv, channels)

	total := result["shopify"] + result["amazon"]
	if total > 100 {
		t.Errorf("total allocation = %d, should not exceed 100", total)
	}
}

func TestAllocator_Allocate_Fixed_MaxStock(t *testing.T) {
	a := NewAllocator()
	a.SetStrategy(StrategyFixed)
	a.SetGlobalBuffer(0)

	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "shopify",
		FixedQty:  100,
		MaxStock:  50, // Cap at 50
	})

	inv := &models.Inventory{
		SKU:       "TEST-001",
		Available: 100,
	}

	channels := []ChannelHandler{
		&mockChannelHandler{channelID: "shopify"},
	}

	result := a.Allocate(inv, channels)

	if result["shopify"] > 50 {
		t.Errorf("shopify allocation = %d, want <= 50 (maxStock)", result["shopify"])
	}
}

func TestAllocator_Allocate_Priority(t *testing.T) {
	a := NewAllocator()
	a.SetStrategy(StrategyPriority)
	a.SetGlobalBuffer(0)

	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "shopify",
		Priority:  1, // Highest priority
		MinStock:  30,
	})
	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "amazon",
		Priority:  2, // Second priority
		MinStock:  30,
	})

	inv := &models.Inventory{
		SKU:       "TEST-001",
		Available: 100,
	}

	channels := []ChannelHandler{
		&mockChannelHandler{channelID: "amazon"},
		&mockChannelHandler{channelID: "shopify"},
	}

	result := a.Allocate(inv, channels)

	// Shopify should get stock first due to higher priority
	if result["shopify"] < result["amazon"] {
		t.Errorf("shopify (priority 1) should get at least as much as amazon (priority 2)")
	}
}

func TestAllocator_Allocate_Dynamic(t *testing.T) {
	a := NewAllocator()
	a.SetStrategy(StrategyDynamic)
	a.SetGlobalBuffer(10.0)

	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "shopify",
		Weight:    100.0,
	})

	inv := &models.Inventory{
		SKU:       "TEST-001",
		Available: 100,
	}

	channels := []ChannelHandler{
		&mockChannelHandler{channelID: "shopify"},
	}

	result := a.Allocate(inv, channels)

	// Dynamic currently falls back to weighted
	if result["shopify"] > 90 {
		t.Errorf("allocation = %d, should account for buffer", result["shopify"])
	}
}

func TestAllocator_Allocate_NoConfig(t *testing.T) {
	a := NewAllocator()
	a.SetStrategy(StrategyWeighted)
	a.SetGlobalBuffer(0)

	// No channel configurations set - should use default equal weight
	inv := &models.Inventory{
		SKU:       "TEST-001",
		Available: 100,
	}

	channels := []ChannelHandler{
		&mockChannelHandler{channelID: "shopify"},
		&mockChannelHandler{channelID: "amazon"},
	}

	result := a.Allocate(inv, channels)

	// Should distribute evenly
	if result["shopify"] != 50 || result["amazon"] != 50 {
		t.Errorf("shopify=%d, amazon=%d, expected 50/50 split", result["shopify"], result["amazon"])
	}
}

func TestAllocator_Allocate_ZeroAvailable(t *testing.T) {
	a := NewAllocator()
	a.SetStrategy(StrategyWeighted)
	a.SetGlobalBuffer(0)

	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "shopify",
		Weight:    100.0,
	})

	inv := &models.Inventory{
		SKU:       "TEST-001",
		Available: 0,
	}

	channels := []ChannelHandler{
		&mockChannelHandler{channelID: "shopify"},
	}

	result := a.Allocate(inv, channels)

	if result["shopify"] != 0 {
		t.Errorf("allocation = %d, want 0 when no stock available", result["shopify"])
	}
}

func TestAllocator_Allocate_NegativeAfterBuffer(t *testing.T) {
	a := NewAllocator()
	a.SetStrategy(StrategyWeighted)
	a.SetGlobalBuffer(200) // More than 100%, would make available negative

	inv := &models.Inventory{
		SKU:       "TEST-001",
		Available: 100,
	}

	channels := []ChannelHandler{
		&mockChannelHandler{channelID: "shopify"},
	}

	result := a.Allocate(inv, channels)

	// Should handle gracefully and not allocate negative
	if result["shopify"] < 0 {
		t.Errorf("allocation = %d, should not be negative", result["shopify"])
	}
}

func TestAllocator_RecalculateAllocation(t *testing.T) {
	a := NewAllocator()
	a.SetStrategy(StrategyWeighted)
	a.SetGlobalBuffer(0)

	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "shopify",
		Weight:    60.0,
	})
	a.SetChannelAllocation(&ChannelAllocation{
		ChannelID: "amazon",
		Weight:    40.0,
	})

	inv := &models.Inventory{
		SKU:       "TEST-001",
		Available: 100,
	}

	currentAllocations := map[string]int{
		"shopify": 50,
		"amazon":  50,
	}

	channels := []ChannelHandler{
		&mockChannelHandler{channelID: "shopify"},
		&mockChannelHandler{channelID: "amazon"},
	}

	newAllocations := a.RecalculateAllocation(inv, currentAllocations, channels)

	if newAllocations["shopify"] != 60 {
		t.Errorf("shopify new allocation = %d, want 60", newAllocations["shopify"])
	}
	if newAllocations["amazon"] != 40 {
		t.Errorf("amazon new allocation = %d, want 40", newAllocations["amazon"])
	}
}

func TestAllocationStrategy_Constants(t *testing.T) {
	if StrategyFixed != "fixed" {
		t.Errorf("StrategyFixed = %s, want fixed", StrategyFixed)
	}
	if StrategyWeighted != "weighted" {
		t.Errorf("StrategyWeighted = %s, want weighted", StrategyWeighted)
	}
	if StrategyPriority != "priority" {
		t.Errorf("StrategyPriority = %s, want priority", StrategyPriority)
	}
	if StrategyDynamic != "dynamic" {
		t.Errorf("StrategyDynamic = %s, want dynamic", StrategyDynamic)
	}
}
