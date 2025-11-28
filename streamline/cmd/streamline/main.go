package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/savegress/streamline/internal/api"
	"github.com/savegress/streamline/internal/config"
	"github.com/savegress/streamline/internal/inventory"
	"github.com/savegress/streamline/internal/orders"
	"github.com/savegress/streamline/internal/pricing"
	"github.com/savegress/streamline/pkg/models"
)

func main() {
	log.Println("Starting StreamLine...")

	// Load configuration
	cfg := loadConfig()

	// Initialize inventory engine
	invEngine := inventory.NewEngine()

	// Initialize order router and processor
	orderRouter := orders.NewRouter()
	orderRouter.SetOptimization(orders.OptimizationStrategy(cfg.Orders.RoutingOptimization))
	orderProcessor := orders.NewProcessor(orderRouter)

	// Initialize pricing engine
	pricingEngine := pricing.NewEngine()
	pricingEngine.SetMinMargin(cfg.Pricing.MinMargin)

	// Set up alert callback
	invEngine.SetAlertCallback(func(alert *models.Alert) {
		log.Printf("Alert [%s]: %s - %s", alert.Severity, alert.Type, alert.Message)
		// TODO: Send to alert channels (Slack, etc.)
	})

	// Start inventory engine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := invEngine.Start(ctx); err != nil {
		log.Fatalf("Failed to start inventory engine: %v", err)
	}

	// Create API server
	server := api.NewServer(cfg, invEngine, orderProcessor, pricingEngine)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      server.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("StreamLine API listening on port %d", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down StreamLine...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	invEngine.Stop()

	log.Println("StreamLine stopped")
}

func loadConfig() *config.Config {
	configPath := os.Getenv("STREAMLINE_CONFIG")
	if configPath != "" {
		cfg, err := config.Load(configPath)
		if err != nil {
			log.Printf("Failed to load config from %s: %v, using defaults", configPath, err)
			return config.LoadFromEnv()
		}
		return cfg
	}
	return config.LoadFromEnv()
}
