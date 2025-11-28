package pricing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/savegress/streamline/pkg/models"
	"github.com/shopspring/decimal"
)

// Engine handles price calculation and synchronization
type Engine struct {
	rules     map[string]*models.PricingRule
	rulesMu   sync.RWMutex

	// Channel pricing
	channelPrices   map[string]map[string]decimal.Decimal // channelID -> SKU -> price
	channelPricesMu sync.RWMutex

	// Configuration
	minMargin   float64 // Minimum margin percentage
	roundTo     string  // Price rounding (0.99, 0.95, etc.)

	// Callbacks
	onPriceChange func(sku, channelID string, oldPrice, newPrice decimal.Decimal)
}

// NewEngine creates a new pricing engine
func NewEngine() *Engine {
	return &Engine{
		rules:         make(map[string]*models.PricingRule),
		channelPrices: make(map[string]map[string]decimal.Decimal),
		minMargin:     20.0,
		roundTo:       "0.99",
	}
}

// SetMinMargin sets the minimum margin
func (e *Engine) SetMinMargin(margin float64) {
	e.minMargin = margin
}

// SetRounding sets the price rounding
func (e *Engine) SetRounding(roundTo string) {
	e.roundTo = roundTo
}

// SetPriceChangeCallback sets the callback for price changes
func (e *Engine) SetPriceChangeCallback(fn func(sku, channelID string, oldPrice, newPrice decimal.Decimal)) {
	e.onPriceChange = fn
}

// AddRule adds a pricing rule
func (e *Engine) AddRule(rule *models.PricingRule) {
	e.rulesMu.Lock()
	defer e.rulesMu.Unlock()
	e.rules[rule.ID] = rule
}

// RemoveRule removes a pricing rule
func (e *Engine) RemoveRule(ruleID string) {
	e.rulesMu.Lock()
	defer e.rulesMu.Unlock()
	delete(e.rules, ruleID)
}

// GetRules returns all pricing rules
func (e *Engine) GetRules() []*models.PricingRule {
	e.rulesMu.RLock()
	defer e.rulesMu.RUnlock()

	result := make([]*models.PricingRule, 0, len(e.rules))
	for _, rule := range e.rules {
		result = append(result, rule)
	}
	return result
}

// CalculatePrice calculates the price for a product on a channel
func (e *Engine) CalculatePrice(ctx context.Context, product *models.Product, channelID string, channelFee float64) (*PriceCalculation, error) {
	calc := &PriceCalculation{
		SKU:       product.SKU,
		ChannelID: channelID,
		BasePrice: product.BasePrice,
		Cost:      product.Cost,
	}

	// Start with base price
	price := product.BasePrice

	// Get applicable rules (sorted by priority)
	rules := e.getApplicableRules(product.SKU, channelID)

	// Apply rules
	for _, rule := range rules {
		price = e.applyRule(price, product, rule)
		calc.AppliedRules = append(calc.AppliedRules, rule.Name)
	}

	// Calculate fees
	feeAmount := price.Mul(decimal.NewFromFloat(channelFee / 100))
	calc.ChannelFee = feeAmount

	// Calculate margin
	priceAfterFee := price.Sub(feeAmount)
	if product.Cost.GreaterThan(decimal.Zero) {
		margin := priceAfterFee.Sub(product.Cost).Div(price).Mul(decimal.NewFromInt(100))
		calc.Margin, _ = margin.Float64()
	}

	// Ensure minimum margin
	if calc.Margin < e.minMargin && product.Cost.GreaterThan(decimal.Zero) {
		// Adjust price to meet minimum margin
		requiredMargin := decimal.NewFromFloat(e.minMargin / 100)
		feePct := decimal.NewFromFloat(channelFee / 100)
		// price = cost / (1 - margin - fee)
		denominator := decimal.NewFromInt(1).Sub(requiredMargin).Sub(feePct)
		if denominator.GreaterThan(decimal.Zero) {
			price = product.Cost.Div(denominator)
			calc.AdjustedForMargin = true
		}
	}

	// Apply rounding
	price = e.roundPrice(price)
	calc.FinalPrice = price

	// Recalculate final margin
	finalFee := price.Mul(decimal.NewFromFloat(channelFee / 100))
	if product.Cost.GreaterThan(decimal.Zero) {
		finalMargin := price.Sub(finalFee).Sub(product.Cost).Div(price).Mul(decimal.NewFromInt(100))
		calc.Margin, _ = finalMargin.Float64()
	}

	return calc, nil
}

