package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
)

type healthOrchestrator interface {
	HealthCheck(ctx context.Context) error
	EmbedderCheck(ctx context.Context) error
}

type healthDatabase interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	orch healthOrchestrator
	db   healthDatabase
}

func NewHealthHandler(orch healthOrchestrator, db healthDatabase) *HealthHandler {
	return &HealthHandler{orch: orch, db: db}
}

// Live ตรวจว่า process ยังตอบ HTTP ได้ โดยไม่แตะ dependency ภายนอก
func (h *HealthHandler) Live(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"status": "alive",
		},
	})
}

// Check คง endpoint /health เดิมไว้ โดยให้ semantics เท่ากับ readiness
func (h *HealthHandler) Check(c fiber.Ctx) error {
	return h.Ready(c)
}

// Ready ตรวจ dependency ที่จำเป็นก่อนรับ traffic
func (h *HealthHandler) Ready(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := fiber.Map{
		"status": "ready",
	}
	ready := true

	if err := h.orch.HealthCheck(ctx); err != nil {
		ready = false
		status["llm"] = "unhealthy: " + err.Error()
	} else {
		status["llm"] = "healthy"
	}

	if err := h.db.Ping(ctx); err != nil {
		ready = false
		status["db"] = "unhealthy: " + err.Error()
	} else {
		status["db"] = "healthy"
	}

	if err := h.orch.EmbedderCheck(ctx); err != nil {
		ready = false
		status["embedder"] = "unhealthy: " + err.Error()
	} else {
		status["embedder"] = "healthy"
	}

	code := 200
	if !ready {
		status["status"] = "not_ready"
		code = 503
	}

	return c.Status(code).JSON(fiber.Map{"data": status})
}
