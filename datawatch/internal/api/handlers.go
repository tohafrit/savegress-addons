package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/savegress/datawatch/internal/alerts"
	"github.com/savegress/datawatch/internal/anomaly"
	"github.com/savegress/datawatch/internal/metrics"
	"github.com/savegress/datawatch/internal/quality"
	"github.com/savegress/datawatch/internal/schema"
	"github.com/savegress/datawatch/internal/storage"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	metrics  *metrics.Engine
	anomaly  *anomaly.Detector
	storage  storage.MetricStorage
	quality  *quality.Monitor
	schema   *schema.Tracker
	alerts   *alerts.Engine
}

// NewHandlers creates new handlers
func NewHandlers(
	metricsEngine *metrics.Engine,
	anomalyDetector *anomaly.Detector,
	store storage.MetricStorage,
	qualityMonitor *quality.Monitor,
	schemaTracker *schema.Tracker,
	alertsEngine *alerts.Engine,
) *Handlers {
	return &Handlers{
		metrics: metricsEngine,
		anomaly: anomalyDetector,
		storage: store,
		quality: qualityMonitor,
		schema:  schemaTracker,
		alerts:  alertsEngine,
	}
}

// Response helpers

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Success: true, Data: data})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Success: false, Error: message})
}

// HealthCheck returns the health status
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// ListMetrics returns all available metrics
func (h *Handlers) ListMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	metricNames, err := h.storage.ListMetrics(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"metrics": metricNames,
		"count":   len(metricNames),
	})
}

// GetMetric returns details for a specific metric
func (h *Handlers) GetMetric(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	meta, err := h.storage.GetMetricMeta(ctx, name)
	if err != nil {
		writeError(w, http.StatusNotFound, "Metric not found")
		return
	}

	writeJSON(w, http.StatusOK, meta)
}

// QueryMetric queries metric data with aggregation
func (h *Handlers) QueryMetric(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	// Parse query parameters
	from, to, err := parseTimeRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	aggStr := r.URL.Query().Get("aggregation")
	if aggStr == "" {
		aggStr = "avg"
	}
	aggregation := storage.AggregationType(aggStr)

	result, err := h.storage.Query(ctx, name, from, to, aggregation)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// QueryMetricRange queries metric data with time steps
func (h *Handlers) QueryMetricRange(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	from, to, err := parseTimeRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	stepStr := r.URL.Query().Get("step")
	if stepStr == "" {
		stepStr = "1m"
	}
	step, err := time.ParseDuration(stepStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid step duration")
		return
	}

	aggStr := r.URL.Query().Get("aggregation")
	if aggStr == "" {
		aggStr = "avg"
	}
	aggregation := storage.AggregationType(aggStr)

	result, err := h.storage.QueryRange(ctx, name, from, to, step, aggregation)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ListAnomalies returns detected anomalies
func (h *Handlers) ListAnomalies(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	includeAck := r.URL.Query().Get("include_acknowledged") == "true"

	anomalies := h.anomaly.ListAnomalies(limit, includeAck)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"anomalies": anomalies,
		"count":     len(anomalies),
	})
}

// GetAnomaly returns a specific anomaly
func (h *Handlers) GetAnomaly(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	anom, ok := h.anomaly.GetAnomaly(id)
	if !ok {
		writeError(w, http.StatusNotFound, "Anomaly not found")
		return
	}

	writeJSON(w, http.StatusOK, anom)
}

// AcknowledgeAnomaly marks an anomaly as acknowledged
func (h *Handlers) AcknowledgeAnomaly(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		User string `json:"user"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.anomaly.AcknowledgeAnomaly(id, req.User); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}

// Dashboard handlers (simplified - would need dashboard storage)

type Dashboard struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Widgets     []Widget  `json:"widgets"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Widget struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // counter, line_chart, bar_chart, etc.
	Title    string                 `json:"title"`
	Metric   string                 `json:"metric"`
	Config   map[string]interface{} `json:"config"`
	Position WidgetPosition         `json:"position"`
}

type WidgetPosition struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// In-memory dashboard storage (would be replaced with proper storage)
var dashboards = make(map[string]*Dashboard)

func (h *Handlers) ListDashboards(w http.ResponseWriter, r *http.Request) {
	list := make([]*Dashboard, 0, len(dashboards))
	for _, d := range dashboards {
		list = append(list, d)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"dashboards": list,
		"count":      len(list),
	})
}

