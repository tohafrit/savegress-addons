package inventory

import (
	"github.com/savegress/streamline/pkg/models"
)

// AllocationStrategy defines how inventory is allocated across channels
type AllocationStrategy string

const (
	StrategyFixed    AllocationStrategy = "fixed"    // Fixed quantities per channel
	StrategyWeighted AllocationStrategy = "weighted" // Percentage-based allocation
	StrategyPriority AllocationStrategy = "priority" // Priority-based (first come, first served)
	StrategyDynamic  AllocationStrategy = "dynamic"  // Based on sales velocity
)

// ChannelAllocation defines allocation settings for a channel
type ChannelAllocation struct {
	ChannelID   string
	Weight      float64 // For weighted strategy (0-100)
	Priority    int     // For priority strategy (lower = higher priority)
	MinStock    int     // Minimum stock to maintain
	MaxStock    int     // Maximum stock to allocate (0 = unlimited)
	Buffer      float64 // Safety buffer percentage
	FixedQty    int     // For fixed strategy
}

// Allocator handles inventory allocation across channels
type Allocator struct {
	strategy     AllocationStrategy
	allocations  map[string]*ChannelAllocation // channelID -> allocation config
	globalBuffer float64                       // Global safety buffer percentage
}

// NewAllocator creates a new allocator
func NewAllocator() *Allocator {
	return &Allocator{
		strategy:     StrategyWeighted,
		allocations:  make(map[string]*ChannelAllocation),
		globalBuffer: 10.0, // 10% default safety buffer
	}
}

// SetStrategy sets the allocation strategy
func (a *Allocator) SetStrategy(strategy AllocationStrategy) {
	a.strategy = strategy
}

// SetChannelAllocation sets allocation config for a channel
func (a *Allocator) SetChannelAllocation(config *ChannelAllocation) {
	a.allocations[config.ChannelID] = config
}

// SetGlobalBuffer sets the global safety buffer
func (a *Allocator) SetGlobalBuffer(buffer float64) {
	a.globalBuffer = buffer
}

// Allocate calculates inventory allocation for each channel
func (a *Allocator) Allocate(inv *models.Inventory, channels []ChannelHandler) map[string]int {
	result := make(map[string]int)

	// Calculate available stock after buffer
	bufferQty := int(float64(inv.Available) * (a.globalBuffer / 100))
	availableForAllocation := inv.Available - bufferQty
	if availableForAllocation < 0 {
		availableForAllocation = 0
	}

	switch a.strategy {
	case StrategyFixed:
		result = a.allocateFixed(availableForAllocation, channels)
	case StrategyWeighted:
		result = a.allocateWeighted(availableForAllocation, channels)
	case StrategyPriority:
		result = a.allocatePriority(availableForAllocation, channels)
	case StrategyDynamic:
		result = a.allocateDynamic(inv, channels)
	default:
		result = a.allocateWeighted(availableForAllocation, channels)
	}

	return result
}

func (a *Allocator) allocateFixed(available int, channels []ChannelHandler) map[string]int {
	result := make(map[string]int)
	remaining := available

	for _, ch := range channels {
		channelID := ch.GetChannelID()
		config, ok := a.allocations[channelID]
		if !ok {
			continue
		}

		qty := config.FixedQty
		if qty > remaining {
			qty = remaining
		}
		if config.MaxStock > 0 && qty > config.MaxStock {
			qty = config.MaxStock
		}

		result[channelID] = qty
		remaining -= qty
	}

	return result
}

func (a *Allocator) allocateWeighted(available int, channels []ChannelHandler) map[string]int {
	result := make(map[string]int)

	// Calculate total weight
	var totalWeight float64
	for _, ch := range channels {
		if config, ok := a.allocations[ch.GetChannelID()]; ok {
			totalWeight += config.Weight
		} else {
			totalWeight += 100 / float64(len(channels)) // Default equal weight
		}
	}

	if totalWeight == 0 {
		totalWeight = 100
	}

	// Allocate based on weight
	remaining := available
	for _, ch := range channels {
		channelID := ch.GetChannelID()
		config := a.allocations[channelID]

		var weight float64
		if config != nil {
			weight = config.Weight
		} else {
			weight = 100 / float64(len(channels))
		}

		qty := int(float64(available) * (weight / totalWeight))

		// Apply channel-specific buffer
		if config != nil && config.Buffer > 0 {
			bufferQty := int(float64(qty) * (config.Buffer / 100))
			qty -= bufferQty
		}

		// Apply min/max constraints
		if config != nil {
			if config.MinStock > 0 && qty < config.MinStock && remaining >= config.MinStock {
				qty = config.MinStock
			}
			if config.MaxStock > 0 && qty > config.MaxStock {
				qty = config.MaxStock
			}
		}

		if qty > remaining {
			qty = remaining
		}
		if qty < 0 {
			qty = 0
		}

		result[channelID] = qty
		remaining -= qty
	}

	return result
}

func (a *Allocator) allocatePriority(available int, channels []ChannelHandler) map[string]int {
	result := make(map[string]int)

	// Sort channels by priority
	type channelPriority struct {
		handler  ChannelHandler
		priority int
	}
	sorted := make([]channelPriority, 0, len(channels))
	for _, ch := range channels {
		priority := 100 // Default priority
		if config, ok := a.allocations[ch.GetChannelID()]; ok {
			priority = config.Priority
		}
		sorted = append(sorted, channelPriority{handler: ch, priority: priority})
	}

	// Simple sort by priority
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].priority < sorted[i].priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Allocate in priority order
	remaining := available
	for _, cp := range sorted {
		channelID := cp.handler.GetChannelID()
		config := a.allocations[channelID]

		qty := remaining
		if config != nil {
			if config.MaxStock > 0 && qty > config.MaxStock {
				qty = config.MaxStock
			}
			if config.MinStock > 0 && qty > config.MinStock {
				qty = config.MinStock // Allocate only min to allow for other channels
			}
		}

		if qty < 0 {
			qty = 0
		}

		result[channelID] = qty
		remaining -= qty

		if remaining <= 0 {
			break
		}
	}

	return result
}

func (a *Allocator) allocateDynamic(inv *models.Inventory, channels []ChannelHandler) map[string]int {
	// Dynamic allocation based on sales velocity
	// For now, fall back to weighted allocation
	// TODO: Implement sales velocity tracking

	bufferQty := int(float64(inv.Available) * (a.globalBuffer / 100))
	available := inv.Available - bufferQty
	if available < 0 {
		available = 0
	}

	return a.allocateWeighted(available, channels)
}

// RecalculateAllocation recalculates allocation when stock changes
func (a *Allocator) RecalculateAllocation(inv *models.Inventory, currentAllocations map[string]int, channels []ChannelHandler) map[string]int {
	newAllocations := a.Allocate(inv, channels)

	// Calculate changes
	changes := make(map[string]int)
	for channelID, newQty := range newAllocations {
		currentQty := currentAllocations[channelID]
		if newQty != currentQty {
			changes[channelID] = newQty - currentQty
		}
	}

	return newAllocations
}
