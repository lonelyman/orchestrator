package handlers

import (
	"context"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/core/orchestrator"
	"github.com/gofiber/fiber/v3"
)

type HealthHandler struct {
	orch *orchestrator.Orchestrator
}

func NewHealthHandler(orch *orchestrator.Orchestrator) *HealthHandler {
	return &HealthHandler{orch: orch}
}

func (h *HealthHandler) Check(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.orch.HealthCheck(ctx); err != nil {
		return Fail(c, 503, err.Error(), "SERVICE_UNAVAILABLE")
	}

	return OK(c, fiber.Map{"status": "healthy"})
}
