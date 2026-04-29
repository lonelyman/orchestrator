package ports

import (
	"context"
	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

// AuditPort คือ interface สำหรับบันทึก Audit Log
type AuditPort interface {
	// Save บันทึก audit log
	Save(ctx context.Context, log models.AuditLog) error

	// List ดึง audit logs ตาม filter
	List(ctx context.Context, filter models.AuditLogFilter) ([]models.AuditLog, error)
}