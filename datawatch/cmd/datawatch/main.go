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

	"github.com/savegress/datawatch/internal/anomaly"
	"github.com/savegress/datawatch/internal/api"
	"github.com/savegress/datawatch/internal/config"
	"github.com/savegress/datawatch/internal/metrics"
	"github.com/savegress/datawatch/internal/storage"
)

func main() {
	log.Println("Starting DataWatch...")

	// Load configuration
	cfg := loadConfig()

	// Initialize storage
	store, err := initStorage(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize metrics engine
	metricsEngine := metrics.NewEngine(store)

	// Initialize anomaly detector
	anomalyConfig := anomaly.DetectorConfig{
		Algorithms:     []anomaly.Algorithm{anomaly.AlgorithmStatistical, anomaly.AlgorithmSeasonal},
		Sensitivity:    cfg.Anomaly.Sensitivity,
		BaselineWindow: cfg.Anomaly.BaselineWindow,
		MinDataPoints:  100,
	}
	anomalyDetector := anomaly.NewDetector(anomalyConfig, store)

	// Set up anomaly callback for alerts
	anomalyDetector.SetAnomalyCallback(func(a *anomaly.Anomaly) {
		log.Printf("Anomaly detected: %s - %s (severity: %s)", a.MetricName, a.Description, a.Severity)
		// TODO: Send to alert channels (Slack, PagerDuty, etc.)
	})

	// Start metrics engine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := metricsEngine.Start(ctx); err != nil {
		log.Fatalf("Failed to start metrics engine: %v", err)
	}

	// Create API server
	server := api.NewServer(cfg, metricsEngine, anomalyDetector, store)

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
		log.Printf("DataWatch API listening on port %d", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down DataWatch...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	metricsEngine.Stop()

	log.Println("DataWatch stopped")
}

func loadConfig() *config.Config {
	configPath := os.Getenv("DATAWATCH_CONFIG")
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

func initStorage(cfg *config.Config) (storage.MetricStorage, error) {
	switch cfg.Metrics.StorageType {
	case "embedded":
		dataPath := "/var/lib/datawatch/data"
		if cfg.Storage.Embedded != nil && cfg.Storage.Embedded.Path != "" {
			dataPath = cfg.Storage.Embedded.Path
		}
		// Use temp dir in development
		if cfg.Server.Environment == "development" {
			dataPath = "/tmp/datawatch/data"
		}
		return storage.NewEmbeddedStorage(dataPath)

	// TODO: Implement other storage backends
	// case "prometheus":
	// case "influxdb":
	// case "timescale":

	default:
		return storage.NewEmbeddedStorage("/tmp/datawatch/data")
	}
}
