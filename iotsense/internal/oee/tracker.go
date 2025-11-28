package oee

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Tracker manages Overall Equipment Effectiveness tracking
type Tracker struct {
	config       *Config
	equipment    map[string]*Equipment
	shifts       map[string]*Shift
	oeeData      map[string]*OEEData // equipment_id:date -> OEE data
	mu           sync.RWMutex
	running      bool
	stopCh       chan struct{}
}

// Config holds OEE tracker configuration
type Config struct {
	CalculationInterval time.Duration `json:"calculation_interval"`
	ShiftDuration       time.Duration `json:"shift_duration"`
	IdealCycleTime      time.Duration `json:"ideal_cycle_time"`
	TargetOEE           float64       `json:"target_oee"`
	TargetAvailability  float64       `json:"target_availability"`
	TargetPerformance   float64       `json:"target_performance"`
	TargetQuality       float64       `json:"target_quality"`
}

// Equipment represents a piece of manufacturing equipment
type Equipment struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Type            string            `json:"type"`
	Line            string            `json:"line"`
	Area            string            `json:"area"`
	IdealCycleTime  time.Duration     `json:"ideal_cycle_time"`
	PlannedRuntime  time.Duration     `json:"planned_runtime"`
	Tags            map[string]string `json:"tags,omitempty"`
	CurrentState    EquipmentState    `json:"current_state"`
	StateChangedAt  time.Time         `json:"state_changed_at"`
	CreatedAt       time.Time         `json:"created_at"`
}

// EquipmentState represents the current state of equipment
type EquipmentState string

const (
	StateRunning     EquipmentState = "running"
	StateIdle        EquipmentState = "idle"
	StatePlannedDown EquipmentState = "planned_down"
	StateUnplannedDown EquipmentState = "unplanned_down"
	StateSetup       EquipmentState = "setup"
	StateChangeover  EquipmentState = "changeover"
	StateMaintenance EquipmentState = "maintenance"
)

// Shift represents a work shift
type Shift struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	Duration    time.Duration `json:"duration"`
	PlannedBreaks []Break     `json:"planned_breaks"`
}

// Break represents a planned break
type Break struct {
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Type      string        `json:"type"` // break, lunch, meeting
}

// OEEData contains OEE calculation data
type OEEData struct {
	EquipmentID string    `json:"equipment_id"`
	Date        time.Time `json:"date"`
	ShiftID     string    `json:"shift_id,omitempty"`

	// Time metrics (in minutes)
	PlannedProductionTime float64 `json:"planned_production_time"`
	ActualRunTime         float64 `json:"actual_run_time"`
	PlannedDowntime       float64 `json:"planned_downtime"`
	UnplannedDowntime     float64 `json:"unplanned_downtime"`
	SetupTime             float64 `json:"setup_time"`

	// Production metrics
	TotalParts       int     `json:"total_parts"`
	GoodParts        int     `json:"good_parts"`
	DefectParts      int     `json:"defect_parts"`
	ReworkParts      int     `json:"rework_parts"`
	IdealCycleTime   float64 `json:"ideal_cycle_time_seconds"`
	ActualCycleTime  float64 `json:"actual_cycle_time_seconds"`

	// OEE metrics (0-1 scale)
	Availability    float64 `json:"availability"`
	Performance     float64 `json:"performance"`
	Quality         float64 `json:"quality"`
	OEE             float64 `json:"oee"`

	// Losses
	AvailabilityLoss float64        `json:"availability_loss"`
	PerformanceLoss  float64        `json:"performance_loss"`
	QualityLoss      float64        `json:"quality_loss"`
	LossCategories   []LossCategory `json:"loss_categories,omitempty"`

	// Metadata
	CalculatedAt time.Time `json:"calculated_at"`
}

// LossCategory represents a category of OEE loss
type LossCategory struct {
	Category    string  `json:"category"`
	Type        string  `json:"type"` // availability, performance, quality
	Duration    float64 `json:"duration_minutes"`
	Count       int     `json:"count,omitempty"`
	Description string  `json:"description,omitempty"`
}

// DowntimeEvent represents a downtime event
type DowntimeEvent struct {
	ID          string         `json:"id"`
	EquipmentID string         `json:"equipment_id"`
	Type        DowntimeType   `json:"type"`
	Reason      string         `json:"reason"`
	Category    string         `json:"category"`
	StartTime   time.Time      `json:"start_time"`
	EndTime     *time.Time     `json:"end_time,omitempty"`
	Duration    time.Duration  `json:"duration"`
	Planned     bool           `json:"planned"`
	Notes       string         `json:"notes,omitempty"`
}

