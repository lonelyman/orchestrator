package main

import (
	"context"
	"fmt"
	"log"

	"github.com/enterprise-ai/orchestrator/config"
	"github.com/enterprise-ai/orchestrator/internal/api/handlers"
	"github.com/enterprise-ai/orchestrator/internal/core/orchestrator"
	"github.com/enterprise-ai/orchestrator/internal/core/rag"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/embedder"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/llm"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/vector"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvector "github.com/pgvector/pgvector-go/pgx"
)

func main() {
	// โหลด config
	cfg := config.Load()

	// เชื่อมต่อ PostgreSQL
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("parse db config: %v", err)
	}

	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvector.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()
	log.Println("✅ PostgreSQL connected")

	// สร้าง Adapters
	ollamaURL := fmt.Sprintf("http://%s:%s", cfg.LLMHost, cfg.LLMPort)
	ollamaAdapter := llm.NewOllamaAdapter(ollamaURL, cfg.LLMModel)
	nomicAdapter := embedder.NewNomicAdapter(ollamaURL, cfg.EmbedModel)
	pgvectorAdapter := vector.NewPgvectorAdapter(pool)

	// สร้าง Schema
	if err := pgvectorAdapter.InitSchema(context.Background()); err != nil {
		log.Fatalf("init schema: %v", err)
	}
	log.Println("✅ pgvector schema ready")

	// สร้าง Core
	ragEngine := rag.New(nomicAdapter, pgvectorAdapter)
	orch := orchestrator.New(ollamaAdapter, ragEngine, cfg.SystemPrompt)

	// สร้าง Handlers
	healthHandler := handlers.NewHealthHandler(orch, pgvectorAdapter)
	chatHandler := handlers.NewChatHandler(orch)
	ragHandler := handlers.NewRAGHandler(ragEngine)

	// สร้าง Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Enterprise AI Orchestrator v0.1",
	})

	// Routes
	app.Get("/health", healthHandler.Check)
	app.Get("/v1/models", chatHandler.Models)
	app.Post("/v1/chat/completions", chatHandler.Completions)
	app.Post("/v1/rag/ingest", ragHandler.Ingest)
	app.Post("/v1/rag/upload", ragHandler.Upload)

	log.Printf("🚀 Server starting on port %s", cfg.APIPort)
	log.Fatal(app.Listen(":" + cfg.APIPort))
}
