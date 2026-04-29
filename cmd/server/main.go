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
	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	auditInfra "github.com/enterprise-ai/orchestrator/internal/infrastructure/audit"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/auth"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/embedder"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/llm"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/migrations"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/session"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/vector"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	pgxvector "github.com/pgvector/pgvector-go/pgx"
)

func main() {
	// ตั้งค่า Structured Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// โหลด config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid config", "error", err)
		os.Exit(1)
	}
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

	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	migrationCtx, migrationCancel := context.WithTimeout(context.Background(), cfg.MigrationTimeout)
	defer migrationCancel()
	if err := migrations.Up(migrationCtx, sqlDB); err != nil {
		slog.Error("run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations applied")

	// สร้าง Adapters
	ollamaURL := fmt.Sprintf("http://%s:%s", cfg.LLMHost, cfg.LLMPort)
	ollamaAdapter := llm.NewOllamaAdapter(ollamaURL, cfg.LLMModel)
	nomicAdapter := embedder.NewNomicAdapter(ollamaURL, cfg.EmbedModel)
	pgvectorAdapter := vector.NewPgvectorAdapter(pool)

	// สร้าง Session Adapter
	sessionAdapter := session.NewPostgresAdapter(pool, cfg.SessionExpiry)

	// สร้าง Audit Adapter
	auditAdapter := auditInfra.NewPostgresAdapter(pool)
	auditHandler := handlers.NewAuditHandler(auditAdapter)

	// สร้าง Core
	ragEngine := rag.New(nomicAdapter, pgvectorAdapter)
	orch := orchestrator.New(ollamaAdapter, ragEngine, sessionAdapter, cfg.SystemPrompt)

	// สร้าง Auth
	ldapAdapter := auth.NewLDAPAdapter(cfg.ADServer, cfg.ADPort, cfg.ADBaseDN, cfg.ADDomain, cfg.DevMode, cfg.DevUsername, cfg.DevPassword)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiry)
	authHandler := handlers.NewAuthHandler(ldapAdapter, jwtManager)

	// สร้าง Handlers
	healthHandler := handlers.NewHealthHandlerWithTimeout(orch, pgvectorAdapter, cfg.HealthTimeout)
	chatHandler := handlers.NewChatHandlerWithTimeout(orch, cfg.ChatTimeout)
	ragHandler := handlers.NewRAGHandlerWithLimit(ragEngine, cfg.MaxUploadBytes, cfg.RAGIngestTimeout)

	// สร้าง Fiber app
	app := fiber.New(fiber.Config{
		AppName:   "Enterprise AI Orchestrator v0.1",
		BodyLimit: cfg.BodyLimit,
	})

	// Public routes (ไม่ต้อง login)
	app.Post("/auth/login", authHandler.Login)
	app.Get("/live", healthHandler.Live)
	app.Get("/ready", healthHandler.Ready)
	app.Get("/health", healthHandler.Check)
	app.Get("/v1/models", chatHandler.Models)

	protected := app.Group("/", middleware.JWTMiddleware(jwtManager))
	protected.Use(middleware.SessionMiddleware(sessionAdapter))
	protected.Use(middleware.AuditMiddleware(auditAdapter, cfg.AuditTimeout))

	// Rate Limiting — 20 requests per minute per user
	protected.Use(limiter.New(limiter.Config{
		Max:        cfg.RateLimitPerMin,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c fiber.Ctx) string {
			// ใช้ user_id เป็น key ถ้ามี JWT
			if claims, ok := c.Locals("claims").(*models.Claims); ok && claims != nil {
				return claims.UserID
			}
			// fallback ใช้ IP
			return c.IP()
		},
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "too many requests, please slow down",
					"code":    "RATE_LIMIT_EXCEEDED",
				},
			})
		},
	}))

	protected.Get("/auth/me", authHandler.Me)
	protected.Post("/v1/chat/completions", chatHandler.Completions)
	protected.Post("/v1/rag/ingest", ragHandler.Ingest)
	protected.Post("/v1/rag/upload", ragHandler.Upload)

	// Admin routes
	admin := app.Group("/v1/admin", middleware.JWTMiddleware(jwtManager))
	admin.Use(middleware.RequireRole("admin"))
	admin.Get("/logs", auditHandler.List)

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
