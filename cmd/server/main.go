package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/enterprise-ai/orchestrator/config"
	"github.com/enterprise-ai/orchestrator/internal/core/orchestrator"
	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/llm"
	"github.com/gofiber/fiber/v3"
)

func main() {
	// โหลด config
	cfg := config.Load()

	// สร้าง Ollama Adapter
	ollamaURL := fmt.Sprintf("http://%s:%s", cfg.LLMHost, cfg.LLMPort)
	ollamaAdapter := llm.NewOllamaAdapter(ollamaURL, cfg.LLMModel)

	// สร้าง Orchestrator
	orch := orchestrator.New(ollamaAdapter)

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

		// เพิ่ม System Prompt
		messages := append([]models.ChatMessage{
			{Role: "system", Content: cfg.SystemPrompt},
		}, req.Messages...)
		req.Messages = messages

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

	log.Printf("🚀 Server starting on port %s", cfg.APIPort)
	log.Fatal(app.Listen(":" + cfg.APIPort))
}
