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

type ChatHandler struct {
	orch        *orchestrator.Orchestrator
	chatTimeout time.Duration
}

func NewChatHandler(orch *orchestrator.Orchestrator) *ChatHandler {
	return NewChatHandlerWithTimeout(orch, 120*time.Second)
}

func NewChatHandlerWithTimeout(orch *orchestrator.Orchestrator, chatTimeout time.Duration) *ChatHandler {
	if chatTimeout <= 0 {
		chatTimeout = 120 * time.Second
	}
	return &ChatHandler{orch: orch, chatTimeout: chatTimeout}
}

func (h *ChatHandler) Models(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"object": "list",
		"data": []fiber.Map{
			{
				"id":       "qwen2.5:7b",
				"object":   "model",
				"owned_by": "ollama",
			},
		},
	})
}

func (h *ChatHandler) Completions(c fiber.Ctx) error {
	var req models.ChatRequest
	if err := c.Bind().JSON(&req); err != nil {
		return Fail(c, 400, "invalid request", "BAD_REQUEST")
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

		streamCtx := context.WithValue(context.Background(), "session_id", sessionID)
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
	ctx := context.WithValue(context.Background(), "session_id", sessionID)
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