// DowntimeType represents the type of downtime
type DowntimeType string

const (
	DowntypeBreakdown  DowntimeType = "breakdown"
	DowntypeSetup      DowntimeType = "setup"
	DowntypeChangeover DowntimeType = "changeover"
	DowntypeMaterial   DowntimeType = "material_shortage"
	DowntypeOperator   DowntimeType = "operator_shortage"
	DowntypePlanned    DowntimeType = "planned_maintenance"
	DowntypeOther      DowntimeType = "other"
)

// ProductionCount represents a production count event
type ProductionCount struct {
	EquipmentID string    `json:"equipment_id"`
	Timestamp   time.Time `json:"timestamp"`
	Count       int       `json:"count"`
	ProductID   string    `json:"product_id,omitempty"`
	Quality     string    `json:"quality"` // good, defect, rework
	BatchID     string    `json:"batch_id,omitempty"`
}

// NewTracker creates a new OEE tracker
func NewTracker(config *Config) *Tracker {
	if config.CalculationInterval == 0 {
		config.CalculationInterval = 5 * time.Minute
	}
	if config.ShiftDuration == 0 {
		config.ShiftDuration = 8 * time.Hour
	}
	if config.TargetOEE == 0 {
		config.TargetOEE = 0.85
	}
	if config.TargetAvailability == 0 {
		config.TargetAvailability = 0.90
	}
	if config.TargetPerformance == 0 {
		config.TargetPerformance = 0.95
	}
	if config.TargetQuality == 0 {
		config.TargetQuality = 0.99
	}

	return &Tracker{
		config:    config,
		equipment: make(map[string]*Equipment),
		shifts:    make(map[string]*Shift),
		oeeData:   make(map[string]*OEEData),
		stopCh:    make(chan struct{}),
	}
}

// Start starts the OEE tracker
func (t *Tracker) Start(ctx context.Context) error {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return nil
	}
	t.running = true
	t.mu.Unlock()

	go t.calculationLoop(ctx)
	return nil
}

// Stop stops the OEE tracker
func (t *Tracker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running {
		close(t.stopCh)
		t.running = false
	}
}

func (t *Tracker) calculationLoop(ctx context.Context) {
	ticker := time.NewTicker(t.config.CalculationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.stopCh:
			return
		case <-ticker.C:
			t.calculateAllOEE()
		}
	}
}

// RegisterEquipment registers equipment for OEE tracking
func (t *Tracker) RegisterEquipment(equipment *Equipment) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if equipment.IdealCycleTime == 0 {
		equipment.IdealCycleTime = t.config.IdealCycleTime
	}
	if equipment.PlannedRuntime == 0 {
		equipment.PlannedRuntime = t.config.ShiftDuration
	}
	equipment.CurrentState = StateIdle
	equipment.StateChangedAt = time.Now()
	equipment.CreatedAt = time.Now()

	t.equipment[equipment.ID] = equipment
}

// RecordStateChange records an equipment state change
func (t *Tracker) RecordStateChange(equipmentID string, newState EquipmentState) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	equipment, ok := t.equipment[equipmentID]
	if !ok {
		return fmt.Errorf("equipment not found: %s", equipmentID)
	}

	now := time.Now()
	stateDuration := now.Sub(equipment.StateChangedAt)

	// Update OEE data based on state duration
	key := t.getOEEKey(equipmentID, now)
	data := t.getOrCreateOEEData(equipmentID, now, key)

	// Add time to appropriate bucket
	switch equipment.CurrentState {
	case StateRunning:
		data.ActualRunTime += stateDuration.Minutes()
	case StatePlannedDown, StateMaintenance:
		data.PlannedDowntime += stateDuration.Minutes()
	case StateUnplannedDown:
		data.UnplannedDowntime += stateDuration.Minutes()
	case StateSetup, StateChangeover:
		data.SetupTime += stateDuration.Minutes()
	}

	equipment.CurrentState = newState
	equipment.StateChangedAt = now

	return nil
}

