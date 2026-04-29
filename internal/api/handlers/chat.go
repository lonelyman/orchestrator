package handlers

import (
	"bufio"
	"context"
	"fmt"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/core/orchestrator"
	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/gofiber/fiber/v3"
)

type ChatHandler struct {
	orch *orchestrator.Orchestrator
}

func NewChatHandler(orch *orchestrator.Orchestrator) *ChatHandler {
	return &ChatHandler{orch: orch}
}

func (h *ChatHandler) Completions(c fiber.Ctx) error {
	var req models.ChatRequest
	if err := c.Bind().JSON(&req); err != nil {
		return Fail(c, 400, "invalid request", "BAD_REQUEST")
	}

	// Stream mode
	if req.Stream {
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")

		// ใช้ Background context ที่ไม่ถูก cancel โดย Fiber
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)

		return c.SendStreamWriter(func(w *bufio.Writer) {
			defer cancel()
			h.orch.ChatStream(ctx, req, func(chunk string) {
				id := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
				data := fmt.Sprintf(`{"id":"%s","object":"chat.completion.chunk","model":"%s","choices":[{"delta":{"content":%q},"index":0}]}`,
					id, req.Model, chunk)
				fmt.Fprintf(w, "data: %s\n\n", data)
				w.Flush()
			})
			fmt.Fprintf(w, "data: [DONE]\n\n")
			w.Flush()
		})
	}

	// Non-stream mode
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	result, err := h.orch.Chat(ctx, req)
	if err != nil {
		return Fail(c, 500, err.Error(), "LLM_ERROR")
	}

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
