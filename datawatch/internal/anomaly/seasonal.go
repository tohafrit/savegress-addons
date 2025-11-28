package anomaly

import (
	"math"
	"time"
)

// SeasonalDetector detects anomalies based on seasonal patterns
type SeasonalDetector struct {
	threshold float64
}

// NewSeasonalDetector creates a new seasonal detector
func NewSeasonalDetector(threshold float64) *SeasonalDetector {
	return &SeasonalDetector{
		threshold: threshold,
	}
}

// Detect checks if a value deviates from expected seasonal pattern
func (d *SeasonalDetector) Detect(value float64, ts time.Time, baseline *MetricBaseline) (score float64, isAnomaly bool, anomalyType AnomalyType) {
	if baseline.SeasonalData == nil || !baseline.SeasonalData.HasSeasonality {
		return 0, false, ""
	}

	// Get expected value based on hour and day
	hour := ts.Hour()
	day := int(ts.Weekday())

	expectedHourly := baseline.SeasonalData.HourlyPattern[hour]
	expectedDaily := baseline.SeasonalData.DailyPattern[day]

	// Combine hourly and daily expectations
	// Weight hourly more since it's usually more granular
	expected := 0.7*expectedHourly + 0.3*expectedDaily

	if baseline.StdDev == 0 {
		return 0, false, ""
	}

	// Calculate deviation from expected
	deviation := math.Abs(value - expected) / baseline.StdDev

	// Seasonal threshold is typically lower than pure statistical
	seasonalThreshold := d.threshold * 0.8

	score = 1 - 1/(1+math.Exp(deviation-seasonalThreshold))
	isAnomaly = deviation > seasonalThreshold

	if isAnomaly {
		anomalyType = AnomalySeasonal
	}

	return score, isAnomaly, anomalyType
}

// STLDecomposition performs Seasonal-Trend decomposition using LOESS
// Simplified version - full implementation would use proper LOESS
type STLDecomposition struct {
	period int // Seasonal period (e.g., 24 for hourly data with daily seasonality)
}

// NewSTLDecomposition creates a new STL decomposer
func NewSTLDecomposition(period int) *STLDecomposition {
	return &STLDecomposition{
		period: period,
	}
}

// Decompose decomposes a time series into trend, seasonal, and residual components
func (s *STLDecomposition) Decompose(values []float64) (trend, seasonal, residual []float64) {
	n := len(values)
	if n < s.period*2 {
		return values, make([]float64, n), make([]float64, n)
	}

	// Step 1: Extract seasonal component using period averages
	seasonal = make([]float64, n)
	periodMeans := make([]float64, s.period)
	periodCounts := make([]int, s.period)

	for i, v := range values {
		idx := i % s.period
		periodMeans[idx] += v
		periodCounts[idx]++
	}

	for i := range periodMeans {
		if periodCounts[i] > 0 {
			periodMeans[i] /= float64(periodCounts[i])
		}
	}

	// Center the seasonal component
	meanSeasonal := mean(periodMeans)
	for i := range periodMeans {
		periodMeans[i] -= meanSeasonal
	}

	for i := range seasonal {
		seasonal[i] = periodMeans[i%s.period]
	}

	// Step 2: Remove seasonal and compute trend using moving average
	deseasonalized := make([]float64, n)
	for i, v := range values {
		deseasonalized[i] = v - seasonal[i]
	}

	trend = movingAverage(deseasonalized, s.period)

	// Step 3: Compute residual
	residual = make([]float64, n)
	for i := range values {
		residual[i] = values[i] - trend[i] - seasonal[i]
	}

	return trend, seasonal, residual
}

// DetectFromResidual detects anomalies from STL residual component
func (s *STLDecomposition) DetectFromResidual(residual []float64, threshold float64) []int {
	if len(residual) == 0 {
		return nil
	}

	m := mean(residual)
	sd := stdDev(residual, m)

	if sd == 0 {
		return nil
	}

	var anomalyIndices []int
	for i, r := range residual {
		zScore := math.Abs(r-m) / sd
		if zScore > threshold {
			anomalyIndices = append(anomalyIndices, i)
		}
	}

	return anomalyIndices
}

// HoltWinters implements Holt-Winters exponential smoothing
type HoltWinters struct {
	alpha  float64 // Level smoothing
	beta   float64 // Trend smoothing
	gamma  float64 // Seasonal smoothing
	period int     // Seasonal period
}

// NewHoltWinters creates a new Holt-Winters model
func NewHoltWinters(alpha, beta, gamma float64, period int) *HoltWinters {
	return &HoltWinters{
		alpha:  alpha,
		beta:   beta,
		gamma:  gamma,
		period: period,
	}
}

// Forecast forecasts the next n values
func (hw *HoltWinters) Forecast(values []float64, n int) []float64 {
	if len(values) < hw.period*2 {
		return nil
	}

	// Initialize components
	level := mean(values[:hw.period])
	trend := (mean(values[hw.period:2*hw.period]) - level) / float64(hw.period)

	seasonal := make([]float64, hw.period)
	for i := 0; i < hw.period; i++ {
		seasonal[i] = values[i] - level
	}

	// Fit model
	for i := hw.period; i < len(values); i++ {
		seasonIdx := i % hw.period
		prevLevel := level
		prevSeasonal := seasonal[seasonIdx]

		// Update level
		level = hw.alpha*(values[i]-prevSeasonal) + (1-hw.alpha)*(level+trend)

		// Update trend
		trend = hw.beta*(level-prevLevel) + (1-hw.beta)*trend

		// Update seasonal
		seasonal[seasonIdx] = hw.gamma*(values[i]-level) + (1-hw.gamma)*prevSeasonal
	}

	// Generate forecasts
	forecasts := make([]float64, n)
	for i := 0; i < n; i++ {
		seasonIdx := (len(values) + i) % hw.period
		forecasts[i] = level + float64(i+1)*trend + seasonal[seasonIdx]
	}

	return forecasts
}

// Helper function for moving average
func movingAverage(values []float64, window int) []float64 {
	n := len(values)
	result := make([]float64, n)

	halfWindow := window / 2

	for i := 0; i < n; i++ {
		start := i - halfWindow
		end := i + halfWindow + 1

		if start < 0 {
			start = 0
		}
		if end > n {
			end = n
		}

		sum := 0.0
		count := 0
		for j := start; j < end; j++ {
			sum += values[j]
			count++
		}

		result[i] = sum / float64(count)
	}

	return result
}
