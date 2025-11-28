package orders

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/savegress/streamline/pkg/models"
	"github.com/shopspring/decimal"
)

// Router handles intelligent order routing to warehouses
type Router struct {
	warehouses    map[string]*models.Warehouse
	optimization  OptimizationStrategy
	shippingRates map[string]ShippingRate // carrier -> rate
}

// OptimizationStrategy defines how orders are routed
type OptimizationStrategy string

const (
	OptimizationCost    OptimizationStrategy = "cost"    // Minimize shipping cost
	OptimizationSpeed   OptimizationStrategy = "speed"   // Minimize delivery time
	OptimizationBalance OptimizationStrategy = "balance" // Balance stock across warehouses
)

// ShippingRate defines shipping cost calculation
type ShippingRate struct {
	CarrierID   string
	BaseRate    decimal.Decimal
	PerMile     decimal.Decimal
	PerPound    decimal.Decimal
	Zones       map[string]decimal.Decimal // zone -> multiplier
}

// RoutingOption represents a possible routing decision
type RoutingOption struct {
	WarehouseID     string          `json:"warehouse_id"`
	WarehouseName   string          `json:"warehouse_name"`
	Distance        float64         `json:"distance_miles"`
	EstimatedDays   int             `json:"estimated_days"`
	ShippingCost    decimal.Decimal `json:"shipping_cost"`
	AllItemsInStock bool            `json:"all_items_in_stock"`
	ItemsAvailable  map[string]int  `json:"items_available"` // SKU -> qty available
	Score           float64         `json:"score"`           // Overall routing score
	Recommended     bool            `json:"recommended"`
}

// RoutingDecision represents the final routing decision
type RoutingDecision struct {
	OrderID         string          `json:"order_id"`
	Options         []RoutingOption `json:"options"`
	SelectedOption  *RoutingOption  `json:"selected_option"`
	IsSplitShipment bool            `json:"is_split_shipment"`
	SplitDetails    []SplitShipment `json:"split_details,omitempty"`
	Reason          string          `json:"reason"`
}

// SplitShipment represents a split shipment
type SplitShipment struct {
	WarehouseID  string   `json:"warehouse_id"`
	Items        []string `json:"items"` // SKUs
	ShippingCost decimal.Decimal `json:"shipping_cost"`
}

// NewRouter creates a new order router
func NewRouter() *Router {
	return &Router{
		warehouses:    make(map[string]*models.Warehouse),
		optimization:  OptimizationCost,
		shippingRates: make(map[string]ShippingRate),
	}
}

// SetOptimization sets the optimization strategy
func (r *Router) SetOptimization(strategy OptimizationStrategy) {
	r.optimization = strategy
}

// AddWarehouse adds a warehouse
func (r *Router) AddWarehouse(warehouse *models.Warehouse) {
	r.warehouses[warehouse.ID] = warehouse
}

// SetShippingRate sets shipping rates for a carrier
func (r *Router) SetShippingRate(rate ShippingRate) {
	r.shippingRates[rate.CarrierID] = rate
}

// RouteOrder determines the best warehouse for an order
func (r *Router) RouteOrder(ctx context.Context, order *models.Order, inventory map[string]*InventoryInfo) (*RoutingDecision, error) {
	decision := &RoutingDecision{
		OrderID: order.ID,
		Options: make([]RoutingOption, 0),
	}

	// Evaluate each warehouse
	for _, warehouse := range r.warehouses {
		if !warehouse.Active {
			continue
		}

		option := r.evaluateWarehouse(order, warehouse, inventory)
		decision.Options = append(decision.Options, option)
	}

	// Sort options by score (higher is better)
	sort.Slice(decision.Options, func(i, j int) bool {
		return decision.Options[i].Score > decision.Options[j].Score
	})

	// Select best option
	if len(decision.Options) > 0 {
		// Find best option with all items in stock
		for i := range decision.Options {
			if decision.Options[i].AllItemsInStock {
				decision.Options[i].Recommended = true
				decision.SelectedOption = &decision.Options[i]
				decision.Reason = r.generateReason(decision.Options[i])
				break
			}
		}

		// If no single warehouse has all items, consider split shipment
		if decision.SelectedOption == nil {
			splitDecision := r.evaluateSplitShipment(order, inventory)
			if splitDecision != nil {
				decision.IsSplitShipment = true
				decision.SplitDetails = splitDecision.SplitDetails
				decision.Reason = "Split shipment required - no single warehouse has all items"
			} else {
				// Just pick the best available
				decision.Options[0].Recommended = true
				decision.SelectedOption = &decision.Options[0]
				decision.Reason = "Best available option (some items may be unavailable)"
			}
		}
	}

	return decision, nil
}

// InventoryInfo provides inventory info for routing
type InventoryInfo struct {
	SKU         string
	WarehouseID string
	Available   int
}

