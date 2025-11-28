package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/savegress/datawatch/internal/anomaly"
	"github.com/savegress/datawatch/internal/metrics"
	"github.com/savegress/datawatch/internal/storage"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	metrics  *metrics.Engine
	anomaly  *anomaly.Detector
	storage  storage.MetricStorage
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
