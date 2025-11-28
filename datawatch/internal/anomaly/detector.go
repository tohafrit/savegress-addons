package anomaly

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/savegress/datawatch/internal/storage"
)

// Detector is the main anomaly detection coordinator
type Detector struct {
	config     DetectorConfig
	storage    storage.MetricStorage
	baselines  map[string]*MetricBaseline
	baselineMu sync.RWMutex

	// Detection algorithms
	statistical *StatisticalDetector
	seasonal    *SeasonalDetector

	// Anomaly storage
	anomalies   map[string]*Anomaly
	anomaliesMu sync.RWMutex

	// Callbacks
	onAnomaly func(*Anomaly)
}

// NewDetector creates a new anomaly detector
func NewDetector(cfg DetectorConfig, store storage.MetricStorage) *Detector {
	d := &Detector{
		config:    cfg,
		storage:   store,
		baselines: make(map[string]*MetricBaseline),
		anomalies: make(map[string]*Anomaly),
	}

	// Initialize detection algorithms
	sensitivity := getSensitivityThreshold(cfg.Sensitivity)
	d.statistical = NewStatisticalDetector(sensitivity)
	d.seasonal = NewSeasonalDetector(sensitivity)

	return d
}

// SetAnomalyCallback sets the callback function for detected anomalies
func (d *Detector) SetAnomalyCallback(fn func(*Anomaly)) {
	d.onAnomaly = fn
}

// Check checks a metric value for anomalies
func (d *Detector) Check(ctx context.Context, metric string, value float64, labels map[string]string, ts time.Time) *DetectionResult {
	key := metricKey(metric, labels)

	// Get or create baseline
	baseline := d.getOrCreateBaseline(ctx, metric, labels)
	if baseline == nil || baseline.DataPoints < int64(d.config.MinDataPoints) {
		// Not enough data for detection
		return &DetectionResult{
			IsAnomaly:   false,
			Score:       0,
			Explanation: "Insufficient baseline data",
		}
	}

	result := &DetectionResult{
		IsAnomaly:   false,
		Score:       0,
		Algorithms:  []string{},
	}

	var maxScore float64
	var triggeredAlgorithms []string
	var anomalyType AnomalyType

	// Run configured algorithms
	for _, algo := range d.config.Algorithms {
		switch algo {
		case AlgorithmStatistical:
			score, isAnomaly, aType := d.statistical.Detect(value, baseline)
			if score > maxScore {
				maxScore = score
				if isAnomaly {
					anomalyType = aType
				}
			}
			if isAnomaly {
				triggeredAlgorithms = append(triggeredAlgorithms, string(algo))
			}

		case AlgorithmSeasonal:
			if baseline.SeasonalData != nil && baseline.SeasonalData.HasSeasonality {
				score, isAnomaly, aType := d.seasonal.Detect(value, ts, baseline)
				if score > maxScore {
					maxScore = score
					if isAnomaly {
						anomalyType = aType
					}
				}
				if isAnomaly {
					triggeredAlgorithms = append(triggeredAlgorithms, string(algo))
				}
			}
		}
	}

	result.Score = maxScore
	result.Algorithms = triggeredAlgorithms

	if len(triggeredAlgorithms) > 0 {
		result.IsAnomaly = true

		anomaly := &Anomaly{
			ID:          uuid.New().String(),
			MetricName:  metric,
			Type:        anomalyType,
			Severity:    scoreSeverity(maxScore),
			Value:       value,
			Expected:    Range{Min: baseline.Mean - 2*baseline.StdDev, Max: baseline.Mean + 2*baseline.StdDev},
			Score:       maxScore,
			DetectedAt:  ts,
			Description: d.generateDescription(metric, value, baseline, anomalyType),
			Labels:      labels,
		}

		result.Anomaly = anomaly
		result.Explanation = anomaly.Description

		// Store and notify
		d.storeAnomaly(anomaly)
		if d.onAnomaly != nil {
			d.onAnomaly(anomaly)
		}
	} else {
		result.Explanation = "Value within expected range"
	}

	// Update baseline with new value
	d.updateBaseline(key, value, ts)

	return result
}