func (r *Router) evaluateWarehouse(order *models.Order, warehouse *models.Warehouse, inventory map[string]*InventoryInfo) RoutingOption {
	option := RoutingOption{
		WarehouseID:     warehouse.ID,
		WarehouseName:   warehouse.Name,
		ItemsAvailable:  make(map[string]int),
		AllItemsInStock: true,
	}

	// Calculate distance
	option.Distance = r.calculateDistance(warehouse.Address, order.Shipping.Address)

	// Estimate delivery days
	option.EstimatedDays = r.estimateDeliveryDays(option.Distance, order.Shipping.Method)

	// Calculate shipping cost
	option.ShippingCost = r.calculateShippingCost(option.Distance, order)

	// Check item availability
	for _, item := range order.Items {
		key := fmt.Sprintf("%s:%s", item.SKU, warehouse.ID)
		if inv, ok := inventory[key]; ok {
			option.ItemsAvailable[item.SKU] = inv.Available
			if inv.Available < item.Quantity {
				option.AllItemsInStock = false
			}
		} else {
			option.ItemsAvailable[item.SKU] = 0
			option.AllItemsInStock = false
		}
	}

	// Check warehouse capabilities
	// This is simplified - would check item tags against warehouse capabilities
	_ = order.Items

	// Calculate score based on optimization strategy
	option.Score = r.calculateScore(option)

	return option
}

func (r *Router) calculateDistance(from, to models.Address) float64 {
	// Simplified distance calculation using zip codes
	// In production, would use geocoding API
	// For now, return a mock distance based on state
	if from.State == to.State {
		return 50 // Same state
	}
	// Different states - use rough estimate
	return 500
}

func (r *Router) estimateDeliveryDays(distance float64, method string) int {
	switch method {
	case "express", "overnight":
		return 1
	case "expedited", "2day":
		return 2
	default: // standard
		if distance < 100 {
			return 2
		} else if distance < 500 {
			return 3
		} else if distance < 1000 {
			return 5
		}
		return 7
	}
}

func (r *Router) calculateShippingCost(distance float64, order *models.Order) decimal.Decimal {
	// Simplified shipping cost calculation
	// Base rate + per mile + per item
	baseCost := decimal.NewFromFloat(5.00)
	perMile := decimal.NewFromFloat(0.01)
	perItem := decimal.NewFromFloat(0.50)

	distanceCost := perMile.Mul(decimal.NewFromFloat(distance))
	itemsCost := perItem.Mul(decimal.NewFromInt(int64(len(order.Items))))

	return baseCost.Add(distanceCost).Add(itemsCost)
}

func (r *Router) calculateScore(option RoutingOption) float64 {
	score := 100.0

	// Penalize based on optimization strategy
	switch r.optimization {
	case OptimizationCost:
		// Lower cost = higher score
		cost, _ := option.ShippingCost.Float64()
		score -= cost * 2

	case OptimizationSpeed:
		// Fewer days = higher score
		score -= float64(option.EstimatedDays) * 10

	case OptimizationBalance:
		// Prefer warehouses with more stock (to balance inventory)
		var totalAvailable int
		for _, qty := range option.ItemsAvailable {
			totalAvailable += qty
		}
		score += float64(totalAvailable) * 0.5
	}

	// Bonus for having all items in stock
	if option.AllItemsInStock {
		score += 50
	}

	// Penalize for distance
	score -= option.Distance * 0.05

	return math.Max(0, score)
}

func (r *Router) generateReason(option RoutingOption) string {
	switch r.optimization {
	case OptimizationCost:
		return fmt.Sprintf("Lowest shipping cost: $%.2f", option.ShippingCost.InexactFloat64())
	case OptimizationSpeed:
		return fmt.Sprintf("Fastest delivery: %d days", option.EstimatedDays)
	case OptimizationBalance:
		return "Best stock balance"
	default:
		return "Best overall option"
	}
}

func (r *Router) evaluateSplitShipment(order *models.Order, inventory map[string]*InventoryInfo) *RoutingDecision {
	// Find items available in each warehouse
	warehouseItems := make(map[string][]string) // warehouseID -> SKUs

	for _, item := range order.Items {
		for warehouseID := range r.warehouses {
			key := fmt.Sprintf("%s:%s", item.SKU, warehouseID)
			if inv, ok := inventory[key]; ok && inv.Available >= item.Quantity {
				warehouseItems[warehouseID] = append(warehouseItems[warehouseID], item.SKU)
			}
		}
	}

	// Check if we can fulfill all items across warehouses
	var allItems []string
	for _, item := range order.Items {
		allItems = append(allItems, item.SKU)
	}

	coveredItems := make(map[string]bool)
	var splits []SplitShipment

	for warehouseID, items := range warehouseItems {
		var itemsForThisWarehouse []string
		for _, sku := range items {
			if !coveredItems[sku] {
				itemsForThisWarehouse = append(itemsForThisWarehouse, sku)
				coveredItems[sku] = true
			}
		}
		if len(itemsForThisWarehouse) > 0 {
			splits = append(splits, SplitShipment{
				WarehouseID:  warehouseID,
				Items:        itemsForThisWarehouse,
				ShippingCost: decimal.NewFromFloat(5.00), // Base cost per shipment
			})
		}
	}

	// Check if all items are covered
	if len(coveredItems) == len(allItems) {
		return &RoutingDecision{
			IsSplitShipment: true,
			SplitDetails:    splits,
		}
	}

	return nil
}

// AutoRoute automatically routes an order based on configuration
func (r *Router) AutoRoute(ctx context.Context, order *models.Order, inventory map[string]*InventoryInfo) (*models.Order, error) {
	decision, err := r.RouteOrder(ctx, order, inventory)
	if err != nil {
		return nil, err
	}

	if decision.SelectedOption != nil {
		order.WarehouseID = decision.SelectedOption.WarehouseID
	} else if decision.IsSplitShipment && len(decision.SplitDetails) > 0 {
		// For split shipments, assign to first warehouse (would need to handle splits differently)
		order.WarehouseID = decision.SplitDetails[0].WarehouseID
	}

	return order, nil
}
