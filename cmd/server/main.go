package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/enterprise-ai/orchestrator/config"
	"github.com/enterprise-ai/orchestrator/internal/api/handlers"
	"github.com/enterprise-ai/orchestrator/internal/api/middleware"
	"github.com/enterprise-ai/orchestrator/internal/core/orchestrator"
	"github.com/enterprise-ai/orchestrator/internal/core/rag"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/auth"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/embedder"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/llm"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/session"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/vector"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvector "github.com/pgvector/pgvector-go/pgx"
)

func main() {
	// ตั้งค่า Structured Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// โหลด config
	cfg := config.Load()
	slog.Info("config loaded", "port", cfg.APIPort, "model", cfg.LLMModel, "embed", cfg.EmbedModel)

	// เชื่อมต่อ PostgreSQL
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		slog.Error("parse db config", "error", err)
		os.Exit(1)
	}

	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvector.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		slog.Error("connect postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("postgresql connected")

	// สร้าง Adapters
	ollamaURL := fmt.Sprintf("http://%s:%s", cfg.LLMHost, cfg.LLMPort)
	ollamaAdapter := llm.NewOllamaAdapter(ollamaURL, cfg.LLMModel)
	nomicAdapter := embedder.NewNomicAdapter(ollamaURL, cfg.EmbedModel)
	pgvectorAdapter := vector.NewPgvectorAdapter(pool)

	// สร้าง Session Adapter
	sessionAdapter := session.NewPostgresAdapter(pool, 30*time.Minute)

	// สร้าง Schema
	if err := pgvectorAdapter.InitSchema(context.Background()); err != nil {
		slog.Error("init schema", "error", err)
		os.Exit(1)
	}
	slog.Info("pgvector schema ready")

	// สร้าง Core
	ragEngine := rag.New(nomicAdapter, pgvectorAdapter)
	orch := orchestrator.New(ollamaAdapter, ragEngine, sessionAdapter, cfg.SystemPrompt)

	// สร้าง Auth
	ldapAdapter := auth.NewLDAPAdapter(cfg.ADServer, cfg.ADPort, cfg.ADBaseDN, cfg.ADDomain)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, 8*time.Hour)
	authHandler := handlers.NewAuthHandler(ldapAdapter, jwtManager)

	// สร้าง Handlers
	healthHandler := handlers.NewHealthHandler(orch, pgvectorAdapter)
	chatHandler := handlers.NewChatHandler(orch)
	ragHandler := handlers.NewRAGHandler(ragEngine)

	// สร้าง Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Enterprise AI Orchestrator v0.1",
	})

	// Public routes (ไม่ต้อง login)
	app.Post("/auth/login", authHandler.Login)
	app.Get("/health", healthHandler.Check)
	app.Get("/v1/models", chatHandler.Models)

	// Protected routes (ต้อง login)
	protected := app.Group("/", middleware.JWTMiddleware(jwtManager))
	protected.Use(middleware.SessionMiddleware(sessionAdapter))
	protected.Get("/auth/me", authHandler.Me)
	protected.Post("/v1/chat/completions", chatHandler.Completions)
	protected.Post("/v1/rag/ingest", ragHandler.Ingest)
	protected.Post("/v1/rag/upload", ragHandler.Upload)

	// Routes
	app.Get("/health", healthHandler.Check)
	app.Get("/v1/models", chatHandler.Models)
	app.Post("/v1/chat/completions", chatHandler.Completions)
	app.Post("/v1/rag/ingest", ragHandler.Ingest)
	app.Post("/v1/rag/upload", ragHandler.Upload)

	// Graceful Shutdown
	go func() {
		slog.Info("server starting", "port", cfg.APIPort)
		if err := app.Listen(":" + cfg.APIPort); err != nil {
			slog.Error("server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down gracefully...")
	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		slog.Error("force shutdown", "error", err)
	}
	pool.Close()
	slog.Info("server stopped cleanly")
}