func (h *Handlers) CreateDashboard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	id := generateID()
	now := time.Now()
	dashboard := &Dashboard{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Widgets:     []Widget{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	dashboards[id] = dashboard
	writeJSON(w, http.StatusCreated, dashboard)
}

func (h *Handlers) GetDashboard(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dashboard, ok := dashboards[id]
	if !ok {
		writeError(w, http.StatusNotFound, "Dashboard not found")
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

func (h *Handlers) UpdateDashboard(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dashboard, ok := dashboards[id]
	if !ok {
		writeError(w, http.StatusNotFound, "Dashboard not found")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	dashboard.Name = req.Name
	dashboard.Description = req.Description
	dashboard.UpdatedAt = time.Now()

	writeJSON(w, http.StatusOK, dashboard)
}

func (h *Handlers) DeleteDashboard(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := dashboards[id]; !ok {
		writeError(w, http.StatusNotFound, "Dashboard not found")
		return
	}
	delete(dashboards, id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handlers) AddWidget(w http.ResponseWriter, r *http.Request) {
	dashboardID := chi.URLParam(r, "id")
	dashboard, ok := dashboards[dashboardID]
	if !ok {
		writeError(w, http.StatusNotFound, "Dashboard not found")
		return
	}

	var widget Widget
	if err := json.NewDecoder(r.Body).Decode(&widget); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	widget.ID = generateID()
	dashboard.Widgets = append(dashboard.Widgets, widget)
	dashboard.UpdatedAt = time.Now()

	writeJSON(w, http.StatusCreated, widget)
}

func (h *Handlers) UpdateWidget(w http.ResponseWriter, r *http.Request) {
	widgetID := chi.URLParam(r, "id")

	var update Widget
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Find and update widget
	for _, dashboard := range dashboards {
		for i, widget := range dashboard.Widgets {
			if widget.ID == widgetID {
				update.ID = widgetID
				dashboard.Widgets[i] = update
				dashboard.UpdatedAt = time.Now()
				writeJSON(w, http.StatusOK, update)
				return
			}
		}
	}

	writeError(w, http.StatusNotFound, "Widget not found")
}

func (h *Handlers) DeleteWidget(w http.ResponseWriter, r *http.Request) {
	widgetID := chi.URLParam(r, "id")

	for _, dashboard := range dashboards {
		for i, widget := range dashboard.Widgets {
			if widget.ID == widgetID {
				dashboard.Widgets = append(dashboard.Widgets[:i], dashboard.Widgets[i+1:]...)
				dashboard.UpdatedAt = time.Now()
				writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
				return
			}
		}
	}

	writeError(w, http.StatusNotFound, "Widget not found")
}

// GetStats returns overview statistics
func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	metricNames, _ := h.storage.ListMetrics(ctx)
	anomalies := h.anomaly.ListAnomalies(0, false)

	stats := map[string]interface{}{
		"metrics_count":        len(metricNames),
		"active_anomalies":     len(anomalies),
		"dashboards_count":     len(dashboards),
	}

	writeJSON(w, http.StatusOK, stats)
}

// IngestEvent handles single CDC event ingestion
func (h *Handlers) IngestEvent(w http.ResponseWriter, r *http.Request) {
	var event metrics.CDCEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid event format")
		return
	}

	h.metrics.ProcessEvent(&event)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

// IngestEventBatch handles batch CDC event ingestion
func (h *Handlers) IngestEventBatch(w http.ResponseWriter, r *http.Request) {
	var events []metrics.CDCEvent
	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid events format")
		return
	}

	for i := range events {
		h.metrics.ProcessEvent(&events[i])
	}

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":   "accepted",
		"received": len(events),
	})
}

// Helper functions

func parseTimeRange(r *http.Request) (from, to time.Time, err error) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	// Default: last hour
	to = time.Now()
	from = to.Add(-time.Hour)

	if fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			// Try Unix timestamp
			if ts, e := strconv.ParseInt(fromStr, 10, 64); e == nil {
				from = time.Unix(ts, 0)
				err = nil
			}
		}
	}

	if toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			if ts, e := strconv.ParseInt(toStr, 10, 64); e == nil {
				to = time.Unix(ts, 0)
				err = nil
			}
		}
	}

	return from, to, err
}

func generateID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// Quality Handlers

// ListQualityRules returns all quality rules
func (h *Handlers) ListQualityRules(w http.ResponseWriter, r *http.Request) {
	rules := h.quality.ListRules()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rules": rules,
		"count": len(rules),
	})
}

// CreateQualityRule creates a new quality rule
func (h *Handlers) CreateQualityRule(w http.ResponseWriter, r *http.Request) {
	var rule quality.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if rule.ID == "" {
		rule.ID = generateID()
	}
	rule.Enabled = true

	h.quality.AddRule(&rule)
	writeJSON(w, http.StatusCreated, rule)
}