func (d *Detector) getOrCreateBaseline(ctx context.Context, metric string, labels map[string]string) *MetricBaseline {
	key := metricKey(metric, labels)

	d.baselineMu.RLock()
	baseline, exists := d.baselines[key]
	d.baselineMu.RUnlock()

	if exists && time.Since(baseline.LastUpdated) < time.Hour {
		return baseline
	}

	// Compute baseline from historical data
	baseline = d.computeBaseline(ctx, metric, labels)
	if baseline != nil {
		d.baselineMu.Lock()
		d.baselines[key] = baseline
		d.baselineMu.Unlock()
	}

	return baseline
}

func (d *Detector) computeBaseline(ctx context.Context, metric string, labels map[string]string) *MetricBaseline {
	to := time.Now()
	from := to.Add(-d.config.BaselineWindow)

	result, err := d.storage.QueryRange(ctx, metric, from, to, time.Hour, storage.AggregationAvg)
	if err != nil || len(result.Series) == 0 {
		return nil
	}

	// Collect all values
	var values []float64
	hourlySum := [24]float64{}
	hourlyCount := [24]int{}
	dailySum := [7]float64{}
	dailyCount := [7]int{}

	for _, series := range result.Series {
		for _, dp := range series.DataPoints {
			values = append(values, dp.Value)

			// Track hourly pattern
			hour := dp.Timestamp.Hour()
			hourlySum[hour] += dp.Value
			hourlyCount[hour]++

			// Track daily pattern
			day := int(dp.Timestamp.Weekday())
			dailySum[day] += dp.Value
			dailyCount[day]++
		}
	}

	if len(values) < d.config.MinDataPoints {
		return nil
	}

	baseline := &MetricBaseline{
		MetricName:  metric,
		Labels:      labels,
		DataPoints:  int64(len(values)),
		LastUpdated: time.Now(),
	}

	// Calculate statistics
	baseline.Mean = mean(values)
	baseline.StdDev = stdDev(values, baseline.Mean)
	baseline.Min = min(values)
	baseline.Max = max(values)

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sortFloat64s(sorted)
	baseline.P50 = percentile(sorted, 50)
	baseline.P90 = percentile(sorted, 90)
	baseline.P99 = percentile(sorted, 99)

	// Calculate seasonal patterns
	seasonal := &SeasonalBaseline{}
	for i := 0; i < 24; i++ {
		if hourlyCount[i] > 0 {
			seasonal.HourlyPattern[i] = hourlySum[i] / float64(hourlyCount[i])
		}
	}
	for i := 0; i < 7; i++ {
		if dailyCount[i] > 0 {
			seasonal.DailyPattern[i] = dailySum[i] / float64(dailyCount[i])
		}
	}

	// Check if there's significant seasonality
	seasonal.HasSeasonality = hasSignificantSeasonality(seasonal.HourlyPattern[:], baseline.Mean, baseline.StdDev)
	baseline.SeasonalData = seasonal

	return baseline
}

func (d *Detector) updateBaseline(key string, value float64, ts time.Time) {
	d.baselineMu.Lock()
	defer d.baselineMu.Unlock()

	baseline, exists := d.baselines[key]
	if !exists {
		return
	}

	// Exponential moving average update
	alpha := 0.01 // Small alpha for slow adaptation
	baseline.Mean = alpha*value + (1-alpha)*baseline.Mean
	baseline.DataPoints++
	baseline.LastUpdated = ts

	// Update seasonal data
	if baseline.SeasonalData != nil {
		hour := ts.Hour()
		baseline.SeasonalData.HourlyPattern[hour] = alpha*value + (1-alpha)*baseline.SeasonalData.HourlyPattern[hour]

		day := int(ts.Weekday())
		baseline.SeasonalData.DailyPattern[day] = alpha*value + (1-alpha)*baseline.SeasonalData.DailyPattern[day]
	}
}

