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
	orch    healthOrchestrator
	db      healthDatabase
	timeout time.Duration
}

func NewHealthHandler(orch healthOrchestrator, db healthDatabase) *HealthHandler {
	return NewHealthHandlerWithTimeout(orch, db, 5*time.Second)
}

func NewHealthHandlerWithTimeout(orch healthOrchestrator, db healthDatabase, timeout time.Duration) *HealthHandler {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &HealthHandler{orch: orch, db: db, timeout: timeout}
}

// Live ตรวจว่า process ยังตอบ HTTP ได้ โดยไม่แตะ dependency ภายนอก
func (h *HealthHandler) Live(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"status": "alive",
		},
	})
}

// Check ตรวจสถานะ dependency ทั้งหมดสำหรับ monitoring
func (h *HealthHandler) Check(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	status := fiber.Map{
		"status": "healthy",
	}
	healthy := true

	if err := h.orch.HealthCheck(ctx); err != nil {
		healthy = false
		status["llm"] = "unhealthy: " + err.Error()
	} else {
		status["llm"] = "healthy"
	}

	if err := h.db.Ping(ctx); err != nil {
		healthy = false
		status["db"] = "unhealthy: " + err.Error()
	} else {
		status["db"] = "healthy"
	}

	if err := h.orch.EmbedderCheck(ctx); err != nil {
		healthy = false
		status["embedder"] = "unhealthy: " + err.Error()
	} else {
		status["embedder"] = "healthy"
	}

	code := 200
	if !healthy {
		status["status"] = "unhealthy"
		code = 503
	}

	return c.Status(code).JSON(fiber.Map{"data": status})
}

// Ready ตรวจ dependency ขั้นต่ำที่จำเป็นก่อนรับ traffic
func (h *HealthHandler) Ready(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	status := fiber.Map{
		"status": "ready",
	}

	if err := h.db.Ping(ctx); err != nil {
		status["status"] = "not_ready"
		status["db"] = "unhealthy: " + err.Error()
		return c.Status(503).JSON(fiber.Map{"data": status})
	}

	status["db"] = "healthy"
	return c.JSON(fiber.Map{"data": status})
}