// GetQualityRule returns a specific quality rule
func (h *Handlers) GetQualityRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rule, ok := h.quality.GetRule(id)
	if !ok {
		writeError(w, http.StatusNotFound, "Rule not found")
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

// DeleteQualityRule deletes a quality rule
func (h *Handlers) DeleteQualityRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.quality.DeleteRule(id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GetQualityScore returns the quality score for a table or overall
func (h *Handlers) GetQualityScore(w http.ResponseWriter, r *http.Request) {
	table := r.URL.Query().Get("table")

	if table != "" {
		score, ok := h.quality.GetScore(table)
		if !ok {
			writeError(w, http.StatusNotFound, "Table not found")
			return
		}
		writeJSON(w, http.StatusOK, score)
	} else {
		overallScore := h.quality.GetOverallScore()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"overall_score": overallScore,
		})
	}
}

// ListQualityViolations returns quality violations
func (h *Handlers) ListQualityViolations(w http.ResponseWriter, r *http.Request) {
	filter := quality.ViolationFilter{
		Table:    r.URL.Query().Get("table"),
		Field:    r.URL.Query().Get("field"),
		Severity: r.URL.Query().Get("severity"),
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = l
		}
	}

	violations := h.quality.GetViolations(filter)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"violations": violations,
		"count":      len(violations),
	})
}

// AcknowledgeQualityViolation acknowledges a violation
func (h *Handlers) AcknowledgeQualityViolation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		User string `json:"user"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.quality.AcknowledgeViolation(id, req.User); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}

// GetQualityReport generates a quality report
func (h *Handlers) GetQualityReport(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseTimeRange(r)
	if err != nil {
		from = time.Now().Add(-24 * time.Hour)
		to = time.Now()
	}

	report := h.quality.GetReport(from, to)
	writeJSON(w, http.StatusOK, report)
}

// ValidateRecord validates a record against quality rules
func (h *Handlers) ValidateRecord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Table  string                 `json:"table"`
		Record map[string]interface{} `json:"record"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result := h.quality.ValidateRecord(req.Table, req.Record)
	writeJSON(w, http.StatusOK, result)
}

// GetQualityStats returns quality monitoring statistics
func (h *Handlers) GetQualityStats(w http.ResponseWriter, r *http.Request) {
	stats := h.quality.GetStats()
	writeJSON(w, http.StatusOK, stats)
}

// Schema Handlers

// ListSchemas returns all tracked schemas
func (h *Handlers) ListSchemas(w http.ResponseWriter, r *http.Request) {
	schemas := h.schema.ListSchemas()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"schemas": schemas,
		"count":   len(schemas),
	})
}

// GetSchema returns a specific table schema
func (h *Handlers) GetSchema(w http.ResponseWriter, r *http.Request) {
	database := chi.URLParam(r, "database")
	schemaName := chi.URLParam(r, "schema")
	table := chi.URLParam(r, "table")

	s, ok := h.schema.GetSchema(database, schemaName, table)
	if !ok {
		writeError(w, http.StatusNotFound, "Schema not found")
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// RegisterSchema registers a table schema
func (h *Handlers) RegisterSchema(w http.ResponseWriter, r *http.Request) {
	var s schema.TableSchema
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	h.schema.RegisterSchema(&s)
	writeJSON(w, http.StatusCreated, s)
}

// ListSchemaChanges returns schema changes
func (h *Handlers) ListSchemaChanges(w http.ResponseWriter, r *http.Request) {
	filter := schema.ChangeFilter{
		Table: r.URL.Query().Get("table"),
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = l
		}
	}

	if breaking := r.URL.Query().Get("breaking"); breaking == "true" {
		b := true
		filter.Breaking = &b
	}

	changes := h.schema.GetChanges(filter)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"changes": changes,
		"count":   len(changes),
	})
}

// GetSchemaChange returns a specific schema change
func (h *Handlers) GetSchemaChange(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	change, ok := h.schema.GetChange(id)
	if !ok {
		writeError(w, http.StatusNotFound, "Change not found")
		return
	}
	writeJSON(w, http.StatusOK, change)
}

// ProcessDDLEvent processes a DDL event
func (h *Handlers) ProcessDDLEvent(w http.ResponseWriter, r *http.Request) {
	var ddl schema.DDLEvent
	if err := json.NewDecoder(r.Body).Decode(&ddl); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	change, err := h.schema.ProcessDDLEvent(ddl)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if change != nil {
		writeJSON(w, http.StatusOK, change)
	} else {
		writeJSON(w, http.StatusOK, map[string]string{"status": "processed"})
	}
}

// CreateSchemaSnapshot creates a schema snapshot
func (h *Handlers) CreateSchemaSnapshot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	snapshot := h.schema.CreateSnapshot(req.Description)
	writeJSON(w, http.StatusCreated, snapshot)
}

// ListSchemaSnapshots returns all snapshots
func (h *Handlers) ListSchemaSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots := h.schema.ListSnapshots()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"snapshots": snapshots,
		"count":     len(snapshots),
	})
}

