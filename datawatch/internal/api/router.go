package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/savegress/datawatch/internal/alerts"
	"github.com/savegress/datawatch/internal/anomaly"
	"github.com/savegress/datawatch/internal/config"
	"github.com/savegress/datawatch/internal/metrics"
	"github.com/savegress/datawatch/internal/quality"
	"github.com/savegress/datawatch/internal/schema"
	"github.com/savegress/datawatch/internal/storage"
)

// Server represents the API server
type Server struct {
	config   *config.Config
	router   chi.Router
	handlers *Handlers
}

// NewServer creates a new API server
func NewServer(
	cfg *config.Config,
	metricsEngine *metrics.Engine,
	anomalyDetector *anomaly.Detector,
	store storage.MetricStorage,
	qualityMonitor *quality.Monitor,
	schemaTracker *schema.Tracker,
	alertsEngine *alerts.Engine,
) *Server {
	s := &Server{
		config: cfg,
		router: chi.NewRouter(),
		handlers: NewHandlers(metricsEngine, anomalyDetector, store, qualityMonitor, schemaTracker, alertsEngine),
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Compress(5))

	// CORS
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
}

func (s *Server) setupRoutes() {
	s.router.Get("/health", s.handlers.HealthCheck)

	s.router.Route("/api/v1/datawatch", func(r chi.Router) {
		// Metrics endpoints
		r.Route("/metrics", func(r chi.Router) {
			r.Get("/", s.handlers.ListMetrics)
			r.Get("/{name}", s.handlers.GetMetric)
			r.Get("/{name}/query", s.handlers.QueryMetric)
			r.Get("/{name}/range", s.handlers.QueryMetricRange)
		})

		// Anomalies endpoints
		r.Route("/anomalies", func(r chi.Router) {
			r.Get("/", s.handlers.ListAnomalies)
			r.Get("/{id}", s.handlers.GetAnomaly)
			r.Post("/{id}/acknowledge", s.handlers.AcknowledgeAnomaly)
		})

		// Dashboards endpoints
		r.Route("/dashboards", func(r chi.Router) {
			r.Get("/", s.handlers.ListDashboards)
			r.Post("/", s.handlers.CreateDashboard)
			r.Get("/{id}", s.handlers.GetDashboard)
			r.Put("/{id}", s.handlers.UpdateDashboard)
			r.Delete("/{id}", s.handlers.DeleteDashboard)

			// Widgets
			r.Post("/{id}/widgets", s.handlers.AddWidget)
		})

		r.Route("/widgets", func(r chi.Router) {
			r.Put("/{id}", s.handlers.UpdateWidget)
			r.Delete("/{id}", s.handlers.DeleteWidget)
		})

		// Overview/stats
		r.Get("/stats", s.handlers.GetStats)

		// CDC event ingestion (for internal use)
		r.Post("/events", s.handlers.IngestEvent)
		r.Post("/events/batch", s.handlers.IngestEventBatch)

		// Quality endpoints
		r.Route("/quality", func(r chi.Router) {
			r.Get("/rules", s.handlers.ListQualityRules)
			r.Post("/rules", s.handlers.CreateQualityRule)
			r.Get("/rules/{id}", s.handlers.GetQualityRule)
			r.Delete("/rules/{id}", s.handlers.DeleteQualityRule)

			r.Get("/score", s.handlers.GetQualityScore)
			r.Get("/stats", s.handlers.GetQualityStats)
			r.Get("/report", s.handlers.GetQualityReport)
			r.Post("/validate", s.handlers.ValidateRecord)

			r.Get("/violations", s.handlers.ListQualityViolations)
			r.Post("/violations/{id}/acknowledge", s.handlers.AcknowledgeQualityViolation)
		})

		// Schema endpoints
		r.Route("/schema", func(r chi.Router) {
			r.Get("/tables", s.handlers.ListSchemas)
			r.Post("/tables", s.handlers.RegisterSchema)
			r.Get("/tables/{database}/{schema}/{table}", s.handlers.GetSchema)

			r.Get("/changes", s.handlers.ListSchemaChanges)
			r.Get("/changes/{id}", s.handlers.GetSchemaChange)
			r.Post("/ddl", s.handlers.ProcessDDLEvent)

			r.Get("/snapshots", s.handlers.ListSchemaSnapshots)
			r.Post("/snapshots", s.handlers.CreateSchemaSnapshot)
			r.Get("/snapshots/{id}", s.handlers.GetSchemaSnapshot)

			r.Get("/stats", s.handlers.GetSchemaStats)
		})

		// Alerts endpoints
		r.Route("/alerts", func(r chi.Router) {
			r.Get("/rules", s.handlers.ListAlertRules)
			r.Post("/rules", s.handlers.CreateAlertRule)
			r.Get("/rules/{id}", s.handlers.GetAlertRule)
			r.Put("/rules/{id}", s.handlers.UpdateAlertRule)
			r.Delete("/rules/{id}", s.handlers.DeleteAlertRule)

			r.Get("/", s.handlers.ListAlerts)
			r.Get("/{id}", s.handlers.GetAlert)
			r.Post("/{id}/acknowledge", s.handlers.AcknowledgeAlert)
			r.Post("/{id}/resolve", s.handlers.ResolveAlert)
			r.Post("/{id}/snooze", s.handlers.SnoozeAlert)

			r.Get("/channels", s.handlers.ListAlertChannels)
			r.Post("/channels", s.handlers.CreateAlertChannel)
			r.Delete("/channels/{id}", s.handlers.DeleteAlertChannel)

			r.Get("/summary", s.handlers.GetAlertSummary)
			r.Post("/test", s.handlers.FireTestAlert)
		})
	})
}

// Router returns the chi router
func (s *Server) Router() http.Handler {
	return s.router
}
