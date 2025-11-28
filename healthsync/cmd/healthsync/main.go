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

	"github.com/savegress/healthsync/internal/anonymization"
	"github.com/savegress/healthsync/internal/api"
	"github.com/savegress/healthsync/internal/audit"
	"github.com/savegress/healthsync/internal/compliance"
	"github.com/savegress/healthsync/internal/config"
	"github.com/savegress/healthsync/internal/consent"
)

func main() {
	log.Println("Starting HealthSync...")

	// Load configuration
	cfg := loadConfig()

	// Initialize compliance engine
	complianceEngine := compliance.NewEngine(&cfg.Compliance)

	// Initialize audit logger
	auditLogger := audit.NewLogger(&cfg.Audit)

	// Initialize anonymization engine
	anonEngine := anonymization.NewEngine(&anonymization.Config{
		Method:           "safe_harbor",
		DateShiftRange:   365,
		PreserveBirthYear: true,
		ZipCodeTruncation: 3,
		Salt:             cfg.Server.JWTSecret,
	})

	// Initialize consent manager
	consentManager := consent.NewManager(&cfg.Consent)

	// Start engines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := complianceEngine.Start(ctx); err != nil {
		log.Fatalf("Failed to start compliance engine: %v", err)
	}

	if err := auditLogger.Start(ctx); err != nil {
		log.Fatalf("Failed to start audit logger: %v", err)
	}

	// Create API server
	server := api.NewServer(cfg, complianceEngine, auditLogger, anonEngine, consentManager)

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
		log.Printf("HealthSync API listening on port %d", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down HealthSync...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	complianceEngine.Stop()
	auditLogger.Stop()

	log.Println("HealthSync stopped")
}

func loadConfig() *config.Config {
	configPath := os.Getenv("HEALTHSYNC_CONFIG")
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