// RecordDowntime records a downtime event
func (t *Tracker) RecordDowntime(event *DowntimeEvent) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	_, ok := t.equipment[event.EquipmentID]
	if !ok {
		return fmt.Errorf("equipment not found: %s", event.EquipmentID)
	}

	if event.ID == "" {
		event.ID = fmt.Sprintf("dt_%d", time.Now().UnixNano())
	}

	// Calculate duration if end time is set
	if event.EndTime != nil {
		event.Duration = event.EndTime.Sub(event.StartTime)
	}

	// Update OEE data
	key := t.getOEEKey(event.EquipmentID, event.StartTime)
	data := t.getOrCreateOEEData(event.EquipmentID, event.StartTime, key)

	if event.Planned {
		data.PlannedDowntime += event.Duration.Minutes()
	} else {
		data.UnplannedDowntime += event.Duration.Minutes()
	}

	data.LossCategories = append(data.LossCategories, LossCategory{
		Category:    event.Category,
		Type:        "availability",
		Duration:    event.Duration.Minutes(),
		Description: event.Reason,
	})

	return nil
}

// RecordProduction records production counts
func (t *Tracker) RecordProduction(count *ProductionCount) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	_, ok := t.equipment[count.EquipmentID]
	if !ok {
		return fmt.Errorf("equipment not found: %s", count.EquipmentID)
	}

	key := t.getOEEKey(count.EquipmentID, count.Timestamp)
	data := t.getOrCreateOEEData(count.EquipmentID, count.Timestamp, key)

	data.TotalParts += count.Count

	switch count.Quality {
	case "good":
		data.GoodParts += count.Count
	case "defect":
		data.DefectParts += count.Count
	case "rework":
		data.ReworkParts += count.Count
	default:
		data.GoodParts += count.Count // Assume good if not specified
	}

	return nil
}

func (t *Tracker) getOEEKey(equipmentID string, ts time.Time) string {
	return fmt.Sprintf("%s:%s", equipmentID, ts.Format("2006-01-02"))
}

func (t *Tracker) getOrCreateOEEData(equipmentID string, ts time.Time, key string) *OEEData {
	data, ok := t.oeeData[key]
	if !ok {
		equipment := t.equipment[equipmentID]

		data = &OEEData{
			EquipmentID:           equipmentID,
			Date:                  ts.Truncate(24 * time.Hour),
			PlannedProductionTime: equipment.PlannedRuntime.Minutes(),
			IdealCycleTime:        equipment.IdealCycleTime.Seconds(),
		}
		t.oeeData[key] = data
	}
	return data
}

func (t *Tracker) calculateAllOEE() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for key, data := range t.oeeData {
		t.calculateOEE(data)
		t.oeeData[key] = data
	}
}

func (t *Tracker) calculateOEE(data *OEEData) {
	// Calculate Availability
	// Availability = Run Time / Planned Production Time
	availableTime := data.PlannedProductionTime - data.PlannedDowntime
	if availableTime > 0 {
		data.Availability = data.ActualRunTime / availableTime
	} else {
		data.Availability = 0
	}
	data.AvailabilityLoss = 1 - data.Availability

	// Calculate Performance
	// Performance = (Ideal Cycle Time × Total Count) / Run Time
	if data.ActualRunTime > 0 && data.IdealCycleTime > 0 {
		idealRunTime := float64(data.TotalParts) * data.IdealCycleTime / 60 // Convert to minutes
		data.Performance = idealRunTime / data.ActualRunTime
		if data.Performance > 1 {
			data.Performance = 1 // Cap at 100%
		}
	} else {
		data.Performance = 0
	}
	data.PerformanceLoss = 1 - data.Performance

	// Calculate actual cycle time
	if data.TotalParts > 0 && data.ActualRunTime > 0 {
		data.ActualCycleTime = (data.ActualRunTime * 60) / float64(data.TotalParts)
	}

	// Calculate Quality
	// Quality = Good Count / Total Count
	if data.TotalParts > 0 {
		data.Quality = float64(data.GoodParts) / float64(data.TotalParts)
	} else {
		data.Quality = 1 // No production, assume 100% quality
	}
	data.QualityLoss = 1 - data.Quality

	// Calculate OEE
	// OEE = Availability × Performance × Quality
	data.OEE = data.Availability * data.Performance * data.Quality

	// Ensure values are within bounds
	data.Availability = clamp(data.Availability, 0, 1)
	data.Performance = clamp(data.Performance, 0, 1)
	data.Quality = clamp(data.Quality, 0, 1)
	data.OEE = clamp(data.OEE, 0, 1)

	data.CalculatedAt = time.Now()
}

// GetOEE returns OEE data for an equipment and date
func (t *Tracker) GetOEE(equipmentID string, date time.Time) (*OEEData, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	key := t.getOEEKey(equipmentID, date)
	data, ok := t.oeeData[key]
	return data, ok
}

