package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/enterprise-ai/orchestrator/internal/core/intent"
	"github.com/enterprise-ai/orchestrator/internal/core/rag"
	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/enterprise-ai/orchestrator/internal/domain/ports"
)

const historyLimit = 10 // Sliding Window — ดึงแค่ 10 messages ล่าสุด

type Orchestrator struct {
	llm          ports.LLMPort
	rag          *rag.RAGEngine
	session      ports.SessionPort
	classifier   *intent.Classifier
	systemPrompt string
}

func New(llm ports.LLMPort, rag *rag.RAGEngine, session ports.SessionPort, systemPrompt string) *Orchestrator {
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = "You are a helpful enterprise AI assistant. You must always respond in Thai language only."
	}
	return &Orchestrator{
		llm:          llm,
		rag:          rag,
		session:      session,
		classifier:   intent.New(),
		systemPrompt: systemPrompt,
	}
}

func (o *Orchestrator) Chat(ctx context.Context, req models.ChatRequest) (string, models.Intent, error) {
	if len(req.Messages) == 0 {
		return "", models.IntentDirect, fmt.Errorf("messages is required")
	}

	lastMsg := req.Messages[len(req.Messages)-1].Content

	// ดึง Session History (Sliding Window)
	var historyMessages []models.ChatMessage
	sessionID, _ := ctx.Value("session_id").(string)
	if sessionID != "" && o.session != nil {
		history, err := o.session.GetHistory(ctx, sessionID, historyLimit)
		if err == nil {
			for _, h := range history {
				historyMessages = append(historyMessages, models.ChatMessage{
					Role:    h.Role,
					Content: h.Content,
				})
			}
		}
	}

	// Intent Classification
	intentResult := o.classifier.Classify(lastMsg)
	slog.Info("intent classified",
		"query_len", len([]rune(lastMsg)),
		"intent", intentResult.Intent,
		"confidence", intentResult.Confidence,
	)

	systemContent := o.systemPrompt

	switch intentResult.Intent {
	case models.IntentRAG:
		docs, err := o.rag.Search(ctx, lastMsg, 3)
		slog.Info("rag search", "docs", len(docs), "error", err)
		if err == nil && len(docs) > 0 {
			ragContext := o.rag.BuildContext(docs)
			systemContent = systemContent + "\n\nUse the following information to answer:\n\n" + ragContext
		}
	case models.IntentMCP:
		slog.Info("mcp intent - not connected yet")
	case models.IntentDirect:
		slog.Info("direct intent")
	}

	// รวม system + history + คำถามใหม่
	messages := []models.ChatMessage{{Role: "system", Content: systemContent}}
	messages = append(messages, historyMessages...)
	messages = append(messages, req.Messages[len(req.Messages)-1])

	result, err := o.llm.Chat(ctx, messages)
	if err != nil {
		return "", intentResult.Intent, fmt.Errorf("llm chat: %w", err)
	}

	// บันทึก messages ลง DB
	if sessionID != "" && o.session != nil {
		userMsg := models.Message{
			SessionID: sessionID,
			Role:      "user",
			Content:   lastMsg,
			Metadata:  map[string]any{"intent": string(intentResult.Intent)},
		}
		assistantMsg := models.Message{
			SessionID: sessionID,
			Role:      "assistant",
			Content:   result,
		}
		go o.session.SaveMessage(context.Background(), userMsg)
		go o.session.SaveMessage(context.Background(), assistantMsg)
	}

	return result, intentResult.Intent, nil
}

func (o *Orchestrator) ChatStream(ctx context.Context, req models.ChatRequest, onChunk func(string)) (models.Intent, error) {
	slog.Info("chat stream started", "messages", len(req.Messages))

	if len(req.Messages) == 0 {
		return models.IntentDirect, fmt.Errorf("messages is required")
	}

	lastMsg := req.Messages[len(req.Messages)-1].Content

	// ดึง Session History
	var historyMessages []models.ChatMessage
	sessionID, _ := ctx.Value("session_id").(string)
	if sessionID != "" && o.session != nil {
		history, err := o.session.GetHistory(ctx, sessionID, historyLimit)
		if err == nil {
			for _, h := range history {
				historyMessages = append(historyMessages, models.ChatMessage{
					Role:    h.Role,
					Content: h.Content,
				})
			}
		}
	}

	// Intent Classification
	intentResult := o.classifier.Classify(lastMsg)
	systemContent := o.systemPrompt

	if intentResult.Intent == models.IntentRAG {
		docs, err := o.rag.Search(ctx, lastMsg, 3)
		if err == nil && len(docs) > 0 {
			ragContext := o.rag.BuildContext(docs)
			systemContent = systemContent + "\n\nUse the following information to answer:\n\n" + ragContext
		}
	}

	// รวม system + history + คำถามใหม่
	messages := []models.ChatMessage{{Role: "system", Content: systemContent}}
	messages = append(messages, historyMessages...)
	messages = append(messages, req.Messages[len(req.Messages)-1])

	// Collect full response สำหรับบันทึก
	var fullResponse strings.Builder
	err := o.llm.ChatStream(ctx, messages, func(chunk string) {
		fullResponse.WriteString(chunk)
		onChunk(chunk)
	})

	// บันทึก messages ลง DB
	if sessionID != "" && o.session != nil && err == nil {
		userMsg := models.Message{
			SessionID: sessionID,
			Role:      "user",
			Content:   lastMsg,
			Metadata:  map[string]any{"intent": string(intentResult.Intent)},
		}
		assistantMsg := models.Message{
			SessionID: sessionID,
			Role:      "assistant",
			Content:   fullResponse.String(),
		}
		go o.session.SaveMessage(context.Background(), userMsg)
		go o.session.SaveMessage(context.Background(), assistantMsg)
	}

	slog.Info("chat stream done", "error", err)
	return intentResult.Intent, err
}

func (o *Orchestrator) HealthCheck(ctx context.Context) error {
	return o.llm.HealthCheck(ctx)
}

func (o *Orchestrator) EmbedderCheck(ctx context.Context) error {
	result, err := o.rag.Embedder().Embed(ctx, "health check")
	if err != nil {
		return err
	}
	if len(result) == 0 {
		return fmt.Errorf("empty embedding returned")
	}
	return nil
}
