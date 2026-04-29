package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server, err := app.New(rootCtx, cfg)
	if err != nil {
		slog.Error("initialize server", "error", err)
		os.Exit(1)
	}
	defer server.Close()

	listenErr := make(chan error, 1)
	go func() {
		slog.Info("server starting", "port", cfg.APIPort)
		err := server.App.Listen(":" + cfg.APIPort)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		listenErr <- err
	}()

	var exitCode int
	listenReturned := false
	select {
	case <-rootCtx.Done():
		slog.Info("shutdown signal received")
	case err := <-listenErr:
		listenReturned = true
		if err != nil {
			slog.Error("server listen failed", "error", err)
			exitCode = 1
		} else {
			slog.Info("server stopped")
		}
	}
	stop()

	slog.Info("shutting down gracefully", "timeout", cfg.ShutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("force shutdown", "error", err)
		exitCode = 1
	}
	if !listenReturned {
		err := <-listenErr
		if err != nil {
			slog.Error("server stopped with error", "error", err)
			exitCode = 1
		}
	} else if exitCode == 0 {
		slog.Info("listener already stopped")
	}
	slog.Info("server stopped cleanly")
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
