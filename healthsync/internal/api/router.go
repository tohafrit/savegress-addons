package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/savegress/healthsync/internal/anonymization"
	"github.com/savegress/healthsync/internal/audit"
	"github.com/savegress/healthsync/internal/compliance"
	"github.com/savegress/healthsync/internal/config"
	"github.com/savegress/healthsync/internal/consent"
)

// Server represents the API server
type Server struct {
	config   *config.Config
	router   chi.Router
	handlers *Handlers
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, comp *compliance.Engine, auditLog *audit.Logger, anon *anonymization.Engine, consent *consent.Manager) *Server {
	s := &Server{
		config:   cfg,
		router:   chi.NewRouter(),
		handlers: NewHandlers(comp, auditLog, anon, consent),
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

	s.router.Route("/api/v1/healthsync", func(r chi.Router) {
		// FHIR Resources
		r.Route("/fhir", func(r chi.Router) {
			// Patient
			r.Route("/Patient", func(r chi.Router) {
				r.Get("/", s.handlers.SearchPatients)
				r.Post("/", s.handlers.CreatePatient)
				r.Get("/{id}", s.handlers.GetPatient)
				r.Put("/{id}", s.handlers.UpdatePatient)
				r.Delete("/{id}", s.handlers.DeletePatient)
			})

			// Observation
			r.Route("/Observation", func(r chi.Router) {
				r.Get("/", s.handlers.SearchObservations)
				r.Post("/", s.handlers.CreateObservation)
				r.Get("/{id}", s.handlers.GetObservation)
			})

			// Encounter
			r.Route("/Encounter", func(r chi.Router) {
				r.Get("/", s.handlers.SearchEncounters)
				r.Post("/", s.handlers.CreateEncounter)
				r.Get("/{id}", s.handlers.GetEncounter)
			})
		})

		// Compliance
		r.Route("/compliance", func(r chi.Router) {
			r.Get("/violations", s.handlers.ListViolations)
			r.Get("/violations/{id}", s.handlers.GetViolation)
			r.Post("/violations/{id}/resolve", s.handlers.ResolveViolation)
			r.Post("/validate", s.handlers.ValidateResource)
			r.Post("/phi-scan", s.handlers.ScanForPHI)
			r.Post("/minimum-necessary", s.handlers.CheckMinimumNecessary)
			r.Get("/stats", s.handlers.GetComplianceStats)
		})

		// Audit
		r.Route("/audit", func(r chi.Router) {
			r.Get("/events", s.handlers.ListAuditEvents)
			r.Get("/events/{id}", s.handlers.GetAuditEvent)
			r.Get("/stats", s.handlers.GetAuditStats)
		})

		// Consent
		r.Route("/consent", func(r chi.Router) {
			r.Get("/", s.handlers.ListConsents)
			r.Post("/", s.handlers.CreateConsent)
			r.Get("/{id}", s.handlers.GetConsent)
			r.Post("/{id}/revoke", s.handlers.RevokeConsent)
			r.Post("/check", s.handlers.CheckAccess)
			r.Get("/stats", s.handlers.GetConsentStats)
		})

		// Access Requests
		r.Route("/access-requests", func(r chi.Router) {
			r.Get("/", s.handlers.ListAccessRequests)
			r.Post("/", s.handlers.CreateAccessRequest)
			r.Get("/{id}", s.handlers.GetAccessRequest)
			r.Post("/{id}/approve", s.handlers.ApproveAccessRequest)
			r.Post("/{id}/deny", s.handlers.DenyAccessRequest)
		})

		// Anonymization
		r.Route("/anonymize", func(r chi.Router) {
			r.Post("/patient", s.handlers.AnonymizePatient)
			r.Post("/resource", s.handlers.AnonymizeResource)
			r.Post("/text", s.handlers.RedactText)
			r.Post("/k-anonymity", s.handlers.CheckKAnonymity)
		})

		// Stats
		r.Get("/stats", s.handlers.GetOverallStats)
	})
}

// Router returns the chi router
func (s *Server) Router() http.Handler {
	return s.router
}