// GetOEERange returns OEE data for a date range
func (t *Tracker) GetOEERange(equipmentID string, startDate, endDate time.Time) []*OEEData {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var results []*OEEData

	for date := startDate; !date.After(endDate); date = date.Add(24 * time.Hour) {
		key := t.getOEEKey(equipmentID, date)
		if data, ok := t.oeeData[key]; ok {
			results = append(results, data)
		}
	}

	return results
}

// GetOEESummary returns a summary of OEE metrics
func (t *Tracker) GetOEESummary(equipmentID string, period time.Duration) *OEESummary {
	t.mu.RLock()
	defer t.mu.RUnlock()

	endDate := time.Now()
	startDate := endDate.Add(-period)

	summary := &OEESummary{
		EquipmentID: equipmentID,
		StartDate:   startDate,
		EndDate:     endDate,
	}

	var count int
	var totalOEE, totalAvail, totalPerf, totalQual float64

	for date := startDate; !date.After(endDate); date = date.Add(24 * time.Hour) {
		key := t.getOEEKey(equipmentID, date)
		if data, ok := t.oeeData[key]; ok {
			totalOEE += data.OEE
			totalAvail += data.Availability
			totalPerf += data.Performance
			totalQual += data.Quality
			summary.TotalParts += data.TotalParts
			summary.GoodParts += data.GoodParts
			summary.DefectParts += data.DefectParts
			summary.TotalDowntime += data.UnplannedDowntime
			count++
		}
	}

	if count > 0 {
		summary.AverageOEE = totalOEE / float64(count)
		summary.AverageAvailability = totalAvail / float64(count)
		summary.AveragePerformance = totalPerf / float64(count)
		summary.AverageQuality = totalQual / float64(count)
	}

	// Check against targets
	summary.MeetsOEETarget = summary.AverageOEE >= t.config.TargetOEE
	summary.MeetsAvailabilityTarget = summary.AverageAvailability >= t.config.TargetAvailability
	summary.MeetsPerformanceTarget = summary.AveragePerformance >= t.config.TargetPerformance
	summary.MeetsQualityTarget = summary.AverageQuality >= t.config.TargetQuality

	return summary
}

// OEESummary contains summarized OEE data
type OEESummary struct {
	EquipmentID             string    `json:"equipment_id"`
	StartDate               time.Time `json:"start_date"`
	EndDate                 time.Time `json:"end_date"`
	AverageOEE              float64   `json:"average_oee"`
	AverageAvailability     float64   `json:"average_availability"`
	AveragePerformance      float64   `json:"average_performance"`
	AverageQuality          float64   `json:"average_quality"`
	TotalParts              int       `json:"total_parts"`
	GoodParts               int       `json:"good_parts"`
	DefectParts             int       `json:"defect_parts"`
	TotalDowntime           float64   `json:"total_downtime_minutes"`
	MeetsOEETarget          bool      `json:"meets_oee_target"`
	MeetsAvailabilityTarget bool      `json:"meets_availability_target"`
	MeetsPerformanceTarget  bool      `json:"meets_performance_target"`
	MeetsQualityTarget      bool      `json:"meets_quality_target"`
}

// GetStats returns overall OEE statistics
func (t *Tracker) GetStats() *OEEStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := &OEEStats{
		TotalEquipment: len(t.equipment),
		ByEquipment:    make(map[string]float64),
	}

	today := time.Now().Truncate(24 * time.Hour)

	for equipmentID := range t.equipment {
		key := t.getOEEKey(equipmentID, today)
		if data, ok := t.oeeData[key]; ok {
			stats.ByEquipment[equipmentID] = data.OEE
			stats.TotalOEE += data.OEE

			if data.OEE >= t.config.TargetOEE {
				stats.EquipmentMeetingTarget++
			}
		}
	}

	if len(stats.ByEquipment) > 0 {
		stats.AverageOEE = stats.TotalOEE / float64(len(stats.ByEquipment))
	}

	return stats
}

// OEEStats contains overall OEE statistics
type OEEStats struct {
	TotalEquipment         int                `json:"total_equipment"`
	AverageOEE             float64            `json:"average_oee"`
	TotalOEE               float64            `json:"total_oee"`
	EquipmentMeetingTarget int                `json:"equipment_meeting_target"`
	ByEquipment            map[string]float64 `json:"by_equipment"`
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