// PriceCalculation represents the result of price calculation
type PriceCalculation struct {
	SKU               string          `json:"sku"`
	ChannelID         string          `json:"channel_id"`
	BasePrice         decimal.Decimal `json:"base_price"`
	Cost              decimal.Decimal `json:"cost"`
	FinalPrice        decimal.Decimal `json:"final_price"`
	ChannelFee        decimal.Decimal `json:"channel_fee"`
	Margin            float64         `json:"margin_percent"`
	AppliedRules      []string        `json:"applied_rules"`
	AdjustedForMargin bool            `json:"adjusted_for_margin"`
}

func (e *Engine) getApplicableRules(sku, channelID string) []*models.PricingRule {
	e.rulesMu.RLock()
	defer e.rulesMu.RUnlock()

	var applicable []*models.PricingRule
	now := time.Now()

	for _, rule := range e.rules {
		if !rule.Active {
			continue
		}

		// Check date range
		if rule.StartDate != nil && now.Before(*rule.StartDate) {
			continue
		}
		if rule.EndDate != nil && now.After(*rule.EndDate) {
			continue
		}

		// Check channel
		if len(rule.Channels) > 0 {
			found := false
			for _, ch := range rule.Channels {
				if ch == channelID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check product
		if len(rule.Products) > 0 {
			found := false
			for _, p := range rule.Products {
				if p == sku {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		applicable = append(applicable, rule)
	}

	// Sort by priority (lower = higher priority)
	for i := 0; i < len(applicable)-1; i++ {
		for j := i + 1; j < len(applicable); j++ {
			if applicable[j].Priority < applicable[i].Priority {
				applicable[i], applicable[j] = applicable[j], applicable[i]
			}
		}
	}

	return applicable
}

func (e *Engine) applyRule(price decimal.Decimal, product *models.Product, rule *models.PricingRule) decimal.Decimal {
	action := rule.Action

	switch action.Type {
	case "percent":
		// Apply percentage adjustment
		adjustment := price.Mul(action.Value.Div(decimal.NewFromInt(100)))
		if rule.Type == models.PricingRuleTypeDiscount {
			price = price.Sub(adjustment)
		} else {
			price = price.Add(adjustment)
		}

	case "fixed":
		// Apply fixed adjustment
		if rule.Type == models.PricingRuleTypeDiscount {
			price = price.Sub(action.Value)
		} else if rule.Type == models.PricingRuleTypeFixed {
			price = action.Value
		} else {
			price = price.Add(action.Value)
		}

	case "match":
		// Match competitor price (would need competitor data)
		// For now, just use the value as target
		if action.Value.GreaterThan(decimal.Zero) {
			price = action.Value
		}
	}

	// Apply min/max constraints
	if action.MinPrice.GreaterThan(decimal.Zero) && price.LessThan(action.MinPrice) {
		price = action.MinPrice
	}
	if action.MaxPrice.GreaterThan(decimal.Zero) && price.GreaterThan(action.MaxPrice) {
		price = action.MaxPrice
	}

	return price
}

func (e *Engine) roundPrice(price decimal.Decimal) decimal.Decimal {
	switch e.roundTo {
	case "0.99":
		// Round to nearest .99
		rounded := price.Round(0)
		return rounded.Sub(decimal.NewFromFloat(0.01))
	case "0.95":
		// Round to nearest .95
		rounded := price.Round(0)
		return rounded.Sub(decimal.NewFromFloat(0.05))
	case "0.00":
		// Round to whole number
		return price.Round(0)
	default:
		return price.Round(2)
	}
}

// UpdateChannelPrice updates the cached price for a channel
func (e *Engine) UpdateChannelPrice(channelID, sku string, price decimal.Decimal) {
	e.channelPricesMu.Lock()
	defer e.channelPricesMu.Unlock()

	if e.channelPrices[channelID] == nil {
		e.channelPrices[channelID] = make(map[string]decimal.Decimal)
	}

	oldPrice := e.channelPrices[channelID][sku]
	e.channelPrices[channelID][sku] = price

	if e.onPriceChange != nil && !oldPrice.Equal(price) {
		e.onPriceChange(sku, channelID, oldPrice, price)
	}
}

// GetChannelPrice returns the current price for a SKU on a channel
func (e *Engine) GetChannelPrice(channelID, sku string) (decimal.Decimal, bool) {
	e.channelPricesMu.RLock()
	defer e.channelPricesMu.RUnlock()

	if prices, ok := e.channelPrices[channelID]; ok {
		if price, ok := prices[sku]; ok {
			return price, true
		}
	}
	return decimal.Zero, false
}

// GetAllChannelPrices returns all prices for a SKU across channels
func (e *Engine) GetAllChannelPrices(sku string) map[string]decimal.Decimal {
	e.channelPricesMu.RLock()
	defer e.channelPricesMu.RUnlock()

	result := make(map[string]decimal.Decimal)
	for channelID, prices := range e.channelPrices {
		if price, ok := prices[sku]; ok {
			result[channelID] = price
		}
	}
	return result
}

// CalculateAllChannelPrices calculates prices for all channels
func (e *Engine) CalculateAllChannelPrices(ctx context.Context, product *models.Product, channelFees map[string]float64) (map[string]*PriceCalculation, error) {
	result := make(map[string]*PriceCalculation)

	for channelID, fee := range channelFees {
		calc, err := e.CalculatePrice(ctx, product, channelID, fee)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate price for channel %s: %w", channelID, err)
		}
		result[channelID] = calc
	}

	return result, nil
}

// PricingComparison compares prices across channels
type PricingComparison struct {
	SKU         string                        `json:"sku"`
	BasePrice   decimal.Decimal               `json:"base_price"`
	Channels    map[string]ChannelPriceInfo   `json:"channels"`
	LowestPrice decimal.Decimal               `json:"lowest_price"`
	HighestPrice decimal.Decimal              `json:"highest_price"`
}

type ChannelPriceInfo struct {
	Price       decimal.Decimal `json:"price"`
	Fee         decimal.Decimal `json:"fee"`
	Margin      float64         `json:"margin"`
	IsCompetitive bool          `json:"is_competitive"`
}

// ComparePrices compares prices for a product across channels
func (e *Engine) ComparePrices(ctx context.Context, product *models.Product, channelFees map[string]float64) (*PricingComparison, error) {
	comparison := &PricingComparison{
		SKU:       product.SKU,
		BasePrice: product.BasePrice,
		Channels:  make(map[string]ChannelPriceInfo),
	}

	var lowestPrice, highestPrice decimal.Decimal
	first := true

	for channelID, fee := range channelFees {
		calc, err := e.CalculatePrice(ctx, product, channelID, fee)
		if err != nil {
			continue
		}

		comparison.Channels[channelID] = ChannelPriceInfo{
			Price:  calc.FinalPrice,
			Fee:    calc.ChannelFee,
			Margin: calc.Margin,
		}

		if first || calc.FinalPrice.LessThan(lowestPrice) {
			lowestPrice = calc.FinalPrice
		}
		if first || calc.FinalPrice.GreaterThan(highestPrice) {
			highestPrice = calc.FinalPrice
		}
		first = false
	}

	comparison.LowestPrice = lowestPrice
	comparison.HighestPrice = highestPrice

	// Mark competitive prices (within 5% of lowest)
	threshold := lowestPrice.Mul(decimal.NewFromFloat(1.05))
	for channelID, info := range comparison.Channels {
		info.IsCompetitive = info.Price.LessThanOrEqual(threshold)
		comparison.Channels[channelID] = info
	}

	return comparison, nil
}
