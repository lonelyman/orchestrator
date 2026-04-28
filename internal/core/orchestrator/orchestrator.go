package orchestrator

import (
	"context"
	"fmt"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/enterprise-ai/orchestrator/internal/domain/ports"
)

// Orchestrator คือสมองกลางของระบบ
// Phase 0: ส่งต่อไปยัง LLM โดยตรง
// Phase 3: จะเพิ่ม Intent Router (RAG/MCP/Direct)
type Orchestrator struct {
	llm ports.LLMPort
}

// New สร้าง Orchestrator ใหม่
func New(llm ports.LLMPort) *Orchestrator {
	return &Orchestrator{llm: llm}
}

// Chat รับ request และส่งคำตอบกลับ
func (o *Orchestrator) Chat(ctx context.Context, req models.ChatRequest) (string, error) {
	// Phase 0: ส่งตรงไปยัง LLM
	// Phase 3: ตรงนี้จะเพิ่ม Intent Classification
	result, err := o.llm.Chat(ctx, req.Messages)
	if err != nil {
		return "", fmt.Errorf("llm chat: %w", err)
	}

	return result, nil
}

// ChatStream รับ request และ stream คำตอบกลับทีละ chunk
func (o *Orchestrator) ChatStream(ctx context.Context, req models.ChatRequest, onChunk func(string)) error {
	// Phase 0: stream ตรงไปยัง LLM
	return o.llm.ChatStream(ctx, req.Messages, onChunk)
}

// HealthCheck ตรวจสอบว่าระบบพร้อมใช้งาน
func (o *Orchestrator) HealthCheck(ctx context.Context) error {
	return o.llm.HealthCheck(ctx)
}
