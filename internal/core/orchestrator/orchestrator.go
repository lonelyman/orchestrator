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
const noRAGContextInstruction = "\n\nDocument search returned no relevant results. Tell the user in Thai that no information was found in uploaded documents, and do not guess from general knowledge."

type Orchestrator struct {
	llm          ports.LLMPort
	rag          *rag.RAGEngine
	session      ports.SessionPort
	classifier   *intent.Classifier
	systemPrompt string
	tools        *ToolRegistry
}

func New(llm ports.LLMPort, rag *rag.RAGEngine, session ports.SessionPort, systemPrompt string) *Orchestrator {
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = "You are a helpful enterprise AI assistant. You must always respond in Thai language only."
	}
	toolRegistry := NewToolRegistry()
	if rag != nil {
		toolRegistry.Register(NewRAGSearchTool(rag))
	}

	return &Orchestrator{
		llm:          llm,
		rag:          rag,
		session:      session,
		classifier:   intent.New(),
		systemPrompt: systemPrompt,
		tools:        toolRegistry,
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
		systemContent += "\n\nFor questions about uploaded organization documents, use the rag_search tool before answering. Answer only from tool results. If no matching documents are returned, say in Thai that no matching information was found in uploaded documents."
	case models.IntentMCP:
		slog.Info("mcp intent - not connected yet")
	case models.IntentDirect:
		slog.Info("direct intent")
	}

	// รวม system + history + คำถามใหม่
	messages := []models.ChatMessage{{Role: "system", Content: systemContent}}
	messages = append(messages, historyMessages...)
	messages = append(messages, req.Messages[len(req.Messages)-1])

	result, usedTools, err := o.runAgent(ctx, intentResult.Intent, messages)
	if err != nil {
		return "", intentResult.Intent, fmt.Errorf("llm chat: %w", err)
	}
	if intentResult.Intent == models.IntentRAG && (!usedTools || strings.TrimSpace(result) == "") {
		result, err = o.chatWithRAGContext(ctx, systemContent, historyMessages, req.Messages[len(req.Messages)-1], lastMsg)
		if err != nil {
			return "", intentResult.Intent, fmt.Errorf("llm chat: %w", err)
		}
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
		if err := o.session.SaveMessages(ctx, []models.Message{userMsg, assistantMsg}); err != nil {
			slog.Error("save session messages failed", "error", err, "session_id", sessionID)
		}
	}

	return result, intentResult.Intent, nil
}

func (o *Orchestrator) runAgent(ctx context.Context, intentType models.Intent, messages []models.ChatMessage) (string, bool, error) {
	tools := o.agentTools(intentType)
	if len(tools) == 0 {
		content, err := o.llm.Chat(ctx, messages)
		return content, false, err
	}

	workingMessages := append([]models.ChatMessage(nil), messages...)
	usedTools := false
	for i := 0; i < agentMaxIterations; i++ {
		resp, err := o.llm.ChatWithTools(ctx, models.LLMChatRequest{
			Messages:   workingMessages,
			Tools:      tools,
			ToolChoice: "auto",
		})
		if err != nil {
			return "", usedTools, err
		}
		if len(resp.ToolCalls) == 0 {
			return resp.Content, usedTools, nil
		}

		usedTools = true
		slog.Info("agent tool calls requested", "iteration", i+1, "count", len(resp.ToolCalls))
		if strings.TrimSpace(resp.Content) != "" {
			workingMessages = append(workingMessages, models.ChatMessage{Role: "assistant", Content: resp.Content})
		}
		for _, call := range resp.ToolCalls {
			result, execErr := o.tools.Execute(ctx, call)
			if execErr != nil {
				slog.Warn("agent tool call failed", "tool", call.ToolName, "error", execErr)
			}
			workingMessages = append(workingMessages, models.ChatMessage{
				Role:    "user",
				Content: formatToolResultForLLM(result),
			})
		}
	}

	final, err := o.llm.Chat(ctx, append(workingMessages, models.ChatMessage{
		Role:    "user",
		Content: "Summarize the available tool results and answer the original user question in Thai. Do not call more tools.",
	}))
	if err != nil {
		return "", usedTools, err
	}
	return final, usedTools, nil
}

func (o *Orchestrator) agentTools(intentType models.Intent) []models.Tool {
	if o.tools == nil {
		return nil
	}
	switch intentType {
	case models.IntentRAG:
		return o.tools.Definitions()
	default:
		return nil
	}
}

func (o *Orchestrator) chatWithRAGContext(ctx context.Context, systemContent string, historyMessages []models.ChatMessage, userMessage models.ChatMessage, query string) (string, error) {
	if o.rag == nil {
		return o.llm.Chat(ctx, append(append([]models.ChatMessage{{Role: "system", Content: systemContent}}, historyMessages...), userMessage))
	}

	docs, err := o.rag.Search(ctx, query, defaultRAGSearchLimit)
	slog.Info("rag fallback search", "docs", len(docs), "error", err)
	if err == nil && len(docs) > 0 {
		systemContent = systemContent + "\n\nUse the following information to answer:\n\n" + o.rag.BuildContext(docs)
	} else {
		systemContent += noRAGContextInstruction
	}

	messages := []models.ChatMessage{{Role: "system", Content: systemContent}}
	messages = append(messages, historyMessages...)
	messages = append(messages, userMessage)
	return o.llm.Chat(ctx, messages)
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
		} else {
			systemContent += noRAGContextInstruction
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
		if saveErr := o.session.SaveMessages(ctx, []models.Message{userMsg, assistantMsg}); saveErr != nil {
			slog.Error("save session messages failed", "error", saveErr, "session_id", sessionID)
		}
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
