package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/savegress/iotsense/internal/alerts"
	"github.com/savegress/iotsense/internal/devices"
	"github.com/savegress/iotsense/internal/telemetry"
)

// Server represents the API server
type Server struct {
	router    chi.Router
	registry  *devices.Registry
	telemetry *telemetry.Engine
	alerts    *alerts.Engine
}

// NewServer creates a new API server
func NewServer(registry *devices.Registry, telemetryEngine *telemetry.Engine, alertsEngine *alerts.Engine) *Server {
	s := &Server{
		router:    chi.NewRouter(),
		registry:  registry,
		telemetry: telemetryEngine,
		alerts:    alertsEngine,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)

	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	s.router.Get("/health", s.healthCheck)

	// API v1
	s.router.Route("/api/v1", func(r chi.Router) {
		// Devices
		r.Route("/devices", func(r chi.Router) {
			r.Get("/", s.listDevices)
			r.Post("/", s.registerDevice)
			r.Get("/{id}", s.getDevice)
			r.Put("/{id}", s.updateDevice)
			r.Delete("/{id}", s.unregisterDevice)
			r.Post("/{id}/heartbeat", s.deviceHeartbeat)
			r.Get("/{id}/metrics", s.getDeviceMetrics)
			r.Get("/{id}/events", s.getDeviceEvents)
			r.Get("/{id}/commands", s.getDeviceCommands)
		})

		// Device Groups
		r.Route("/groups", func(r chi.Router) {
			r.Get("/", s.listGroups)
			r.Post("/", s.createGroup)
			r.Get("/{id}", s.getGroup)
			r.Get("/{id}/devices", s.getGroupDevices)
			r.Post("/{id}/devices/{deviceId}", s.addToGroup)
			r.Delete("/{id}/devices/{deviceId}", s.removeFromGroup)
		})

		// Telemetry
		r.Route("/telemetry", func(r chi.Router) {
			r.Post("/", s.ingestTelemetry)
			r.Post("/batch", s.ingestBatch)
			r.Get("/query", s.queryTelemetry)
			r.Get("/latest/{deviceId}/{metric}", s.getLatestValue)
			r.Get("/aggregated/{deviceId}/{metric}", s.getAggregated)
		})

		// Commands
		r.Route("/commands", func(r chi.Router) {
			r.Post("/", s.sendCommand)
			r.Get("/{id}", s.getCommand)
			r.Put("/{id}/status", s.updateCommandStatus)
		})

		// Alert Rules
		r.Route("/rules", func(r chi.Router) {
			r.Get("/", s.listRules)
			r.Post("/", s.createRule)
			r.Get("/{id}", s.getRule)
			r.Put("/{id}", s.updateRule)
			r.Delete("/{id}", s.deleteRule)
		})

		// Alerts
		r.Route("/alerts", func(r chi.Router) {
			r.Get("/", s.listAlerts)
			r.Get("/{id}", s.getAlert)
			r.Post("/{id}/acknowledge", s.acknowledgeAlert)
			r.Post("/{id}/resolve", s.resolveAlert)
		})

		// Stats
		r.Get("/stats", s.getSystemStats)
		r.Get("/stats/telemetry", s.getTelemetryStats)
		r.Get("/stats/alerts", s.getAlertStats)
	})
}

// Handler returns the HTTP handler
func (s *Server) Handler() http.Handler {
	return s.router
}
