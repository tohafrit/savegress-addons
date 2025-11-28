package anomaly

import (
	"math"
)

// StatisticalDetector uses Z-score and MAD for anomaly detection
type StatisticalDetector struct {
	threshold float64 // Number of standard deviations
}

// NewStatisticalDetector creates a new statistical detector
func NewStatisticalDetector(threshold float64) *StatisticalDetector {
	return &StatisticalDetector{
		threshold: threshold,
	}
}

// Detect checks if a value is anomalous using statistical methods
func (d *StatisticalDetector) Detect(value float64, baseline *MetricBaseline) (score float64, isAnomaly bool, anomalyType AnomalyType) {
	if baseline.StdDev == 0 {
		// No variance - can't detect anomalies
		return 0, false, ""
	}

	// Calculate Z-score
	zScore := (value - baseline.Mean) / baseline.StdDev
	absZScore := math.Abs(zScore)

	// Convert Z-score to anomaly score (0-1)
	// Using sigmoid-like transformation
	score = 1 - 1/(1+math.Exp(absZScore-d.threshold))

	// Check if it exceeds threshold
	isAnomaly = absZScore > d.threshold

	if isAnomaly {
		if zScore > 0 {
			anomalyType = AnomalySpike
		} else {
			anomalyType = AnomalyDrop
		}
	}

	return score, isAnomaly, anomalyType
}

// DetectWithMAD uses Median Absolute Deviation for more robust detection
func (d *StatisticalDetector) DetectWithMAD(value float64, values []float64) (score float64, isAnomaly bool, anomalyType AnomalyType) {
	if len(values) < 3 {
		return 0, false, ""
	}

	// Calculate median
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sortFloat64s(sorted)
	median := sorted[len(sorted)/2]

	// Calculate MAD
	deviations := make([]float64, len(values))
	for i, v := range values {
		deviations[i] = math.Abs(v - median)
	}
	sortFloat64s(deviations)
	mad := deviations[len(deviations)/2]

	if mad == 0 {
		return 0, false, ""
	}

	// Calculate modified Z-score
	// 0.6745 is the scaling factor for consistency with normal distribution
	modZScore := 0.6745 * (value - median) / mad
	absModZScore := math.Abs(modZScore)

	// Standard threshold for modified Z-score is 3.5
	threshold := d.threshold * 1.17 // Adjust threshold for MAD

	score = 1 - 1/(1+math.Exp(absModZScore-threshold))
	isAnomaly = absModZScore > threshold

	if isAnomaly {
		if modZScore > 0 {
			anomalyType = AnomalySpike
		} else {
			anomalyType = AnomalyDrop
		}
	}

	return score, isAnomaly, anomalyType
}

// GrubbsTest performs Grubbs' test for outliers
type GrubbsTest struct {
	significance float64 // 0.05 for 95% confidence
}

// NewGrubbsTest creates a new Grubbs test
func NewGrubbsTest(significance float64) *GrubbsTest {
	return &GrubbsTest{
		significance: significance,
	}
}

// Test checks if a value is an outlier using Grubbs' test
func (g *GrubbsTest) Test(value float64, values []float64) bool {
	if len(values) < 7 { // Grubbs' test needs at least 7 observations
		return false
	}

	// Calculate mean and std dev
	m := mean(values)
	s := stdDev(values, m)

	if s == 0 {
		return false
	}

	// Calculate Grubbs' statistic
	G := math.Abs(value-m) / s

	// Critical value approximation (simplified)
	// For proper implementation, use t-distribution
	n := float64(len(values))
	tCritical := 2.5 // Approximate for alpha=0.05

	// Grubbs' critical value
	criticalValue := ((n - 1) / math.Sqrt(n)) * math.Sqrt(tCritical*tCritical/(n-2+tCritical*tCritical))

	return G > criticalValue
}

// IQRDetector uses Interquartile Range for outlier detection
type IQRDetector struct {
	multiplier float64 // Usually 1.5 for outliers, 3.0 for extreme outliers
}

// NewIQRDetector creates a new IQR detector
func NewIQRDetector(multiplier float64) *IQRDetector {
	return &IQRDetector{
		multiplier: multiplier,
	}
}

// Detect checks if a value is an outlier using IQR method
func (d *IQRDetector) Detect(value float64, values []float64) (isOutlier bool, lowerBound, upperBound float64) {
	if len(values) < 4 {
		return false, 0, 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sortFloat64s(sorted)

	q1 := percentile(sorted, 25)
	q3 := percentile(sorted, 75)
	iqr := q3 - q1

	lowerBound = q1 - d.multiplier*iqr
	upperBound = q3 + d.multiplier*iqr

	isOutlier = value < lowerBound || value > upperBound
	return isOutlier, lowerBound, upperBound
}
