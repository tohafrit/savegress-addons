package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/savegress/iotsense/internal/alerts"
	"github.com/savegress/iotsense/internal/devices"
	"github.com/savegress/iotsense/pkg/models"
)

// Health check
func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"service": "iotsense",
		"time":    time.Now().UTC(),
	})
}

// Device handlers

func (s *Server) listDevices(w http.ResponseWriter, r *http.Request) {
	filter := devices.DeviceFilter{
		Type:   models.DeviceType(r.URL.Query().Get("type")),
		Status: models.DeviceStatus(r.URL.Query().Get("status")),
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}

	devices := s.registry.List(filter)
	respondJSON(w, http.StatusOK, devices)
}

func (s *Server) registerDevice(w http.ResponseWriter, r *http.Request) {
	var device models.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.registry.Register(&device); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, device)
}

func (s *Server) getDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	device, ok := s.registry.Get(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Device not found")
		return
	}
	respondJSON(w, http.StatusOK, device)
}

func (s *Server) updateDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var device models.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	device.ID = id
	if err := s.registry.Update(&device); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, device)
}

func (s *Server) unregisterDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.registry.Unregister(id); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deviceHeartbeat(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.registry.Heartbeat(id); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) getDeviceMetrics(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	metrics := s.telemetry.GetDeviceMetrics(id)
	respondJSON(w, http.StatusOK, metrics)
}

func (s *Server) getDeviceEvents(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	events := s.registry.GetDeviceEvents(id, limit)
	respondJSON(w, http.StatusOK, events)
}

func (s *Server) getDeviceCommands(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	commands := s.registry.GetPendingCommands(id)
	respondJSON(w, http.StatusOK, commands)
}

// Group handlers

func (s *Server) listGroups(w http.ResponseWriter, r *http.Request) {
	groups := s.registry.ListGroups()
	respondJSON(w, http.StatusOK, groups)
}

func (s *Server) createGroup(w http.ResponseWriter, r *http.Request) {
	var group models.DeviceGroup
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.registry.CreateGroup(&group); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, group)
}

func (s *Server) getGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	group, ok := s.registry.GetGroup(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Group not found")
		return
	}
	respondJSON(w, http.StatusOK, group)
}

func (s *Server) getGroupDevices(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	devices := s.registry.GetGroupDevices(id)
	respondJSON(w, http.StatusOK, devices)
}

func (s *Server) addToGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "id")
	deviceID := chi.URLParam(r, "deviceId")

	if err := s.registry.AddToGroup(groupID, deviceID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "added"})
}

func (s *Server) removeFromGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "id")
	deviceID := chi.URLParam(r, "deviceId")

	if err := s.registry.RemoveFromGroup(groupID, deviceID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Telemetry handlers

func (s *Server) ingestTelemetry(w http.ResponseWriter, r *http.Request) {
	var point models.TelemetryPoint
	if err := json.NewDecoder(r.Body).Decode(&point); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.telemetry.Ingest(&point); err != nil {
		respondError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	// Update device telemetry time
	s.registry.UpdateTelemetryTime(point.DeviceID)

	// Process for alerts
	s.alerts.ProcessTelemetry(&point)

	respondJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *Server) ingestBatch(w http.ResponseWriter, r *http.Request) {
	var batch models.TelemetryBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.telemetry.IngestBatch(&batch); err != nil {
		respondError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	// Update device telemetry time
	s.registry.UpdateTelemetryTime(batch.DeviceID)

	// Process for alerts
	for i := range batch.Points {
		s.alerts.ProcessTelemetry(&batch.Points[i])
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":         "accepted",
		"points_ingested": len(batch.Points),
	})
}

func (s *Server) queryTelemetry(w http.ResponseWriter, r *http.Request) {
	query := models.TimeSeriesQuery{
		DeviceID: r.URL.Query().Get("device_id"),
		Metric:   r.URL.Query().Get("metric"),
	}

	if start := r.URL.Query().Get("start"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			query.StartTime = t
		}
	} else {
		query.StartTime = time.Now().Add(-1 * time.Hour)
	}

	if end := r.URL.Query().Get("end"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			query.EndTime = t
		}
	} else {
		query.EndTime = time.Now()
	}

	results, err := s.telemetry.Query(&query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, results)
}

