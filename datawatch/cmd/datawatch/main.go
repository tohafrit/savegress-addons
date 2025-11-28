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

	"github.com/savegress/datawatch/internal/alerts"
	"github.com/savegress/datawatch/internal/anomaly"
	"github.com/savegress/datawatch/internal/api"
	"github.com/savegress/datawatch/internal/config"
	"github.com/savegress/datawatch/internal/metrics"
	"github.com/savegress/datawatch/internal/quality"
	"github.com/savegress/datawatch/internal/schema"
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

	// Create context for services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	// Initialize alerts engine
	alertsConfig := &alerts.Config{
		Enabled:            true,
		EvaluationInterval: 30 * time.Second,
		RetentionDays:      30,
	}
	alertsEngine := alerts.NewEngine(alertsConfig)

	// Configure alert channels from config
	if cfg.Alerts.Channels.Slack != nil && cfg.Alerts.Channels.Slack.WebhookURL != "" {
		alertsEngine.AddChannel(&alerts.Channel{
			ID:      "slack",
			Type:    alerts.ChannelTypeSlack,
			Name:    "Slack",
			Enabled: true,
			Config: map[string]interface{}{
				"webhook_url": cfg.Alerts.Channels.Slack.WebhookURL,
				"channel":     cfg.Alerts.Channels.Slack.Channel,
			},
		})
	}
	if cfg.Alerts.Channels.Email != nil && cfg.Alerts.Channels.Email.SMTPHost != "" {
		alertsEngine.AddChannel(&alerts.Channel{
			ID:      "email",
			Type:    alerts.ChannelTypeEmail,
			Name:    "Email",
			Enabled: true,
			Config: map[string]interface{}{
				"smtp_host": cfg.Alerts.Channels.Email.SMTPHost,
				"smtp_port": cfg.Alerts.Channels.Email.SMTPPort,
				"from":      cfg.Alerts.Channels.Email.From,
				"username":  cfg.Alerts.Channels.Email.Username,
				"password":  cfg.Alerts.Channels.Email.Password,
			},
		})
	}
	if cfg.Alerts.Channels.PagerDuty != nil && cfg.Alerts.Channels.PagerDuty.ServiceKey != "" {
		alertsEngine.AddChannel(&alerts.Channel{
			ID:      "pagerduty",
			Type:    alerts.ChannelTypePagerDuty,
			Name:    "PagerDuty",
			Enabled: true,
			Config: map[string]interface{}{
				"service_key": cfg.Alerts.Channels.PagerDuty.ServiceKey,
			},
		})
	}
	if cfg.Alerts.Channels.Webhook != nil && cfg.Alerts.Channels.Webhook.URL != "" {
		alertsEngine.AddChannel(&alerts.Channel{
			ID:      "webhook",
			Type:    alerts.ChannelTypeWebhook,
			Name:    "Webhook",
			Enabled: true,
			Config: map[string]interface{}{
				"url":     cfg.Alerts.Channels.Webhook.URL,
				"headers": cfg.Alerts.Channels.Webhook.Headers,
			},
		})
	}

	// Start alerts engine
	if err := alertsEngine.Start(ctx); err != nil {
		log.Printf("Warning: Failed to start alerts engine: %v", err)
	}

	// Set up anomaly callback for alerts
	anomalyDetector.SetAnomalyCallback(func(a *anomaly.Anomaly) {
		log.Printf("Anomaly detected: %s - %s (severity: %s)", a.MetricName, a.Description, a.Severity)

		// Convert anomaly severity to alert severity
		var alertSeverity alerts.Severity
		switch a.Severity {
		case "critical":
			alertSeverity = alerts.SeverityCritical
		case "high":
			alertSeverity = alerts.SeverityHigh
		case "warning":
			alertSeverity = alerts.SeverityWarning
		default:
			alertSeverity = alerts.SeverityInfo
		}

		// Fire alert through alerts engine
		alertsEngine.FireManualAlert(&alerts.Alert{
			Type:           alerts.AlertTypeAnomaly,
			Severity:       alertSeverity,
			Title:          fmt.Sprintf("Anomaly: %s", a.MetricName),
			Message:        a.Description,
			Metric:         a.MetricName,
			CurrentValue:   a.Value,
			ThresholdValue: a.Expected.Max,
			Labels: map[string]string{
				"anomaly_type": string(a.Type),
			},
		})
	})

	// Start metrics engine
	if err := metricsEngine.Start(ctx); err != nil {
		log.Fatalf("Failed to start metrics engine: %v", err)
	}

	// Initialize quality monitor (nil for now, can be configured)
	var qualityMonitor *quality.Monitor
	var schemaTracker *schema.Tracker

	// Create API server
	server := api.NewServer(cfg, metricsEngine, anomalyDetector, store, qualityMonitor, schemaTracker, alertsEngine)

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
	alertsEngine.Stop()

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
