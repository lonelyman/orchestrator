package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/enterprise-ai/orchestrator/config"
	"github.com/enterprise-ai/orchestrator/internal/app"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid config", "error", err)
		os.Exit(1)
	}
	slog.Info("config loaded", "port", cfg.APIPort, "llm_backend", cfg.LLMBackend, "model", cfg.LLMModel, "embed", cfg.EmbedModel)

	server, err := app.New(context.Background(), cfg)
	if err != nil {
		slog.Error("initialize server", "error", err)
		os.Exit(1)
	}
	defer server.Close()

	go func() {
		slog.Info("server starting", "port", cfg.APIPort)
		if err := server.App.Listen(":" + cfg.APIPort); err != nil {
			slog.Error("server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down gracefully...")
	if err := server.App.ShutdownWithTimeout(10 * time.Second); err != nil {
		slog.Error("force shutdown", "error", err)
	}
	slog.Info("server stopped cleanly")
}
