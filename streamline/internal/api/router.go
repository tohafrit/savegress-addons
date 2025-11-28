package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/savegress/streamline/internal/config"
	"github.com/savegress/streamline/internal/inventory"
	"github.com/savegress/streamline/internal/orders"
	"github.com/savegress/streamline/internal/pricing"
)

// Server represents the API server
type Server struct {
	config   *config.Config
	router   chi.Router
	handlers *Handlers
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, inv *inventory.Engine, ord *orders.Processor, pr *pricing.Engine) *Server {
	s := &Server{
		config:   cfg,
		router:   chi.NewRouter(),
		handlers: NewHandlers(inv, ord, pr),
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

	s.router.Route("/api/v1/streamline", func(r chi.Router) {
		// Inventory
		r.Route("/inventory", func(r chi.Router) {
			r.Get("/", s.handlers.GetAllInventory)
			r.Get("/low-stock", s.handlers.GetLowStock)
			r.Get("/{sku}", s.handlers.GetInventory)
			r.Put("/{sku}", s.handlers.UpdateInventory)
			r.Post("/{sku}/adjust", s.handlers.AdjustInventory)
			r.Post("/{sku}/transfer", s.handlers.TransferInventory)
		})

		// Orders
		r.Route("/orders", func(r chi.Router) {
			r.Get("/", s.handlers.ListOrders)
			r.Post("/", s.handlers.CreateOrder)
			r.Get("/stats", s.handlers.GetOrderStats)
			r.Get("/{id}", s.handlers.GetOrder)
			r.Post("/{id}/route", s.handlers.RouteOrder)
			r.Post("/{id}/assign", s.handlers.AssignWarehouse)
			r.Post("/{id}/fulfill", s.handlers.FulfillOrder)
			r.Post("/{id}/cancel", s.handlers.CancelOrder)
		})

		// Pricing
		r.Route("/pricing", func(r chi.Router) {
			r.Get("/rules", s.handlers.GetPricingRules)
			r.Post("/rules", s.handlers.CreatePricingRule)
			r.Delete("/rules/{id}", s.handlers.DeletePricingRule)
		})

		// Stats
		r.Get("/stats", s.handlers.GetStats)
	})
}

// Router returns the chi router
func (s *Server) Router() http.Handler {
	return s.router
}
