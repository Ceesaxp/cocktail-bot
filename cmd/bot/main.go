package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ceesaxp/cocktail-bot/internal/api"
	"github.com/ceesaxp/cocktail-bot/internal/config"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
	"github.com/ceesaxp/cocktail-bot/internal/service"
	"github.com/ceesaxp/cocktail-bot/internal/telegram"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	// Initialize context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	l := logger.New(cfg.LogLevel)
	l.Info("Starting Cocktail Bot")

	// Initialize service
	svc, err := service.New(ctx, cfg, l)
	if err != nil {
		l.Fatal("Failed to initialize service", "error", err)
	}

	// Initialize bot
	bot, err := telegram.NewFromToken(cfg.Telegram.Token, svc, l, cfg)
	if err != nil {
		l.Fatal("Failed to initialize Telegram bot", "error", err)
	}

	// Start bot in a separate goroutine
	if err := bot.Start(); err != nil {
		l.Fatal("Failed to start bot", "error", err)
	}

	// Initialize and start API server if enabled
	var apiServer *api.Server
	if cfg.API.Enabled {
		apiServer, err = api.New(cfg, svc, l)
		if err != nil {
			l.Fatal("Failed to initialize API server", "error", err)
		}

		if err := apiServer.Start(); err != nil {
			l.Fatal("Failed to start API server", "error", err)
		}
		l.Info("API server started", "port", cfg.API.Port)
	}

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Log startup complete
	l.Info("Bot is running. Press Ctrl+C to stop")

	// Wait for termination signal
	<-sigCh
	l.Info("Received termination signal")

	// Graceful shutdown
	l.Info("Shutting down bot")
	bot.Stop()

	// Stop API server if running
	if apiServer != nil {
		l.Info("Shutting down API server")
		if err := apiServer.Stop(); err != nil {
			l.Error("Error stopping API server", "error", err)
		}
		l.Info("API server stopped")
	}

	// Close service
	if err := svc.Close(); err != nil {
		l.Error("Error closing service", "error", err)
	}

	l.Info("Bot stopped")
}
