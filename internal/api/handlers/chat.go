package handlers

import (
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

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	result, err := h.orch.Chat(ctx, req)
	if err != nil {
		return Fail(c, 500, err.Error(), "LLM_ERROR")
	}

	return OK(c, models.ChatResponse{
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
