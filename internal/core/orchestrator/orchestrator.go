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

	// ดึงคำถามล่าสุด — กรอง WebUI format ออก
	lastMsg := req.Messages[len(req.Messages)-1].Content

	// ถ้า WebUI ส่งมาเป็น chat_history format ให้ดึงคำถามล่าสุดออก
	if strings.Contains(lastMsg, "<chat_history>") {
		lines := strings.Split(lastMsg, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if strings.HasPrefix(line, "USER:") {
				lastMsg = strings.TrimSpace(strings.TrimPrefix(line, "USER:"))
				break
			}
		}
	}

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
	log.Printf("ChatStream called: messages=%d", len(req.Messages))

	if len(req.Messages) == 0 {
		return fmt.Errorf("messages is required")
	}

	// ดึงคำถามล่าสุด
	lastMsg := req.Messages[len(req.Messages)-1].Content

	// Intent Classification
	intentResult := o.classifier.Classify(lastMsg)
	log.Printf("ChatStream Intent: intent=%s", intentResult.Intent)

	systemContent := o.systemPrompt

	// RAG Search เหมือน Chat
	if intentResult.Intent == models.IntentRAG {
		docs, err := o.rag.Search(ctx, lastMsg, 3)
		if err == nil && len(docs) > 0 {
			ragContext := o.rag.BuildContext(docs)
			systemContent = systemContent + "\n\nUse the following information to answer the question:\n\n" + ragContext
		}
	}

	messages := append([]models.ChatMessage{
		{Role: "system", Content: systemContent},
	}, req.Messages...)

	err := o.llm.ChatStream(ctx, messages, func(chunk string) {
		log.Printf("Chunk received: %q", chunk)
		onChunk(chunk)
	})
	log.Printf("ChatStream done: err=%v", err)
	return err
}

// HealthCheck ตรวจสอบว่าระบบพร้อมใช้งาน
func (o *Orchestrator) HealthCheck(ctx context.Context) error {
	return o.llm.HealthCheck(ctx)
}

// EmbedderCheck ตรวจสอบว่า Embedder พร้อมใช้งาน
func (o *Orchestrator) EmbedderCheck(ctx context.Context) error {
	_, err := o.rag.Embedder().Embed(ctx, "test")
	return err
}
