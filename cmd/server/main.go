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

	"isl-api-server/config"
	"isl-api-server/internal/router"
	"isl-api-server/internal/util"
)

func main() {
	// Load configuration first
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration", err)
	}

	// Initialize logger with file logging
	logger, err := util.NewLoggerWithFile(cfg.LogPath)
	if err != nil {
		log.Fatal("Failed to initialize logger", err)
	}
	defer logger.Close()

	// Redirect standard log output to file
	log.SetOutput(logger.GetLogFile())
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	logger.Info("Starting ISL API Server...")

	// Create Gin router
	ginRouter := router.NewRouter(cfg, logger)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      ginRouter,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.IdleTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", "error", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exited")
}
