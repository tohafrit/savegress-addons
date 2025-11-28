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

	"github.com/savegress/finsight/internal/api"
	"github.com/savegress/finsight/internal/config"
	"github.com/savegress/finsight/internal/fraud"
	"github.com/savegress/finsight/internal/reconciliation"
	"github.com/savegress/finsight/internal/reporting"
	"github.com/savegress/finsight/internal/transactions"
)

func main() {
	log.Println("Starting FinSight...")

	// Load configuration
	cfg := loadConfig()

	// Initialize transaction engine
	txnEngine := transactions.NewEngine(&cfg.Transactions)

	// Initialize fraud detector
	fraudDetector := fraud.NewDetector(&cfg.Fraud)

	// Initialize reconciliation engine
	reconEngine := reconciliation.NewEngine(&cfg.Reconciliation)

	// Initialize report generator
	reportGen := reporting.NewGenerator(&cfg.Reporting)

	// Start engines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := txnEngine.Start(ctx); err != nil {
		log.Fatalf("Failed to start transaction engine: %v", err)
	}

	if err := fraudDetector.Start(ctx); err != nil {
		log.Fatalf("Failed to start fraud detector: %v", err)
	}

	if err := reconEngine.Start(ctx); err != nil {
		log.Fatalf("Failed to start reconciliation engine: %v", err)
	}

	// Create API server
	server := api.NewServer(cfg, txnEngine, fraudDetector, reconEngine, reportGen)

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
		log.Printf("FinSight API listening on port %d", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down FinSight...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	txnEngine.Stop()
	fraudDetector.Stop()
	reconEngine.Stop()

	log.Println("FinSight stopped")
}

func loadConfig() *config.Config {
	configPath := os.Getenv("FINSIGHT_CONFIG")
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
