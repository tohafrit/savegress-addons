package digitaltwin

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// Simulator handles digital twin simulation
type Simulator struct {
	mu        sync.RWMutex
	manager   *TwinManager
	running   map[string]*runningSimulation
}

// runningSimulation tracks an active simulation
type runningSimulation struct {
	config   *SimulationConfig
	cancel   context.CancelFunc
	result   *SimulationResult
	complete bool
}

// NewSimulator creates a new simulator
func NewSimulator(manager *TwinManager) *Simulator {
	return &Simulator{
		manager: manager,
		running: make(map[string]*runningSimulation),
	}
}

// StartSimulation starts a simulation for a twin
func (s *Simulator) StartSimulation(ctx context.Context, config *SimulationConfig) (string, error) {
	twin, err := s.manager.GetTwin(ctx, config.TwinID)
	if err != nil {
		return "", err
	}

	simID := fmt.Sprintf("sim-%s-%d", config.TwinID[:8], time.Now().UnixNano())

	simCtx, cancel := context.WithCancel(ctx)

	running := &runningSimulation{
		config: config,
		cancel: cancel,
		result: &SimulationResult{
			TwinID:       config.TwinID,
			StartTime:    time.Now(),
			Duration:     config.Duration,
			StateHistory: make([]StateSnapshot, 0),
			Metrics:      make(map[string]float64),
			Alerts:       make([]SimulationAlert, 0),
		},
	}

	s.mu.Lock()
	s.running[simID] = running
	s.mu.Unlock()

	go s.runSimulation(simCtx, simID, twin, running)

	return simID, nil
}

// runSimulation executes the simulation
func (s *Simulator) runSimulation(ctx context.Context, simID string, twin *DigitalTwin, running *runningSimulation) {
	config := running.config
	result := running.result

	timeStep := config.TimeStep
	if timeStep == 0 {
		timeStep = time.Second
	}

	steps := int(config.Duration / timeStep)
	result.TimeSteps = steps

	// Initialize state from twin or config
	currentState := make(map[string]interface{})
	for key, value := range twin.Properties {
		currentState[key] = value
	}
	if config.InitialState != nil {
		for key, value := range config.InitialState {
			currentState[key] = value
		}
	}

	currentTelemetry := make(map[string]interface{})
	for name, point := range twin.Telemetry {
		currentTelemetry[name] = point.Value
	}

	twinState := twin.State

	for step := 0; step <= steps; step++ {
		select {
		case <-ctx.Done():
			// Simulation cancelled
			result.EndTime = time.Now()
			s.markComplete(simID)
			return
		default:
		}

		elapsed := time.Duration(step) * timeStep

		// Check for scenario events at this time
		for _, scenario := range config.Scenarios {
			for _, event := range scenario.Events {
				if event.Timestamp == elapsed {
					// Apply event
					twinState, currentState, currentTelemetry = s.applyEvent(event, twinState, currentState, currentTelemetry)
				}
			}
		}

		// Simulate physics/behavior based on model
		if twin.Model != nil {
			currentTelemetry = s.simulatePhysics(twin.Model, currentState, currentTelemetry, timeStep)
		}

		// Check for alert conditions
		alerts := s.checkAlertConditions(elapsed, currentTelemetry, config.Parameters)
		result.Alerts = append(result.Alerts, alerts...)

		// Record state snapshot (at intervals to avoid huge history)
		if step%(steps/100+1) == 0 {
			snapshot := StateSnapshot{
				Timestamp:  elapsed,
				State:      twinState,
				Properties: copyMap(currentState),
				Telemetry:  copyMap(currentTelemetry),
			}
			result.StateHistory = append(result.StateHistory, snapshot)
		}

		// Simulate time passing (accelerated)
		time.Sleep(time.Millisecond)
	}

	result.EndTime = time.Now()

	// Calculate metrics
	result.Metrics = s.calculateMetrics(result)
	result.Summary = s.generateSummary(result)

	s.markComplete(simID)
}

// applyEvent applies a simulation event
func (s *Simulator) applyEvent(event SimulationEvent, state TwinState, props, telemetry map[string]interface{}) (TwinState, map[string]interface{}, map[string]interface{}) {
	switch event.Type {
	case "state_change":
		if newState, ok := event.Properties["state"].(string); ok {
			state = TwinState(newState)
		}
	case "property_update":
		for key, value := range event.Properties {
			props[key] = value
		}
	case "telemetry_spike":
		for key, value := range event.Properties {
			if factor, ok := value.(float64); ok {
				if current, ok := telemetry[key].(float64); ok {
					telemetry[key] = current * factor
				}
			}
		}
	case "failure":
		state = TwinStateFailed
	case "maintenance":
		state = TwinStateMaintenance
	case "recovery":
		state = TwinStateOnline
	}

	return state, props, telemetry
}

// simulatePhysics simulates physical behavior based on model
func (s *Simulator) simulatePhysics(model *TwinModel, props, telemetry map[string]interface{}, timeStep time.Duration) map[string]interface{} {
	result := copyMap(telemetry)

	for _, telDef := range model.Telemetry {
		currentValue, ok := result[telDef.Name].(float64)
		if !ok {
			continue
		}

		// Add random variation
		variation := (rand.Float64() - 0.5) * 0.02 * currentValue
		newValue := currentValue + variation

		// Apply min/max constraints
		if telDef.Min != nil && newValue < *telDef.Min {
			newValue = *telDef.Min
		}
		if telDef.Max != nil && newValue > *telDef.Max {
			newValue = *telDef.Max
		}

		result[telDef.Name] = newValue
	}

	return result
}

