package metrics

import (
	"context"
	"time"

	"github.com/savegress/datawatch/internal/storage"
)

// Aggregator handles time-window aggregations of metrics
type Aggregator struct {
	storage storage.MetricStorage
}

// NewAggregator creates a new aggregator
func NewAggregator(store storage.MetricStorage) *Aggregator {
	return &Aggregator{
		storage: store,
	}
}

// RunMinuteAggregations computes 1-minute aggregations
func (a *Aggregator) RunMinuteAggregations(ctx context.Context) {
	// This will be called every minute to compute rolling aggregations
	// For now, the storage layer handles this
}

// AggregationWindow represents a time window for aggregation
type AggregationWindow struct {
	Duration time.Duration
	Name     string
}

// Standard aggregation windows
var (
	Window1m  = AggregationWindow{Duration: time.Minute, Name: "1m"}
	Window5m  = AggregationWindow{Duration: 5 * time.Minute, Name: "5m"}
	Window15m = AggregationWindow{Duration: 15 * time.Minute, Name: "15m"}
	Window1h  = AggregationWindow{Duration: time.Hour, Name: "1h"}
	Window6h  = AggregationWindow{Duration: 6 * time.Hour, Name: "6h"}
	Window24h = AggregationWindow{Duration: 24 * time.Hour, Name: "24h"}
	Window7d  = AggregationWindow{Duration: 7 * 24 * time.Hour, Name: "7d"}
	Window30d = AggregationWindow{Duration: 30 * 24 * time.Hour, Name: "30d"}
)

// RateCalculator calculates rates from counter metrics
type RateCalculator struct {
	windowSize time.Duration
}

// NewRateCalculator creates a new rate calculator
func NewRateCalculator(windowSize time.Duration) *RateCalculator {
	return &RateCalculator{
		windowSize: windowSize,
	}
}

// CalculateRate calculates the rate of change per second
func (rc *RateCalculator) CalculateRate(dataPoints []DataPoint) float64 {
	if len(dataPoints) < 2 {
		return 0
	}

	first := dataPoints[0]
	last := dataPoints[len(dataPoints)-1]

	timeDiff := last.Timestamp.Sub(first.Timestamp).Seconds()
	if timeDiff == 0 {
		return 0
	}

	valueDiff := last.Value - first.Value
	return valueDiff / timeDiff
}

// PercentileCalculator calculates percentiles from data points
type PercentileCalculator struct{}

// NewPercentileCalculator creates a new percentile calculator
func NewPercentileCalculator() *PercentileCalculator {
	return &PercentileCalculator{}
}

// Calculate calculates the given percentile (0-100)
func (pc *PercentileCalculator) Calculate(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Sort values
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sortFloat64s(sorted)

	// Calculate index
	index := (percentile / 100.0) * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	// Linear interpolation
	fraction := index - float64(lower)
	return sorted[lower] + fraction*(sorted[upper]-sorted[lower])
}

// Simple sort for float64 slice
func sortFloat64s(values []float64) {
	for i := 0; i < len(values)-1; i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

// MovingAverage calculates moving average
type MovingAverage struct {
	windowSize int
	values     []float64
	sum        float64
}

// NewMovingAverage creates a new moving average calculator
func NewMovingAverage(windowSize int) *MovingAverage {
	return &MovingAverage{
		windowSize: windowSize,
		values:     make([]float64, 0, windowSize),
	}
}

// Add adds a value and returns the current average
func (ma *MovingAverage) Add(value float64) float64 {
	ma.values = append(ma.values, value)
	ma.sum += value

	if len(ma.values) > ma.windowSize {
		ma.sum -= ma.values[0]
		ma.values = ma.values[1:]
	}

	return ma.sum / float64(len(ma.values))
}

// ExponentialMovingAverage calculates EMA
type ExponentialMovingAverage struct {
	alpha    float64
	value    float64
	hasValue bool
}

// NewExponentialMovingAverage creates a new EMA calculator
func NewExponentialMovingAverage(alpha float64) *ExponentialMovingAverage {
	return &ExponentialMovingAverage{
		alpha: alpha,
	}
}

// Add adds a value and returns the current EMA
func (ema *ExponentialMovingAverage) Add(value float64) float64 {
	if !ema.hasValue {
		ema.value = value
		ema.hasValue = true
		return value
	}

	ema.value = ema.alpha*value + (1-ema.alpha)*ema.value
	return ema.value
}
