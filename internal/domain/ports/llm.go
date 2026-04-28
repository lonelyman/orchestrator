package ports

import (
	"context"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

// LLMPort คือ "สัญญา" ว่า LLM ใดๆ ต้องทำได้
// ไม่ว่าจะเป็น Ollama, vLLM, หรือ OpenAI
// Core logic จะเรียกผ่าน Interface นี้เท่านั้น
type LLMPort interface {
	// Chat ส่งข้อความและรอรับคำตอบ
	Chat(ctx context.Context, messages []models.ChatMessage) (string, error)

	// ChatStream ส่งข้อความและรับคำตอบแบบ stream ทีละ chunk
	ChatStream(ctx context.Context, messages []models.ChatMessage, onChunk func(string)) error

	// HealthCheck ตรวจสอบว่า LLM พร้อมใช้งานไหม
	HealthCheck(ctx context.Context) error
}
