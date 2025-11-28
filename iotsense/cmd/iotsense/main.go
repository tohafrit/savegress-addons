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

	"github.com/savegress/iotsense/internal/alerts"
	"github.com/savegress/iotsense/internal/api"
	"github.com/savegress/iotsense/internal/config"
	"github.com/savegress/iotsense/internal/devices"
	"github.com/savegress/iotsense/internal/telemetry"
	"github.com/savegress/iotsense/pkg/models"
)

func main() {
	// Load configuration
	var cfg *config.Config
	if configPath := os.Getenv("CONFIG_PATH"); configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	} else {
		cfg = config.LoadFromEnv()
	}

	log.Printf("Starting IoTSense - IoT Device Telemetry Platform")
	log.Printf("Environment: %s", cfg.Server.Environment)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize device registry
	registry := devices.NewRegistry(&cfg.Devices)
	if err := registry.Start(ctx); err != nil {
		log.Fatalf("Failed to start device registry: %v", err)
	}
	log.Println("Device registry started")

	// Initialize telemetry engine
	telemetryEngine := telemetry.NewEngine(&cfg.Telemetry)
	if err := telemetryEngine.Start(ctx); err != nil {
		log.Fatalf("Failed to start telemetry engine: %v", err)
	}
	log.Println("Telemetry engine started")

	// Initialize alerting engine
	alertsEngine := alerts.NewEngine(&cfg.Alerts)
	if err := alertsEngine.Start(ctx); err != nil {
		log.Fatalf("Failed to start alerts engine: %v", err)
	}
	log.Println("Alerts engine started")

	// Setup notifiers
	if cfg.Alerts.Channels.Slack.WebhookURL != "" {
		alertsEngine.AddNotifier(alerts.NewSlackNotifier(cfg.Alerts.Channels.Slack))
		log.Println("Slack notifier configured")
	}
	if cfg.Alerts.Channels.Webhook.URL != "" {
		alertsEngine.AddNotifier(alerts.NewWebhookNotifier(cfg.Alerts.Channels.Webhook))
		log.Println("Webhook notifier configured")
	}
	if cfg.Server.Environment == "development" {
		alertsEngine.AddNotifier(alerts.NewConsoleNotifier())
	}

	// Wire up alerts callback
	alertsEngine.SetAlertCallback(func(alert *models.Alert) {
		log.Printf("Alert triggered: %s (severity: %s)", alert.Title, alert.Severity)
	})

	// Create API server
	server := api.NewServer(registry, telemetryEngine, alertsEngine)

	// Start HTTP server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: server.Handler(),
	}

	go func() {
		log.Printf("HTTP server listening on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	alertsEngine.Stop()
	telemetryEngine.Stop()
	registry.Stop()

	log.Println("IoTSense stopped")
}
