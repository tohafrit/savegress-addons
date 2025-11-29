package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // Register pprof handlers
	"os"
	"os/signal"
	"syscall"
	"time"

	"getchainlens.com/chainlens/backend/internal/api"
	"getchainlens.com/chainlens/backend/internal/config"
	"getchainlens.com/chainlens/backend/internal/database"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize router
	router := api.NewRouter(cfg, db)

	// Start pprof server if enabled
	if cfg.EnablePprof {
		go func() {
			pprofAddr := fmt.Sprintf("localhost:%d", cfg.PprofPort)
			log.Printf("Starting pprof server on %s", pprofAddr)
			log.Printf("  CPU profile: go tool pprof http://%s/debug/pprof/profile?seconds=30", pprofAddr)
			log.Printf("  Heap profile: go tool pprof http://%s/debug/pprof/heap", pprofAddr)
			log.Printf("  Goroutines:   go tool pprof http://%s/debug/pprof/goroutine", pprofAddr)
			if err := http.ListenAndServe(pprofAddr, nil); err != nil {
				log.Printf("pprof server error: %v", err)
			}
		}()
	}

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("ChainLens API server starting on port %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
