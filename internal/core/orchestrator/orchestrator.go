package orchestrator

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/enterprise-ai/orchestrator/internal/core/intent"
	"github.com/enterprise-ai/orchestrator/internal/core/rag"
	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/enterprise-ai/orchestrator/internal/domain/ports"
)

// Orchestrator คือสมองกลางของระบบ
type Orchestrator struct {
	llm          ports.LLMPort
	rag          *rag.RAGEngine
	classifier   *intent.Classifier
	systemPrompt string
}

// New สร้าง Orchestrator ใหม่
func New(llm ports.LLMPort, rag *rag.RAGEngine, systemPrompt string) *Orchestrator {
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = "You are a helpful enterprise AI assistant. You must always respond in Thai language only."
	}

	return &Orchestrator{
		llm:          llm,
		rag:          rag,
		classifier:   intent.New(),
		systemPrompt: systemPrompt,
	}
}

// Chat รับ request และส่งคำตอบกลับ
func (o *Orchestrator) Chat(ctx context.Context, req models.ChatRequest) (string, error) {
	if len(req.Messages) == 0 {
		return "", fmt.Errorf("invalid request: messages is required")
	}

	// ดึงคำถามล่าสุด
	lastMsg := req.Messages[len(req.Messages)-1].Content

	// Phase 3: Intent Classification
	intentResult := o.classifier.Classify(lastMsg)
	log.Printf("Intent: query=%s, intent=%s, confidence=%.2f, reason=%s",
		lastMsg, intentResult.Intent, intentResult.Confidence, intentResult.Reason)

	systemContent := o.systemPrompt

	switch intentResult.Intent {
	case models.IntentRAG:
		// ค้นหาจาก Vector DB
		docs, err := o.rag.Search(ctx, lastMsg, 3)
		log.Printf("RAG Search: docs=%d, err=%v", len(docs), err)
		if err == nil && len(docs) > 0 {
			ragContext := o.rag.BuildContext(docs)
			systemContent = systemContent + "\n\nUse the following information to answer the question:\n\n" + ragContext
		}

	case models.IntentMCP:
		// TODO: Phase 2 MCP — เมื่อมี SQL Server จริง
		log.Printf("MCP Intent detected — MCP not connected yet, falling back to Direct")

	case models.IntentDirect:
		// ตอบตรงๆ ไม่ต้องค้นหาอะไร
		log.Printf("Direct Intent — answering without RAG/MCP")
	}

	// รวม system prompt + messages
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