func (d *Detector) storeAnomaly(anomaly *Anomaly) {
	d.anomaliesMu.Lock()
	defer d.anomaliesMu.Unlock()
	d.anomalies[anomaly.ID] = anomaly
}

func (d *Detector) generateDescription(metric string, value float64, baseline *MetricBaseline, aType AnomalyType) string {
	deviation := (value - baseline.Mean) / baseline.StdDev
	direction := "above"
	if deviation < 0 {
		direction = "below"
		deviation = -deviation
	}

	switch aType {
	case AnomalySpike:
		return fmt.Sprintf("%s spiked to %.2f (%.1f std devs %s mean of %.2f)", metric, value, deviation, direction, baseline.Mean)
	case AnomalyDrop:
		return fmt.Sprintf("%s dropped to %.2f (%.1f std devs %s mean of %.2f)", metric, value, deviation, direction, baseline.Mean)
	case AnomalySeasonal:
		return fmt.Sprintf("%s value %.2f deviates from expected seasonal pattern", metric, value)
	default:
		return fmt.Sprintf("%s anomaly detected: %.2f (expected %.2f ± %.2f)", metric, value, baseline.Mean, 2*baseline.StdDev)
	}
}

// GetAnomaly returns an anomaly by ID
func (d *Detector) GetAnomaly(id string) (*Anomaly, bool) {
	d.anomaliesMu.RLock()
	defer d.anomaliesMu.RUnlock()
	a, ok := d.anomalies[id]
	return a, ok
}

// ListAnomalies returns recent anomalies
func (d *Detector) ListAnomalies(limit int, includeAcknowledged bool) []*Anomaly {
	d.anomaliesMu.RLock()
	defer d.anomaliesMu.RUnlock()

	result := make([]*Anomaly, 0, len(d.anomalies))
	for _, a := range d.anomalies {
		if !includeAcknowledged && a.Acknowledged {
			continue
		}
		result = append(result, a)
	}

	// Sort by detection time (newest first)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].DetectedAt.After(result[i].DetectedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result
}

// AcknowledgeAnomaly marks an anomaly as acknowledged
func (d *Detector) AcknowledgeAnomaly(id, user string) error {
	d.anomaliesMu.Lock()
	defer d.anomaliesMu.Unlock()

	a, ok := d.anomalies[id]
	if !ok {
		return fmt.Errorf("anomaly not found: %s", id)
	}

	now := time.Now()
	a.Acknowledged = true
	a.AcknowledgedBy = user
	a.AcknowledgedAt = &now

	return nil
}

// Helper functions

func metricKey(metric string, labels map[string]string) string {
	if len(labels) == 0 {
		return metric
	}
	// Simple key generation
	key := metric
	for k, v := range labels {
		key += fmt.Sprintf(",%s=%s", k, v)
	}
	return key
}

func getSensitivityThreshold(sensitivity string) float64 {
	switch sensitivity {
	case "low":
		return 4.0 // 4 standard deviations
	case "high":
		return 2.0 // 2 standard deviations
	default: // medium
		return 3.0 // 3 standard deviations
	}
}

func scoreSeverity(score float64) Severity {
	switch {
	case score >= 0.9:
		return SeverityCritical
	case score >= 0.7:
		return SeverityHigh
	case score >= 0.5:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	var sumSquares float64
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(values)-1))
}

func min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func sortFloat64s(values []float64) {
	for i := 0; i < len(values)-1; i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := (p / 100.0) * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	fraction := index - float64(lower)
	return sorted[lower] + fraction*(sorted[upper]-sorted[lower])
}

func hasSignificantSeasonality(pattern []float64, mean, stdDev float64) bool {
	if stdDev == 0 {
		return false
	}
	// Check if pattern has significant variation
	var patternStdDev float64
	for _, v := range pattern {
		diff := v - mean
		patternStdDev += diff * diff
	}
	patternStdDev = math.Sqrt(patternStdDev / float64(len(pattern)))
	return patternStdDev > stdDev*0.3 // 30% of overall std dev
}
