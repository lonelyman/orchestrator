package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/enterprise-ai/orchestrator/config"
	"github.com/enterprise-ai/orchestrator/internal/core/orchestrator"
	"github.com/enterprise-ai/orchestrator/internal/core/rag"
	"github.com/enterprise-ai/orchestrator/internal/domain/models"
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

	// Register pgvector types สำหรับทุก connection ใน pool
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
	nomicAdapter := embedder.NewNomicAdapter(ollamaURL)
	pgvectorAdapter := vector.NewPgvectorAdapter(pool)

	// สร้าง Schema ใน PostgreSQL
	ctx := context.Background()
	if err := pgvectorAdapter.InitSchema(ctx); err != nil {
		log.Fatalf("init schema: %v", err)
	}
	log.Println("✅ pgvector schema ready")

	// สร้าง RAG Engine
	ragEngine := rag.New(nomicAdapter, pgvectorAdapter)

	// สร้าง Orchestrator
	orch := orchestrator.New(ollamaAdapter, ragEngine)

	// สร้าง Fiber v3 app
	app := fiber.New(fiber.Config{
		AppName: "Enterprise AI Orchestrator v0.1",
	})

	// Health check endpoint
	app.Get("/health", func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := orch.HealthCheck(ctx); err != nil {
			return c.Status(503).JSON(fiber.Map{
				"status": "unhealthy",
				"error":  err.Error(),
			})
		}
		return c.JSON(fiber.Map{"status": "healthy"})
	})

	// OpenAI-compatible chat endpoint
	app.Post("/v1/chat/completions", func(c fiber.Ctx) error {
		var req models.ChatRequest
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		result, err := orch.Chat(ctx, req)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(models.ChatResponse{
			ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []models.Choice{
				{
					Index:        0,
					Message:      models.ChatMessage{Role: "assistant", Content: result},
					FinishReason: "stop",
				},
			},
		})
	})

	// RAG Ingest endpoint
	app.Post("/v1/rag/ingest", func(c fiber.Ctx) error {
		var req struct {
			Content string `json:"content"`
			Source  string `json:"source"`
		}
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := ragEngine.Ingest(ctx, req.Content, req.Source); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"status": "ingested", "source": req.Source})
	})

	log.Printf("🚀 Server starting on port %s", cfg.APIPort)
	log.Fatal(app.Listen(":" + cfg.APIPort))
}
