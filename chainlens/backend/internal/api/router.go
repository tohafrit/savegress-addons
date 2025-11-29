package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"getchainlens.com/chainlens/backend/internal/config"
	"getchainlens.com/chainlens/backend/internal/database"
)

func NewRouter(cfg *config.Config, db *database.DB) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://getchainlens.com", "https://*.getchainlens.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"chainlens-api","version":"0.1.0"}`))
	})

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Group(func(r chi.Router) {
			// Auth
			r.Post("/auth/register", handleRegister(cfg, db))
			r.Post("/auth/login", handleLogin(cfg, db))
			r.Post("/auth/refresh", handleRefreshToken(cfg, db))

			// License validation (for VS Code extension)
			r.Post("/license/validate", handleValidateLicense(cfg, db))
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(cfg))

			// User
			r.Get("/user", handleGetUser(db))
			r.Put("/user", handleUpdateUser(db))
			r.Put("/user/password", handleChangePassword(db))

			// Analysis
			r.Post("/analyze", handleAnalyzeContract(cfg, db))

			// Transaction tracing
			r.Get("/trace/{txHash}", handleTraceTransaction(cfg, db))

			// Simulation
			r.Post("/simulate", handleSimulateTransaction(cfg, db))

			// Forks
			r.Post("/forks", handleCreateFork(cfg, db))
			r.Delete("/forks/{forkId}", handleDeleteFork(cfg, db))

			// Monitors
			r.Get("/monitors", handleGetMonitors(db))
			r.Post("/monitors", handleCreateMonitor(cfg, db))
			r.Put("/monitors/{id}", handleUpdateMonitor(db))
			r.Delete("/monitors/{id}", handleDeleteMonitor(db))

			// Projects
			r.Get("/projects", handleGetProjects(db))
			r.Post("/projects", handleCreateProject(db))
			r.Get("/projects/{id}", handleGetProject(db))
			r.Put("/projects/{id}", handleUpdateProject(db))
			r.Delete("/projects/{id}", handleDeleteProject(db))

			// Contracts
			r.Get("/contracts", handleGetContracts(db))
			r.Post("/contracts", handleAddContract(cfg, db))
			r.Delete("/contracts/{id}", handleDeleteContract(db))

			// Billing
			r.Get("/billing/subscription", handleGetSubscription(db))
			r.Post("/billing/checkout", handleCreateCheckout(cfg, db))
			r.Post("/billing/portal", handleCreatePortal(cfg, db))

			// API usage
			r.Get("/usage", handleGetUsage(db))
		})

		// Stripe webhook (no auth)
		r.Post("/webhooks/stripe", handleStripeWebhook(cfg, db))
	})

	return r
}
