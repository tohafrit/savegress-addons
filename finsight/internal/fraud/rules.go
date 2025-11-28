package fraud

import (
	"math"
	"time"

	"github.com/savegress/finsight/pkg/models"
	"github.com/shopspring/decimal"
)

// AmountRule detects unusual transaction amounts
type AmountRule struct {
	maxAmount float64
}

// NewAmountRule creates a new amount rule
func NewAmountRule(maxAmount float64) *AmountRule {
	return &AmountRule{maxAmount: maxAmount}
}

func (r *AmountRule) Name() string { return "amount" }
func (r *AmountRule) Priority() int { return 100 }

func (r *AmountRule) Evaluate(txn *models.Transaction, ctx *EvaluationContext) *RuleResult {
	result := &RuleResult{}

	amount := txn.Amount.InexactFloat64()

	// Check against absolute max
	if amount > r.maxAmount {
		result.Triggered = true
		result.Score = 3.0
		result.Description = "Transaction exceeds maximum allowed amount"
		result.Indicators = append(result.Indicators, models.FraudIndicator{
			Type:        "amount_anomaly",
			Description: "Amount exceeds threshold",
			Score:       3.0,
			Details: map[string]interface{}{
				"amount":    amount,
				"threshold": r.maxAmount,
			},
		})
	}

	// Check against account profile
	if ctx != nil && ctx.AccountProfile != nil {
		avgAmount := ctx.AccountProfile.AvgTransactionAmount.InexactFloat64()
		if avgAmount > 0 {
			deviation := amount / avgAmount
			if deviation > 5.0 {
				result.Triggered = true
				result.Score += 2.0
				result.Indicators = append(result.Indicators, models.FraudIndicator{
					Type:        "amount_anomaly",
					Description: "Amount significantly higher than average",
					Score:       2.0,
					Details: map[string]interface{}{
						"amount":    amount,
						"average":   avgAmount,
						"deviation": deviation,
					},
				})
			}
		}
	}

	return result
}

// VelocityRule detects unusual transaction velocity
type VelocityRule struct {
	window    time.Duration
	maxDaily  float64
}

// NewVelocityRule creates a new velocity rule
func NewVelocityRule(window time.Duration, maxDaily float64) *VelocityRule {
	return &VelocityRule{
		window:   window,
		maxDaily: maxDaily,
	}
}

func (r *VelocityRule) Name() string { return "velocity" }
func (r *VelocityRule) Priority() int { return 90 }

func (r *VelocityRule) Evaluate(txn *models.Transaction, ctx *EvaluationContext) *RuleResult {
	result := &RuleResult{}

	if ctx == nil || ctx.RecentActivity == nil {
		return result
	}

	activity := ctx.RecentActivity

	// Check transaction count velocity
	if activity.TransactionCount > 10 {
		result.Triggered = true
		result.Score = 2.0
		result.Indicators = append(result.Indicators, models.FraudIndicator{
			Type:        "velocity",
			Description: "High transaction count in short period",
			Score:       2.0,
			Details: map[string]interface{}{
				"count":  activity.TransactionCount,
				"window": activity.TimeWindow.String(),
			},
		})
	}

	// Check amount velocity
	totalAmount := activity.TotalAmount.InexactFloat64()
	if totalAmount > r.maxDaily {
		result.Triggered = true
		result.Score += 3.0
		result.Indicators = append(result.Indicators, models.FraudIndicator{
			Type:        "velocity",
			Description: "Daily amount limit exceeded",
			Score:       3.0,
			Details: map[string]interface{}{
				"total": totalAmount,
				"limit": r.maxDaily,
			},
		})
	}

	// Check unique locations
	if activity.UniqueLocations > 3 {
		result.Triggered = true
		result.Score += 2.5
		result.Indicators = append(result.Indicators, models.FraudIndicator{
			Type:        "velocity",
			Description: "Transactions from multiple locations",
			Score:       2.5,
			Details: map[string]interface{}{
				"locations": activity.UniqueLocations,
			},
		})
	}

	return result
}

// GeolocationRule detects location-based anomalies
type GeolocationRule struct {
	geofence *GeofenceChecker
}

// NewGeolocationRule creates a new geolocation rule
func NewGeolocationRule(geofence *GeofenceChecker) *GeolocationRule {
	return &GeolocationRule{geofence: geofence}
}

func (r *GeolocationRule) Name() string { return "geolocation" }
func (r *GeolocationRule) Priority() int { return 80 }

