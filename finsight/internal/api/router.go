package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/savegress/finsight/internal/config"
	"github.com/savegress/finsight/internal/fraud"
	"github.com/savegress/finsight/internal/reconciliation"
	"github.com/savegress/finsight/internal/reporting"
	"github.com/savegress/finsight/internal/transactions"
)

// Server represents the API server
type Server struct {
	config   *config.Config
	router   chi.Router
	handlers *Handlers
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, txn *transactions.Engine, fraud *fraud.Detector, recon *reconciliation.Engine, report *reporting.Generator) *Server {
	s := &Server{
		config:   cfg,
		router:   chi.NewRouter(),
		handlers: NewHandlers(txn, fraud, recon, report),
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

	s.router.Route("/api/v1/finsight", func(r chi.Router) {
		// Transactions
		r.Route("/transactions", func(r chi.Router) {
			r.Get("/", s.handlers.ListTransactions)
			r.Post("/", s.handlers.CreateTransaction)
			r.Get("/stats", s.handlers.GetTransactionStats)
			r.Get("/{id}", s.handlers.GetTransaction)
			r.Put("/{id}", s.handlers.UpdateTransaction)
		})

		// Accounts
		r.Route("/accounts", func(r chi.Router) {
			r.Get("/", s.handlers.ListAccounts)
			r.Post("/", s.handlers.CreateAccount)
			r.Get("/{id}", s.handlers.GetAccount)
			r.Get("/{id}/transactions", s.handlers.GetAccountTransactions)
			r.Get("/{id}/balance", s.handlers.GetAccountBalance)
		})

		// Fraud Detection
		r.Route("/fraud", func(r chi.Router) {
			r.Get("/alerts", s.handlers.ListFraudAlerts)
			r.Get("/alerts/{id}", s.handlers.GetFraudAlert)
			r.Post("/alerts/{id}/resolve", s.handlers.ResolveFraudAlert)
			r.Post("/evaluate", s.handlers.EvaluateTransaction)
			r.Get("/stats", s.handlers.GetFraudStats)
		})

		// Reconciliation
		r.Route("/reconciliation", func(r chi.Router) {
			r.Get("/batches", s.handlers.ListReconcileBatches)
			r.Post("/batches", s.handlers.CreateReconcileBatch)
			r.Get("/batches/{id}", s.handlers.GetReconcileBatch)
			r.Post("/batches/{id}/run", s.handlers.RunReconciliation)
			r.Get("/batches/{id}/exceptions", s.handlers.GetBatchExceptions)
			r.Post("/exceptions/{id}/resolve", s.handlers.ResolveException)
			r.Get("/stats", s.handlers.GetReconcileStats)
		})

		// Reports
		r.Route("/reports", func(r chi.Router) {
			r.Get("/", s.handlers.ListReports)
			r.Post("/", s.handlers.CreateReport)
			r.Get("/{id}", s.handlers.GetReport)
			r.Post("/{id}/generate", s.handlers.GenerateReport)
			r.Delete("/{id}", s.handlers.DeleteReport)
		})

		// Stats
		r.Get("/stats", s.handlers.GetOverallStats)
	})
}

// Router returns the chi router
func (s *Server) Router() http.Handler {
	return s.router
}
