package maintenance

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/savegress/iotsense/pkg/models"
)

// PredictiveEngine handles predictive maintenance analysis
type PredictiveEngine struct {
	config          *Config
	deviceProfiles  map[string]*DeviceProfile
	predictions     map[string]*MaintenancePrediction
	healthScores    map[string]*HealthScore
	anomalyDetector *AnomalyDetector
	mu              sync.RWMutex
	running         bool
	stopCh          chan struct{}
}

// Config holds predictive maintenance configuration
type Config struct {
	AnalysisInterval   time.Duration `json:"analysis_interval"`
	HistoryWindow      time.Duration `json:"history_window"`
	PredictionHorizon  time.Duration `json:"prediction_horizon"`
	AnomalyThreshold   float64       `json:"anomaly_threshold"`
	DegradationRate    float64       `json:"degradation_rate"`    // Expected degradation rate
	HealthScoreWindow  time.Duration `json:"health_score_window"`
	MinDataPoints      int           `json:"min_data_points"`
}

// DeviceProfile contains learned patterns for a device
type DeviceProfile struct {
	DeviceID         string                  `json:"device_id"`
	DeviceType       models.DeviceType       `json:"device_type"`
	MetricBaselines  map[string]*MetricBaseline `json:"metric_baselines"`
	FailurePatterns  []FailurePattern        `json:"failure_patterns"`
	MaintenanceHistory []MaintenanceEvent    `json:"maintenance_history"`
	OperatingHours   float64                 `json:"operating_hours"`
	MTBF             float64                 `json:"mtbf"` // Mean Time Between Failures
	MTTR             float64                 `json:"mttr"` // Mean Time To Repair
	LastUpdated      time.Time               `json:"last_updated"`
}

// MetricBaseline represents baseline statistics for a metric
type MetricBaseline struct {
	Metric     string    `json:"metric"`
	Mean       float64   `json:"mean"`
	StdDev     float64   `json:"std_dev"`
	Min        float64   `json:"min"`
	Max        float64   `json:"max"`
	P5         float64   `json:"p5"`
	P95        float64   `json:"p95"`
	Samples    int       `json:"samples"`
	LastUpdate time.Time `json:"last_update"`
}

// FailurePattern represents a known failure pattern
type FailurePattern struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Indicators  []PatternIndicator     `json:"indicators"`
	Severity    string                 `json:"severity"`
	MTTR        float64                `json:"mttr_hours"`
	LeadTime    time.Duration          `json:"lead_time"` // Time before failure
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PatternIndicator represents an indicator for a failure pattern
type PatternIndicator struct {
	Metric      string  `json:"metric"`
	Condition   string  `json:"condition"` // increasing, decreasing, threshold, variance
	Threshold   float64 `json:"threshold"`
	Duration    string  `json:"duration"`
	Weight      float64 `json:"weight"`
}

// MaintenanceEvent represents a maintenance event
type MaintenanceEvent struct {
	ID          string    `json:"id"`
	DeviceID    string    `json:"device_id"`
	Type        string    `json:"type"` // preventive, corrective, predictive
	Description string    `json:"description"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Duration    float64   `json:"duration_hours"`
	Cost        float64   `json:"cost,omitempty"`
	PartsUsed   []string  `json:"parts_used,omitempty"`
	Technician  string    `json:"technician,omitempty"`
	Notes       string    `json:"notes,omitempty"`
}

// MaintenancePrediction represents a maintenance prediction
type MaintenancePrediction struct {
	DeviceID           string          `json:"device_id"`
	PredictedFailure   *time.Time      `json:"predicted_failure,omitempty"`
	Confidence         float64         `json:"confidence"`
	RemainingLife      time.Duration   `json:"remaining_life"`
	RiskLevel          string          `json:"risk_level"` // low, medium, high, critical
	PredictedIssues    []PredictedIssue `json:"predicted_issues"`
	RecommendedActions []string        `json:"recommended_actions"`
	AnalyzedAt         time.Time       `json:"analyzed_at"`
	NextAnalysis       time.Time       `json:"next_analysis"`
}

// PredictedIssue represents a predicted issue
type PredictedIssue struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Probability float64   `json:"probability"`
	Impact      string    `json:"impact"`
	ETA         time.Time `json:"eta"`
	Indicators  []string  `json:"indicators"`
}

// HealthScore represents device health assessment
type HealthScore struct {
	DeviceID      string             `json:"device_id"`
	OverallScore  float64            `json:"overall_score"` // 0-100
	Category      string             `json:"category"`      // excellent, good, fair, poor, critical
	ComponentScores map[string]float64 `json:"component_scores"`
	Trends        map[string]string  `json:"trends"` // improving, stable, degrading
	Issues        []HealthIssue      `json:"issues"`
	CalculatedAt  time.Time          `json:"calculated_at"`
}

// HealthIssue represents a health issue
type HealthIssue struct {
	Component   string  `json:"component"`
	Metric      string  `json:"metric"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Impact      float64 `json:"impact"` // Impact on health score
}