// checkAlertConditions checks for alert conditions
func (s *Simulator) checkAlertConditions(elapsed time.Duration, telemetry map[string]interface{}, params map[string]interface{}) []SimulationAlert {
	alerts := make([]SimulationAlert, 0)

	if params == nil {
		return alerts
	}

	thresholds, ok := params["thresholds"].(map[string]interface{})
	if !ok {
		return alerts
	}

	for name, threshold := range thresholds {
		thresholdValue, ok := threshold.(float64)
		if !ok {
			continue
		}

		actualValue, ok := telemetry[name].(float64)
		if !ok {
			continue
		}

		if actualValue > thresholdValue {
			alerts = append(alerts, SimulationAlert{
				Timestamp:   elapsed,
				Severity:    "warning",
				Message:     fmt.Sprintf("%s exceeded threshold", name),
				Property:    name,
				Threshold:   thresholdValue,
				ActualValue: actualValue,
			})
		}
	}

	return alerts
}

// calculateMetrics calculates simulation metrics
func (s *Simulator) calculateMetrics(result *SimulationResult) map[string]float64 {
	metrics := make(map[string]float64)

	if len(result.StateHistory) == 0 {
		return metrics
	}

	// Calculate uptime percentage
	onlineCount := 0
	for _, snapshot := range result.StateHistory {
		if snapshot.State == TwinStateOnline {
			onlineCount++
		}
	}
	metrics["uptime_percentage"] = float64(onlineCount) / float64(len(result.StateHistory)) * 100

	// Calculate alert rate
	metrics["alert_count"] = float64(len(result.Alerts))
	if result.Duration.Hours() > 0 {
		metrics["alerts_per_hour"] = float64(len(result.Alerts)) / result.Duration.Hours()
	}

	// Calculate telemetry statistics for first snapshot's telemetry keys
	if len(result.StateHistory) > 0 {
		firstSnapshot := result.StateHistory[0]
		for name := range firstSnapshot.Telemetry {
			values := make([]float64, 0)
			for _, snapshot := range result.StateHistory {
				if v, ok := snapshot.Telemetry[name].(float64); ok {
					values = append(values, v)
				}
			}

			if len(values) > 0 {
				metrics[name+"_min"] = min(values)
				metrics[name+"_max"] = max(values)
				metrics[name+"_avg"] = avg(values)
				metrics[name+"_stddev"] = stddev(values)
			}
		}
	}

	return metrics
}

// generateSummary generates a summary of the simulation
func (s *Simulator) generateSummary(result *SimulationResult) string {
	uptime := result.Metrics["uptime_percentage"]
	alertCount := int(result.Metrics["alert_count"])

	return fmt.Sprintf(
		"Simulation completed: %d time steps over %v. Uptime: %.1f%%. Alerts: %d.",
		result.TimeSteps,
		result.Duration,
		uptime,
		alertCount,
	)
}

// markComplete marks a simulation as complete
func (s *Simulator) markComplete(simID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if running, ok := s.running[simID]; ok {
		running.complete = true
	}
}

// StopSimulation stops a running simulation
func (s *Simulator) StopSimulation(simID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	running, ok := s.running[simID]
	if !ok {
		return fmt.Errorf("simulation %s not found", simID)
	}

	if running.cancel != nil {
		running.cancel()
	}

	return nil
}

// GetSimulationResult gets the result of a simulation
func (s *Simulator) GetSimulationResult(simID string) (*SimulationResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	running, ok := s.running[simID]
	if !ok {
		return nil, fmt.Errorf("simulation %s not found", simID)
	}

	return running.result, nil
}

// IsSimulationComplete checks if a simulation is complete
func (s *Simulator) IsSimulationComplete(simID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	running, ok := s.running[simID]
	if !ok {
		return false
	}

	return running.complete
}

// ListSimulations lists all simulations
func (s *Simulator) ListSimulations() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.running))
	for id := range s.running {
		ids = append(ids, id)
	}

	return ids
}

// CleanupOldSimulations removes completed simulations older than the given duration
func (s *Simulator) CleanupOldSimulations(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, running := range s.running {
		if running.complete && running.result.EndTime.Before(cutoff) {
			delete(s.running, id)
			removed++
		}
	}

	return removed
}

// RunWhatIfAnalysis runs a what-if analysis on a twin
func (s *Simulator) RunWhatIfAnalysis(ctx context.Context, twinID string, scenarios []SimulationScenario) ([]*SimulationResult, error) {
	results := make([]*SimulationResult, 0, len(scenarios))

	for _, scenario := range scenarios {
		config := &SimulationConfig{
			TwinID:    twinID,
			Duration:  24 * time.Hour,
			TimeStep:  time.Minute,
			Scenarios: []SimulationScenario{scenario},
		}

		simID, err := s.StartSimulation(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to start simulation for scenario %s: %w", scenario.Name, err)
		}

		// Wait for completion
		for !s.IsSimulationComplete(simID) {
			select {
			case <-ctx.Done():
				return results, ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}

		result, err := s.GetSimulationResult(simID)
		if err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}

// Helper functions

func copyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

func min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	result := values[0]
	for _, v := range values[1:] {
		if v < result {
			result = v
		}
	}
	return result
}

func max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	result := values[0]
	for _, v := range values[1:] {
		if v > result {
			result = v
		}
	}
	return result
}

func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stddev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := avg(values)
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(values)))
}