func (s *Server) getLatestValue(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceId")
	metric := chi.URLParam(r, "metric")

	point, ok := s.telemetry.GetLatest(deviceID, metric)
	if !ok {
		respondError(w, http.StatusNotFound, "No data found")
		return
	}

	respondJSON(w, http.StatusOK, point)
}

func (s *Server) getAggregated(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceId")
	metric := chi.URLParam(r, "metric")

	var start, end time.Time
	if s := r.URL.Query().Get("start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			start = t
		}
	} else {
		start = time.Now().Add(-24 * time.Hour)
	}

	if e := r.URL.Query().Get("end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			end = t
		}
	} else {
		end = time.Now()
	}

	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "1h"
	}

	results, err := s.telemetry.GetAggregated(deviceID, metric, start, end, interval)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, results)
}

// Command handlers

func (s *Server) sendCommand(w http.ResponseWriter, r *http.Request) {
	var cmd models.Command
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.registry.SendCommand(&cmd); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, cmd)
}

func (s *Server) getCommand(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cmd, ok := s.registry.GetCommand(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Command not found")
		return
	}
	respondJSON(w, http.StatusOK, cmd)
}

func (s *Server) updateCommandStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Status models.CommandStatus `json:"status"`
		Result interface{}          `json:"result,omitempty"`
		Error  string               `json:"error,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.registry.UpdateCommandStatus(id, req.Status, req.Result, req.Error); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// Alert rule handlers

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	rules := s.alerts.ListRules()
	respondJSON(w, http.StatusOK, rules)
}

func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
	var rule models.AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.alerts.CreateRule(&rule); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, rule)
}

func (s *Server) getRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rule, ok := s.alerts.GetRule(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Rule not found")
		return
	}
	respondJSON(w, http.StatusOK, rule)
}

func (s *Server) updateRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var rule models.AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	rule.ID = id
	if err := s.alerts.UpdateRule(&rule); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, rule)
}

func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.alerts.DeleteRule(id); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Alert handlers

func (s *Server) listAlerts(w http.ResponseWriter, r *http.Request) {
	filter := alerts.AlertFilter{
		DeviceID: r.URL.Query().Get("device_id"),
		RuleID:   r.URL.Query().Get("rule_id"),
		Severity: models.AlertSeverity(r.URL.Query().Get("severity")),
		Status:   models.AlertStatus(r.URL.Query().Get("status")),
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}

	alertList := s.alerts.ListAlerts(filter)
	respondJSON(w, http.StatusOK, alertList)
}

func (s *Server) getAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	alert, ok := s.alerts.GetAlert(id)
	if !ok {
		respondError(w, http.StatusNotFound, "Alert not found")
		return
	}
	respondJSON(w, http.StatusOK, alert)
}

func (s *Server) acknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		User string `json:"user"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.alerts.AcknowledgeAlert(id, req.User); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}

func (s *Server) resolveAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.alerts.ResolveAlert(id); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}

// Stats handlers

func (s *Server) getSystemStats(w http.ResponseWriter, r *http.Request) {
	deviceStats := s.registry.GetStats()
	respondJSON(w, http.StatusOK, deviceStats)
}

func (s *Server) getTelemetryStats(w http.ResponseWriter, r *http.Request) {
	stats := s.telemetry.GetStats()
	respondJSON(w, http.StatusOK, stats)
}

func (s *Server) getAlertStats(w http.ResponseWriter, r *http.Request) {
	stats := s.alerts.GetStats()
	respondJSON(w, http.StatusOK, stats)
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"error":   http.StatusText(status),
		"message": message,
	})
}
