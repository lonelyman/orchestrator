package orchestrator

import (
	"context"
	"fmt"
	"log"

	"github.com/enterprise-ai/orchestrator/internal/core/rag"
	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/enterprise-ai/orchestrator/internal/domain/ports"
)

// Orchestrator คือสมองกลางของระบบ
type Orchestrator struct {
	llm ports.LLMPort
	rag *rag.RAGEngine
}

// New สร้าง Orchestrator ใหม่
func New(llm ports.LLMPort, rag *rag.RAGEngine) *Orchestrator {
	return &Orchestrator{llm: llm, rag: rag}
}

// Chat รับ request และส่งคำตอบกลับ
func (o *Orchestrator) Chat(ctx context.Context, req models.ChatRequest) (string, error) {
	// ดึงคำถามล่าสุด
	lastMsg := req.Messages[len(req.Messages)-1].Content

	// ค้นหา context จาก RAG
	docs, err := o.rag.Search(ctx, lastMsg, 3)
	log.Printf("RAG Search: query=%s, docs=%d, err=%v", lastMsg, len(docs), err)

	var systemContent string
	systemContent = "You are a helpful enterprise AI assistant. You must always respond in Thai language only."

	if err == nil && len(docs) > 0 {
		ragContext := o.rag.BuildContext(docs)
		systemContent = systemContent + "\n\nUse the following information to answer the question:\n\n" + ragContext
	}

	// รวม system prompt + RAG context + messages
	messages := append([]models.ChatMessage{
		{Role: "system", Content: systemContent},
	}, req.Messages...)

	result, err := o.llm.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("llm chat: %w", err)
	}

	return result, nil
}

// ChatStream รับ request และ stream คำตอบกลับทีละ chunk
func (o *Orchestrator) ChatStream(ctx context.Context, req models.ChatRequest, onChunk func(string)) error {
	return o.llm.ChatStream(ctx, req.Messages, onChunk)
}

// HealthCheck ตรวจสอบว่าระบบพร้อมใช้งาน
func (o *Orchestrator) HealthCheck(ctx context.Context) error {
	return o.llm.HealthCheck(ctx)
}
