package anomaly

import (
	"time"
)

// AnomalyType represents the type of anomaly detected
type AnomalyType string

const (
	AnomalySpike    AnomalyType = "spike"    // Sudden increase
	AnomalyDrop     AnomalyType = "drop"     // Sudden decrease
	AnomalyTrend    AnomalyType = "trend"    // Long-term trend change
	AnomalySeasonal AnomalyType = "seasonal" // Seasonal pattern violation
	AnomalyMissing  AnomalyType = "missing"  // Missing data
	AnomalyOutlier  AnomalyType = "outlier"  // Statistical outlier
)

// Severity represents the severity level of an anomaly
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Anomaly represents a detected anomaly
type Anomaly struct {
	ID          string            `json:"id"`
	MetricName  string            `json:"metric_name"`
	Type        AnomalyType       `json:"type"`
	Severity    Severity          `json:"severity"`
	Value       float64           `json:"value"`
	Expected    Range             `json:"expected"`
	Score       float64           `json:"score"` // 0-1, higher = more anomalous
	DetectedAt  time.Time         `json:"detected_at"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels,omitempty"`
	Acknowledged bool             `json:"acknowledged"`
	AcknowledgedBy string         `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time     `json:"acknowledged_at,omitempty"`
}

// Range represents an expected value range
type Range struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// Algorithm represents an anomaly detection algorithm
type Algorithm string

const (
	AlgorithmStatistical Algorithm = "statistical" // Z-score, MAD
	AlgorithmSeasonal    Algorithm = "seasonal"    // STL decomposition
	AlgorithmML          Algorithm = "ml"          // Isolation Forest
)

// DetectorConfig holds configuration for anomaly detection
type DetectorConfig struct {
	Algorithms     []Algorithm   `json:"algorithms"`
	Sensitivity    string        `json:"sensitivity"` // low, medium, high
	BaselineWindow time.Duration `json:"baseline_window"`
	MinDataPoints  int           `json:"min_data_points"`
}

// MetricBaseline represents the baseline statistics for a metric
type MetricBaseline struct {
	MetricName    string            `json:"metric_name"`
	Labels        map[string]string `json:"labels,omitempty"`
	Mean          float64           `json:"mean"`
	StdDev        float64           `json:"std_dev"`
	Min           float64           `json:"min"`
	Max           float64           `json:"max"`
	P50           float64           `json:"p50"`
	P90           float64           `json:"p90"`
	P99           float64           `json:"p99"`
	DataPoints    int64             `json:"data_points"`
	LastUpdated   time.Time         `json:"last_updated"`
	SeasonalData  *SeasonalBaseline `json:"seasonal_data,omitempty"`
}

// SeasonalBaseline holds seasonal pattern data
type SeasonalBaseline struct {
	HourlyPattern  [24]float64 `json:"hourly_pattern"`  // Average by hour of day
	DailyPattern   [7]float64  `json:"daily_pattern"`   // Average by day of week
	HasSeasonality bool        `json:"has_seasonality"`
}

// DetectionResult represents the result of anomaly detection
type DetectionResult struct {
	IsAnomaly   bool        `json:"is_anomaly"`
	Anomaly     *Anomaly    `json:"anomaly,omitempty"`
	Score       float64     `json:"score"`
	Algorithms  []string    `json:"algorithms_triggered"`
	Explanation string      `json:"explanation"`
}
