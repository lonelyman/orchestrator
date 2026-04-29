package ports

import (
	"context"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

// SessionPort คือ interface สำหรับจัดการ Session
type SessionPort interface {
	// CreateSession สร้าง Session ใหม่
	CreateSession(ctx context.Context, userID string) (*models.Session, error)

	// GetSession ดึง Session จาก ID
	GetSession(ctx context.Context, sessionID string) (*models.Session, error)

	// GetHistory ดึง messages ล่าสุด N ข้อความ (Sliding Window)
	GetHistory(ctx context.Context, sessionID string, limit int) ([]models.Message, error)

	// SaveMessage บันทึก message ลง DB
	SaveMessage(ctx context.Context, msg models.Message) error

	// SaveMessages บันทึก messages หลายรายการตามลำดับใน transaction เดียว
	SaveMessages(ctx context.Context, messages []models.Message) error

	// ExpireSession ปิด Session
	ExpireSession(ctx context.Context, sessionID string) error
}