// NewPredictiveEngine creates a new predictive maintenance engine
func NewPredictiveEngine(config *Config) *PredictiveEngine {
	if config.AnalysisInterval == 0 {
		config.AnalysisInterval = 15 * time.Minute
	}
	if config.HistoryWindow == 0 {
		config.HistoryWindow = 30 * 24 * time.Hour
	}
	if config.PredictionHorizon == 0 {
		config.PredictionHorizon = 7 * 24 * time.Hour
	}
	if config.AnomalyThreshold == 0 {
		config.AnomalyThreshold = 2.5 // Standard deviations
	}
	if config.MinDataPoints == 0 {
		config.MinDataPoints = 100
	}

	return &PredictiveEngine{
		config:          config,
		deviceProfiles:  make(map[string]*DeviceProfile),
		predictions:     make(map[string]*MaintenancePrediction),
		healthScores:    make(map[string]*HealthScore),
		anomalyDetector: NewAnomalyDetector(config.AnomalyThreshold),
		stopCh:          make(chan struct{}),
	}
}

// Start starts the predictive engine
func (e *PredictiveEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = true
	e.mu.Unlock()

	go e.analysisLoop(ctx)
	return nil
}

// Stop stops the predictive engine
func (e *PredictiveEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		close(e.stopCh)
		e.running = false
	}
}

func (e *PredictiveEngine) analysisLoop(ctx context.Context) {
	ticker := time.NewTicker(e.config.AnalysisInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.runAnalysis(ctx)
		}
	}
}

func (e *PredictiveEngine) runAnalysis(ctx context.Context) {
	e.mu.Lock()
	deviceIDs := make([]string, 0, len(e.deviceProfiles))
	for id := range e.deviceProfiles {
		deviceIDs = append(deviceIDs, id)
	}
	e.mu.Unlock()

	for _, deviceID := range deviceIDs {
		select {
		case <-ctx.Done():
			return
		default:
			e.analyzeDevice(ctx, deviceID)
		}
	}
}

// ProcessTelemetry processes new telemetry for analysis
func (e *PredictiveEngine) ProcessTelemetry(point *models.TelemetryPoint) {
	e.mu.Lock()
	defer e.mu.Unlock()

	profile, ok := e.deviceProfiles[point.DeviceID]
	if !ok {
		return
	}

	// Update baseline statistics
	baseline, ok := profile.MetricBaselines[point.Metric]
	if !ok {
		baseline = &MetricBaseline{Metric: point.Metric}
		profile.MetricBaselines[point.Metric] = baseline
	}

	// Update running statistics
	value, ok := point.Value.(float64)
	if !ok {
		return
	}

	e.updateBaseline(baseline, value)

	// Check for anomalies
	if e.anomalyDetector.IsAnomaly(value, baseline.Mean, baseline.StdDev) {
		// Log anomaly or trigger alert
	}
}

func (e *PredictiveEngine) updateBaseline(baseline *MetricBaseline, value float64) {
	// Exponential moving average for online updates
	alpha := 0.1
	if baseline.Samples == 0 {
		baseline.Mean = value
		baseline.Min = value
		baseline.Max = value
		baseline.StdDev = 0
	} else {
		// Update mean
		delta := value - baseline.Mean
		baseline.Mean += alpha * delta

		// Update variance (Welford's online algorithm)
		baseline.StdDev = math.Sqrt((1-alpha)*baseline.StdDev*baseline.StdDev + alpha*delta*delta)

		// Update min/max
		if value < baseline.Min {
			baseline.Min = value
		}
		if value > baseline.Max {
			baseline.Max = value
		}
	}

	baseline.Samples++
	baseline.LastUpdate = time.Now()
}

// RegisterDevice registers a device for predictive maintenance
func (e *PredictiveEngine) RegisterDevice(device *models.Device) {
	e.mu.Lock()
	defer e.mu.Unlock()

	profile := &DeviceProfile{
		DeviceID:         device.ID,
		DeviceType:       device.Type,
		MetricBaselines:  make(map[string]*MetricBaseline),
		FailurePatterns:  e.getDefaultPatterns(device.Type),
		LastUpdated:      time.Now(),
	}

	e.deviceProfiles[device.ID] = profile
}