// GetSchemaSnapshot returns a specific snapshot
func (h *Handlers) GetSchemaSnapshot(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	snapshot, ok := h.schema.GetSnapshot(id)
	if !ok {
		writeError(w, http.StatusNotFound, "Snapshot not found")
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

// GetSchemaStats returns schema tracking statistics
func (h *Handlers) GetSchemaStats(w http.ResponseWriter, r *http.Request) {
	stats := h.schema.GetStats()
	writeJSON(w, http.StatusOK, stats)
}

// Alert Handlers

// ListAlertRules returns all alert rules
func (h *Handlers) ListAlertRules(w http.ResponseWriter, r *http.Request) {
	rules := h.alerts.ListRules()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rules": rules,
		"count": len(rules),
	})
}

// CreateAlertRule creates a new alert rule
func (h *Handlers) CreateAlertRule(w http.ResponseWriter, r *http.Request) {
	var rule alerts.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if rule.ID == "" {
		rule.ID = generateID()
	}
	rule.Enabled = true

	h.alerts.AddRule(&rule)
	writeJSON(w, http.StatusCreated, rule)
}

// GetAlertRule returns a specific alert rule
func (h *Handlers) GetAlertRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rule, ok := h.alerts.GetRule(id)
	if !ok {
		writeError(w, http.StatusNotFound, "Rule not found")
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

// UpdateAlertRule updates an alert rule
func (h *Handlers) UpdateAlertRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var rule alerts.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	rule.ID = id
	h.alerts.UpdateRule(&rule)
	writeJSON(w, http.StatusOK, rule)
}

// DeleteAlertRule deletes an alert rule
func (h *Handlers) DeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.alerts.DeleteRule(id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ListAlerts returns alerts
func (h *Handlers) ListAlerts(w http.ResponseWriter, r *http.Request) {
	filter := alerts.AlertFilter{}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = alerts.Status(status)
	}
	if severity := r.URL.Query().Get("severity"); severity != "" {
		filter.Severity = alerts.Severity(severity)
	}
	if alertType := r.URL.Query().Get("type"); alertType != "" {
		filter.Type = alerts.AlertType(alertType)
	}

	alertsList := h.alerts.GetAlerts(filter)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"alerts": alertsList,
		"count":  len(alertsList),
	})
}

// GetAlert returns a specific alert
func (h *Handlers) GetAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	alert, ok := h.alerts.GetAlert(id)
	if !ok {
		writeError(w, http.StatusNotFound, "Alert not found")
		return
	}
	writeJSON(w, http.StatusOK, alert)
}

// AcknowledgeAlert acknowledges an alert
func (h *Handlers) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		User string `json:"user"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.alerts.AcknowledgeAlert(id, req.User); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}

// ResolveAlert resolves an alert
func (h *Handlers) ResolveAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		User string `json:"user"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.alerts.ResolveAlert(id, req.User); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}

// SnoozeAlert snoozes an alert
func (h *Handlers) SnoozeAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Duration string `json:"duration"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	duration, err := time.ParseDuration(req.Duration)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid duration")
		return
	}

	if err := h.alerts.SnoozeAlert(id, duration); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "snoozed"})
}

// ListAlertChannels returns all notification channels
func (h *Handlers) ListAlertChannels(w http.ResponseWriter, r *http.Request) {
	channels := h.alerts.ListChannels()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"channels": channels,
		"count":    len(channels),
	})
}

// CreateAlertChannel creates a notification channel
func (h *Handlers) CreateAlertChannel(w http.ResponseWriter, r *http.Request) {
	var channel alerts.Channel
	if err := json.NewDecoder(r.Body).Decode(&channel); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if channel.ID == "" {
		channel.ID = generateID()
	}
	channel.Enabled = true

	h.alerts.AddChannel(&channel)
	writeJSON(w, http.StatusCreated, channel)
}

// DeleteAlertChannel deletes a notification channel
func (h *Handlers) DeleteAlertChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.alerts.DeleteChannel(id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GetAlertSummary returns an alert summary
func (h *Handlers) GetAlertSummary(w http.ResponseWriter, r *http.Request) {
	summary := h.alerts.GetSummary()
	writeJSON(w, http.StatusOK, summary)
}

// FireTestAlert fires a test alert
func (h *Handlers) FireTestAlert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title    string         `json:"title"`
		Message  string         `json:"message"`
		Severity alerts.Severity `json:"severity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	alert := &alerts.Alert{
		Type:     alerts.AlertTypeThreshold,
		Severity: req.Severity,
		Title:    req.Title,
		Message:  req.Message,
	}

	h.alerts.FireManualAlert(alert)
	writeJSON(w, http.StatusOK, alert)
}