func (r *GeolocationRule) Evaluate(txn *models.Transaction, ctx *EvaluationContext) *RuleResult {
	result := &RuleResult{}

	if ctx == nil || ctx.GeoLocation == nil {
		return result
	}

	geo := ctx.GeoLocation

	// Check for high-risk countries
	if r.geofence.IsHighRiskCountry(geo.Country) {
		result.Triggered = true
		result.Score = 2.5
		result.Indicators = append(result.Indicators, models.FraudIndicator{
			Type:        "geolocation",
			Description: "Transaction from high-risk country",
			Score:       2.5,
			Details: map[string]interface{}{
				"country": geo.Country,
			},
		})
	}

	// Check for unusual location based on profile
	if ctx.AccountProfile != nil {
		isTypical := false
		for _, loc := range ctx.AccountProfile.TypicalLocations {
			if loc == geo.Country || loc == geo.City {
				isTypical = true
				break
			}
		}
		if !isTypical && len(ctx.AccountProfile.TypicalLocations) > 0 {
			result.Triggered = true
			result.Score += 1.5
			result.Indicators = append(result.Indicators, models.FraudIndicator{
				Type:        "geolocation",
				Description: "Transaction from unusual location",
				Score:       1.5,
				Details: map[string]interface{}{
					"location":          geo.City + ", " + geo.Country,
					"typical_locations": ctx.AccountProfile.TypicalLocations,
				},
			})
		}
	}

	// Check for impossible travel
	if ctx.AccountHistory != nil && len(ctx.AccountHistory) > 0 {
		lastTxn := ctx.AccountHistory[0]
		if lastTxn.Merchant != nil && txn.Merchant != nil {
			// Simplified check - in production would calculate actual distance and time
			if lastTxn.Merchant.Country != "" && geo.Country != "" &&
				lastTxn.Merchant.Country != geo.Country {
				timeDiff := txn.CreatedAt.Sub(lastTxn.CreatedAt)
				if timeDiff < 2*time.Hour {
					result.Triggered = true
					result.Score += 4.0
					result.Indicators = append(result.Indicators, models.FraudIndicator{
						Type:        "geolocation",
						Description: "Impossible travel detected",
						Score:       4.0,
						Details: map[string]interface{}{
							"previous_country": lastTxn.Merchant.Country,
							"current_country":  geo.Country,
							"time_difference":  timeDiff.String(),
						},
					})
				}
			}
		}
	}

	return result
}

// PatternRule detects suspicious patterns
type PatternRule struct {
	analyzer *PatternAnalyzer
}

// NewPatternRule creates a new pattern rule
func NewPatternRule(analyzer *PatternAnalyzer) *PatternRule {
	return &PatternRule{analyzer: analyzer}
}

func (r *PatternRule) Name() string { return "pattern" }
func (r *PatternRule) Priority() int { return 70 }

func (r *PatternRule) Evaluate(txn *models.Transaction, ctx *EvaluationContext) *RuleResult {
	result := &RuleResult{}

	if ctx == nil || ctx.AccountHistory == nil {
		return result
	}

	// Check for round amounts
	amount := txn.Amount.InexactFloat64()
	if isRoundAmount(amount) && amount > 100 {
		result.Score += 0.5
		result.Indicators = append(result.Indicators, models.FraudIndicator{
			Type:        "pattern",
			Description: "Large round amount transaction",
			Score:       0.5,
			Details: map[string]interface{}{
				"amount": amount,
			},
		})
	}

	// Check for repeated amounts
	repeatedCount := 0
	for _, hist := range ctx.AccountHistory {
		if hist.Amount.Equal(txn.Amount) {
			repeatedCount++
		}
	}
	if repeatedCount >= 3 {
		result.Triggered = true
		result.Score += 1.5
		result.Indicators = append(result.Indicators, models.FraudIndicator{
			Type:        "pattern",
			Description: "Repeated identical amounts",
			Score:       1.5,
			Details: map[string]interface{}{
				"amount":       amount,
				"repeat_count": repeatedCount,
			},
		})
	}

	// Check for card testing pattern (small amounts)
	if amount < 5 && len(ctx.AccountHistory) > 0 {
		smallTxnCount := 0
		for _, hist := range ctx.AccountHistory {
			if hist.Amount.LessThan(decimal.NewFromFloat(5)) {
				smallTxnCount++
			}
		}
		if smallTxnCount >= 3 {
			result.Triggered = true
			result.Score += 3.0
			result.Indicators = append(result.Indicators, models.FraudIndicator{
				Type:        "pattern",
				Description: "Potential card testing detected",
				Score:       3.0,
				Details: map[string]interface{}{
					"small_transaction_count": smallTxnCount,
				},
			})
		}
	}

	if len(result.Indicators) > 0 {
		result.Triggered = true
	}

	return result
}