func (e *PredictiveEngine) getDefaultPatterns(deviceType models.DeviceType) []FailurePattern {
	patterns := []FailurePattern{
		{
			ID:          "overheating",
			Name:        "Overheating",
			Description: "Device temperature exceeding safe limits",
			Indicators: []PatternIndicator{
				{Metric: "temperature", Condition: "threshold", Threshold: 85, Weight: 1.0},
				{Metric: "temperature", Condition: "increasing", Threshold: 5, Duration: "1h", Weight: 0.8},
			},
			Severity: "high",
			LeadTime: 2 * time.Hour,
		},
		{
			ID:          "vibration_anomaly",
			Name:        "Vibration Anomaly",
			Description: "Abnormal vibration indicating mechanical wear",
			Indicators: []PatternIndicator{
				{Metric: "vibration", Condition: "variance", Threshold: 2.5, Weight: 1.0},
				{Metric: "vibration", Condition: "increasing", Threshold: 0.3, Duration: "24h", Weight: 0.7},
			},
			Severity: "medium",
			LeadTime: 24 * time.Hour,
		},
		{
			ID:          "power_degradation",
			Name:        "Power Degradation",
			Description: "Power consumption patterns indicating degradation",
			Indicators: []PatternIndicator{
				{Metric: "power", Condition: "increasing", Threshold: 10, Duration: "7d", Weight: 0.6},
				{Metric: "efficiency", Condition: "decreasing", Threshold: 5, Duration: "7d", Weight: 0.8},
			},
			Severity: "low",
			LeadTime: 168 * time.Hour, // 1 week
		},
	}

	return patterns
}

func (e *PredictiveEngine) analyzeDevice(ctx context.Context, deviceID string) {
	e.mu.Lock()
	profile := e.deviceProfiles[deviceID]
	if profile == nil {
		e.mu.Unlock()
		return
	}
	e.mu.Unlock()

	// Calculate health score
	healthScore := e.calculateHealthScore(profile)

	// Predict failures
	prediction := e.predictFailures(profile)

	e.mu.Lock()
	e.healthScores[deviceID] = healthScore
	e.predictions[deviceID] = prediction
	e.mu.Unlock()
}

func (e *PredictiveEngine) calculateHealthScore(profile *DeviceProfile) *HealthScore {
	score := &HealthScore{
		DeviceID:        profile.DeviceID,
		ComponentScores: make(map[string]float64),
		Trends:          make(map[string]string),
		Issues:          []HealthIssue{},
		CalculatedAt:    time.Now(),
	}

	var totalScore float64
	var componentCount int

	for metric, baseline := range profile.MetricBaselines {
		if baseline.Samples < e.config.MinDataPoints {
			continue
		}

		// Calculate component score based on deviation from baseline
		componentScore := 100.0

		// Check variance
		if baseline.StdDev > 0 {
			cv := baseline.StdDev / math.Abs(baseline.Mean) // Coefficient of variation
			if cv > 0.5 {
				componentScore -= 30
				score.Issues = append(score.Issues, HealthIssue{
					Metric:      metric,
					Severity:    "warning",
					Description: fmt.Sprintf("High variance in %s", metric),
					Impact:      30,
				})
			} else if cv > 0.3 {
				componentScore -= 15
			}
		}

		score.ComponentScores[metric] = math.Max(0, componentScore)
		totalScore += componentScore
		componentCount++
	}

	if componentCount > 0 {
		score.OverallScore = totalScore / float64(componentCount)
	} else {
		score.OverallScore = 100 // No data, assume healthy
	}

	// Determine category
	switch {
	case score.OverallScore >= 90:
		score.Category = "excellent"
	case score.OverallScore >= 75:
		score.Category = "good"
	case score.OverallScore >= 50:
		score.Category = "fair"
	case score.OverallScore >= 25:
		score.Category = "poor"
	default:
		score.Category = "critical"
	}

	return score
}

func (e *PredictiveEngine) predictFailures(profile *DeviceProfile) *MaintenancePrediction {
	prediction := &MaintenancePrediction{
		DeviceID:           profile.DeviceID,
		PredictedIssues:    []PredictedIssue{},
		RecommendedActions: []string{},
		AnalyzedAt:         time.Now(),
		NextAnalysis:       time.Now().Add(e.config.AnalysisInterval),
	}

	var highestProbability float64
	var earliestFailure *time.Time

	// Check each failure pattern
	for _, pattern := range profile.FailurePatterns {
		probability := e.evaluatePattern(pattern, profile)

		if probability > 0.3 { // Only consider significant probabilities
			eta := time.Now().Add(pattern.LeadTime)

			prediction.PredictedIssues = append(prediction.PredictedIssues, PredictedIssue{
				Type:        pattern.Name,
				Description: pattern.Description,
				Probability: probability,
				Impact:      pattern.Severity,
				ETA:         eta,
			})

			if probability > highestProbability {
				highestProbability = probability
			}

			if earliestFailure == nil || eta.Before(*earliestFailure) {
				earliestFailure = &eta
			}
		}
	}

	prediction.PredictedFailure = earliestFailure
	prediction.Confidence = highestProbability

	// Calculate remaining life estimate
	if earliestFailure != nil {
		prediction.RemainingLife = time.Until(*earliestFailure)
	} else {
		prediction.RemainingLife = e.config.PredictionHorizon
	}

	// Determine risk level
	switch {
	case highestProbability >= 0.8:
		prediction.RiskLevel = "critical"
		prediction.RecommendedActions = append(prediction.RecommendedActions, "Immediate maintenance required")
	case highestProbability >= 0.6:
		prediction.RiskLevel = "high"
		prediction.RecommendedActions = append(prediction.RecommendedActions, "Schedule maintenance within 24 hours")
	case highestProbability >= 0.4:
		prediction.RiskLevel = "medium"
		prediction.RecommendedActions = append(prediction.RecommendedActions, "Plan maintenance within this week")
	default:
		prediction.RiskLevel = "low"
		prediction.RecommendedActions = append(prediction.RecommendedActions, "Continue normal monitoring")
	}

	return prediction
}

