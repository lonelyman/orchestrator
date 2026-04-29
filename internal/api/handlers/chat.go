package handlers

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/api/middleware"
	"github.com/enterprise-ai/orchestrator/internal/core/orchestrator"
	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/gofiber/fiber/v3"
)

const (
	maxChatMessages       = 64
	maxChatContentRunes   = 20_000
	maxChatTotalRunes     = 60_000
	defaultChatModel      = "qwen2.5:7b"
	defaultChatOwner      = "ollama"
	chatValidationErrCode = "BAD_REQUEST"
)

type ChatHandler struct {
	orch        *orchestrator.Orchestrator
	chatTimeout time.Duration
	model       string
	ownedBy     string
}

func NewChatHandler(orch *orchestrator.Orchestrator) *ChatHandler {
	return NewChatHandlerWithTimeout(orch, 120*time.Second, defaultChatModel, defaultChatOwner)
}

func NewChatHandlerWithTimeout(orch *orchestrator.Orchestrator, chatTimeout time.Duration, modelAndOwner ...string) *ChatHandler {
	if chatTimeout <= 0 {
		chatTimeout = 120 * time.Second
	}
	model := defaultChatModel
	ownedBy := defaultChatOwner
	if len(modelAndOwner) > 0 && strings.TrimSpace(modelAndOwner[0]) != "" {
		model = modelAndOwner[0]
	}
	if len(modelAndOwner) > 1 && strings.TrimSpace(modelAndOwner[1]) != "" {
		ownedBy = modelAndOwner[1]
	}
	return &ChatHandler{orch: orch, chatTimeout: chatTimeout, model: model, ownedBy: ownedBy}
}

func (h *ChatHandler) Models(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"object": "list",
		"data": []fiber.Map{
			{
				"id":       h.model,
				"object":   "model",
				"owned_by": h.ownedBy,
			},
		},
	})
}

func (h *ChatHandler) Completions(c fiber.Ctx) error {
	var req models.ChatRequest
	if err := c.Bind().JSON(&req); err != nil {
		return Fail(c, 400, "invalid request", "BAD_REQUEST")
	}
	if err := h.validateRequest(&req); err != nil {
		return Fail(c, 400, err.Error(), chatValidationErrCode)
	}

	// บันทึก query สำหรับ Audit Log
	if len(req.Messages) > 0 {
		lastMsg := req.Messages[len(req.Messages)-1].Content
		middleware.SetAuditQuery(c, lastMsg)
	}

	// ดึง session_id จาก middleware
	sessionID, _ := c.Locals("session_id").(string)

	// Stream mode
	if req.Stream {
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")

		streamCtx := context.WithValue(c.Context(), "session_id", sessionID)
		streamCtx, streamCancel := context.WithTimeout(streamCtx, h.chatTimeout)

		// Collect full response สำหรับ Audit Log
		var fullResponse strings.Builder

		return c.SendStreamWriter(func(w *bufio.Writer) {
			defer streamCancel()
			intentType, _ := h.orch.ChatStream(streamCtx, req, func(chunk string) {
				fullResponse.WriteString(chunk)
				id := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
				data := fmt.Sprintf(`{"id":"%s","object":"chat.completion.chunk","model":"%s","choices":[{"delta":{"content":%q},"index":0}]}`,
					id, req.Model, chunk)
				fmt.Fprintf(w, "data: %s\n\n", data)
				w.Flush()
			})
			middleware.SetAuditIntent(c, string(intentType))
			middleware.SetAuditResponse(c, fullResponse.String())
			fmt.Fprintf(w, "data: [DONE]\n\n")
			w.Flush()
		})
	}

	// Non-stream mode
	ctx := context.WithValue(c.Context(), "session_id", sessionID)
	ctx, cancel := context.WithTimeout(ctx, h.chatTimeout)
	defer cancel()

	result, intentType, err := h.orch.Chat(ctx, req)
	if err != nil {
		return Fail(c, 500, err.Error(), "LLM_ERROR")
	}

	middleware.SetAuditIntent(c, string(intentType))
	middleware.SetAuditResponse(c, result)

	return c.JSON(models.ChatResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []models.Choice{
			{
				Index:        0,
				Message:      models.ChatMessage{Role: "assistant", Content: result},
				FinishReason: "stop",
			},
		},
	})
}

func (h *ChatHandler) validateRequest(req *models.ChatRequest) error {
	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		req.Model = h.model
	}
	if req.Model != h.model {
		return fmt.Errorf("unsupported model: %s", req.Model)
	}
	if len(req.Messages) == 0 {
		return fmt.Errorf("messages is required")
	}
	if len(req.Messages) > maxChatMessages {
		return fmt.Errorf("messages must contain at most %d items", maxChatMessages)
	}

	totalRunes := 0
	for i := range req.Messages {
		msg := &req.Messages[i]
		msg.Role = strings.ToLower(strings.TrimSpace(msg.Role))
		if !isSupportedChatRole(msg.Role) {
			return fmt.Errorf("messages[%d].role must be one of: system, user, assistant", i)
		}
		contentRunes := len([]rune(msg.Content))
		if strings.TrimSpace(msg.Content) == "" {
			return fmt.Errorf("messages[%d].content is required", i)
		}
		if contentRunes > maxChatContentRunes {
			return fmt.Errorf("messages[%d].content must be at most %d characters", i, maxChatContentRunes)
		}
		totalRunes += contentRunes
	}
	if totalRunes > maxChatTotalRunes {
		return fmt.Errorf("messages total content must be at most %d characters", maxChatTotalRunes)
	}
	if req.Messages[len(req.Messages)-1].Role != "user" {
		return fmt.Errorf("last message role must be user")
	}

	return nil
}

func isSupportedChatRole(role string) bool {
	switch role {
	case "system", "user", "assistant":
		return true
	default:
		return false
	}
}