func isRoundAmount(amount float64) bool {
	// Check if amount is a round number (ends in 00)
	return math.Mod(amount, 100) == 0 || math.Mod(amount, 50) == 0
}

// TimeRule detects unusual transaction times
type TimeRule struct{}

// NewTimeRule creates a new time rule
func NewTimeRule() *TimeRule {
	return &TimeRule{}
}

func (r *TimeRule) Name() string { return "time" }
func (r *TimeRule) Priority() int { return 60 }

func (r *TimeRule) Evaluate(txn *models.Transaction, ctx *EvaluationContext) *RuleResult {
	result := &RuleResult{}

	hour := txn.CreatedAt.Hour()

	// Night-time transactions (2 AM - 5 AM)
	if hour >= 2 && hour <= 5 {
		result.Score += 1.0
		result.Indicators = append(result.Indicators, models.FraudIndicator{
			Type:        "time",
			Description: "Transaction during unusual hours",
			Score:       1.0,
			Details: map[string]interface{}{
				"hour":      hour,
				"timestamp": txn.CreatedAt,
			},
		})
	}

	// Check against typical hours
	if ctx != nil && ctx.AccountProfile != nil && len(ctx.AccountProfile.TypicalHours) > 0 {
		isTypical := false
		for _, typicalHour := range ctx.AccountProfile.TypicalHours {
			if hour == typicalHour {
				isTypical = true
				break
			}
		}
		if !isTypical {
			result.Score += 0.5
			result.Indicators = append(result.Indicators, models.FraudIndicator{
				Type:        "time",
				Description: "Transaction outside typical hours",
				Score:       0.5,
				Details: map[string]interface{}{
					"hour":          hour,
					"typical_hours": ctx.AccountProfile.TypicalHours,
				},
			})
		}
	}

	if len(result.Indicators) > 0 {
		result.Triggered = true
	}

	return result
}

// MerchantRule detects suspicious merchant activity
type MerchantRule struct{}

// NewMerchantRule creates a new merchant rule
func NewMerchantRule() *MerchantRule {
	return &MerchantRule{}
}

func (r *MerchantRule) Name() string { return "merchant" }
func (r *MerchantRule) Priority() int { return 50 }

func (r *MerchantRule) Evaluate(txn *models.Transaction, ctx *EvaluationContext) *RuleResult {
	result := &RuleResult{}

	if txn.Merchant == nil {
		return result
	}

	// Check for high-risk MCC codes
	highRiskMCCs := map[string]bool{
		"5967": true, // Direct Marketing - Inbound Teleservices Merchant
		"5966": true, // Direct Marketing - Outbound Telemarketing
		"7995": true, // Betting/Casino Gambling
		"5962": true, // Direct Marketing - Travel
		"4829": true, // Money Transfer
		"6051": true, // Quasi Cash - Cryptocurrency
	}

	if highRiskMCCs[txn.Merchant.MCC] {
		result.Triggered = true
		result.Score = 2.0
		result.Indicators = append(result.Indicators, models.FraudIndicator{
			Type:        "merchant",
			Description: "Transaction with high-risk merchant category",
			Score:       2.0,
			Details: map[string]interface{}{
				"mcc":      txn.Merchant.MCC,
				"merchant": txn.Merchant.Name,
			},
		})
	}

	// Check if merchant is new to account
	if ctx != nil && ctx.AccountProfile != nil {
		isKnown := false
		for _, known := range ctx.AccountProfile.TypicalMerchants {
			if known == txn.Merchant.ID || known == txn.Merchant.Name {
				isKnown = true
				break
			}
		}
		if !isKnown && len(ctx.AccountProfile.TypicalMerchants) > 5 {
			result.Score += 0.5
			result.Indicators = append(result.Indicators, models.FraudIndicator{
				Type:        "merchant",
				Description: "First transaction with this merchant",
				Score:       0.5,
				Details: map[string]interface{}{
					"merchant": txn.Merchant.Name,
				},
			})
		}
	}

	if len(result.Indicators) > 0 && !result.Triggered {
		result.Triggered = true
	}

	return result
}
