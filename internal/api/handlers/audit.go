package handlers

import (
	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/enterprise-ai/orchestrator/internal/domain/ports"
	"github.com/gofiber/fiber/v3"
)

// AuditHandler จัดการ Audit Log endpoints
type AuditHandler struct {
	audit ports.AuditPort
}

func NewAuditHandler(audit ports.AuditPort) *AuditHandler {
	return &AuditHandler{audit: audit}
}

// List ดึง audit logs — Admin only
func (h *AuditHandler) List(c fiber.Ctx) error {
	filter := models.AuditLogFilter{
		UserID: c.Query("user_id"),
		Intent: c.Query("intent"),
		Limit:  50,
		Offset: 0,
	}

	logs, err := h.audit.List(c.Context(), filter)
	if err != nil {
		return Fail(c, 500, err.Error(), "SERVER_ERROR")
	}

	return OK(c, fiber.Map{
		"logs":  logs,
		"count": len(logs),
	})
}
