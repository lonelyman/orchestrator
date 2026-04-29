package handlers

import (
	"context"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/core/orchestrator"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/vector"
	"github.com/gofiber/fiber/v3"
)

type HealthHandler struct {
	orch     *orchestrator.Orchestrator
	pgvector *vector.PgvectorAdapter
}

func NewHealthHandler(orch *orchestrator.Orchestrator, pgvector *vector.PgvectorAdapter) *HealthHandler {
	return &HealthHandler{orch: orch, pgvector: pgvector}
}

func (h *HealthHandler) Check(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := fiber.Map{
		"status": "healthy",
	}

	// เช็ค LLM (Ollama)
	if err := h.orch.HealthCheck(ctx); err != nil {
		status["status"] = "unhealthy"
		status["llm"] = "unhealthy: " + err.Error()
	} else {
		status["llm"] = "healthy"
	}

	// เช็ค PostgreSQL
	if err := h.pgvector.Ping(ctx); err != nil {
		status["status"] = "unhealthy"
		status["db"] = "unhealthy: " + err.Error()
	} else {
		status["db"] = "healthy"
	}

	// เช็ค Embedder
	if err := h.orch.EmbedderCheck(ctx); err != nil {
		status["status"] = "degraded"
		status["embedder"] = "unhealthy: " + err.Error()
	} else {
		status["embedder"] = "healthy"
	}

	code := 200
	if status["status"] == "unhealthy" {
		code = 503
	}

	return c.Status(code).JSON(fiber.Map{"data": status})
}