func (e *PredictiveEngine) evaluatePattern(pattern FailurePattern, profile *DeviceProfile) float64 {
	if len(pattern.Indicators) == 0 {
		return 0
	}

	var totalWeight float64
	var weightedScore float64

	for _, indicator := range pattern.Indicators {
		baseline, ok := profile.MetricBaselines[indicator.Metric]
		if !ok || baseline.Samples < e.config.MinDataPoints {
			continue
		}

		indicatorScore := 0.0

		switch indicator.Condition {
		case "threshold":
			if baseline.Mean > indicator.Threshold {
				indicatorScore = 1.0
			} else {
				indicatorScore = baseline.Mean / indicator.Threshold
			}

		case "variance":
			if baseline.StdDev > 0 {
				cv := baseline.StdDev / math.Abs(baseline.Mean)
				if cv > indicator.Threshold/10 {
					indicatorScore = math.Min(cv/(indicator.Threshold/10), 1.0)
				}
			}

		case "increasing", "decreasing":
			// Would need historical trend data
			indicatorScore = 0.3 // Placeholder
		}

		weightedScore += indicatorScore * indicator.Weight
		totalWeight += indicator.Weight
	}

	if totalWeight == 0 {
		return 0
	}

	return weightedScore / totalWeight
}

// GetPrediction returns the maintenance prediction for a device
func (e *PredictiveEngine) GetPrediction(deviceID string) (*MaintenancePrediction, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	pred, ok := e.predictions[deviceID]
	return pred, ok
}

// GetHealthScore returns the health score for a device
func (e *PredictiveEngine) GetHealthScore(deviceID string) (*HealthScore, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	score, ok := e.healthScores[deviceID]
	return score, ok
}

// GetAllPredictions returns all maintenance predictions
func (e *PredictiveEngine) GetAllPredictions() []*MaintenancePrediction {
	e.mu.RLock()
	defer e.mu.RUnlock()

	predictions := make([]*MaintenancePrediction, 0, len(e.predictions))
	for _, p := range e.predictions {
		predictions = append(predictions, p)
	}

	// Sort by risk level and confidence
	sort.Slice(predictions, func(i, j int) bool {
		if predictions[i].RiskLevel != predictions[j].RiskLevel {
			riskOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
			return riskOrder[predictions[i].RiskLevel] < riskOrder[predictions[j].RiskLevel]
		}
		return predictions[i].Confidence > predictions[j].Confidence
	})

	return predictions
}

// RecordMaintenance records a maintenance event
func (e *PredictiveEngine) RecordMaintenance(event *MaintenanceEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()

	profile, ok := e.deviceProfiles[event.DeviceID]
	if !ok {
		return
	}

	profile.MaintenanceHistory = append(profile.MaintenanceHistory, *event)
	profile.LastUpdated = time.Now()

	// Update MTTR
	if event.CompletedAt != nil {
		event.Duration = event.CompletedAt.Sub(event.StartedAt).Hours()
	}
	e.updateMTTR(profile)
}

func (e *PredictiveEngine) updateMTTR(profile *DeviceProfile) {
	if len(profile.MaintenanceHistory) == 0 {
		return
	}

	var totalDuration float64
	var count int
	for _, event := range profile.MaintenanceHistory {
		if event.Duration > 0 {
			totalDuration += event.Duration
			count++
		}
	}

	if count > 0 {
		profile.MTTR = totalDuration / float64(count)
	}
}

// AnomalyDetector detects anomalies in telemetry data
type AnomalyDetector struct {
	threshold float64 // Number of standard deviations
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(threshold float64) *AnomalyDetector {
	return &AnomalyDetector{threshold: threshold}
}

// IsAnomaly checks if a value is an anomaly
func (d *AnomalyDetector) IsAnomaly(value, mean, stdDev float64) bool {
	if stdDev == 0 {
		return false
	}
	zScore := math.Abs(value-mean) / stdDev
	return zScore > d.threshold
}
