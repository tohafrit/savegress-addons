package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/savegress/datawatch/internal/anomaly"
	"github.com/savegress/datawatch/internal/config"
	"github.com/savegress/datawatch/internal/metrics"
	"github.com/savegress/datawatch/internal/storage"
)

// Server represents the API server
type Server struct {
	config   *config.Config
	router   chi.Router
	handlers *Handlers
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, metricsEngine *metrics.Engine, anomalyDetector *anomaly.Detector, store storage.MetricStorage) *Server {
	s := &Server{
		config: cfg,
		router: chi.NewRouter(),
		handlers: &Handlers{
			metrics:  metricsEngine,
			anomaly:  anomalyDetector,
			storage:  store,
		},
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
	})
}

// Router returns the chi router
func (s *Server) Router() http.Handler {
	return s.router
}
