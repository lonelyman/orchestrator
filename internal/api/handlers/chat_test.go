package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

func TestChatHandlerValidateRequest_DefaultsModelAndNormalizesRole(t *testing.T) {
	handler := NewChatHandlerWithTimeout(nil, 120*time.Second, "qwen2.5:7b", "ollama")
	req := models.ChatRequest{
		Messages: []models.ChatMessage{
			{Role: "USER", Content: "สวัสดี"},
		},
	}

	if err := handler.validateRequest(&req); err != nil {
		t.Fatalf("validateRequest() error = %v", err)
	}
	if req.Model != "qwen2.5:7b" {
		t.Fatalf("expected default model, got %q", req.Model)
	}
	if req.Messages[0].Role != "user" {
		t.Fatalf("expected normalized role, got %q", req.Messages[0].Role)
	}
}

func TestChatHandlerValidateRequest_RejectsEmptyMessages(t *testing.T) {
	handler := NewChatHandlerWithTimeout(nil, 120*time.Second, "qwen2.5:7b", "ollama")
	req := models.ChatRequest{Model: "qwen2.5:7b"}

	if err := handler.validateRequest(&req); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestChatHandlerValidateRequest_RejectsUnsupportedModel(t *testing.T) {
	handler := NewChatHandlerWithTimeout(nil, 120*time.Second, "qwen2.5:7b", "ollama")
	req := models.ChatRequest{
		Model: "other-model",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	}

	if err := handler.validateRequest(&req); err == nil || !strings.Contains(err.Error(), "unsupported model") {
		t.Fatalf("expected unsupported model error, got %v", err)
	}
}

func TestChatHandlerValidateRequest_RejectsInvalidRole(t *testing.T) {
	handler := NewChatHandlerWithTimeout(nil, 120*time.Second, "qwen2.5:7b", "ollama")
	req := models.ChatRequest{
		Model: "qwen2.5:7b",
		Messages: []models.ChatMessage{
			{Role: "tool", Content: "hello"},
		},
	}

	if err := handler.validateRequest(&req); err == nil || !strings.Contains(err.Error(), "role") {
		t.Fatalf("expected role error, got %v", err)
	}
}

func TestChatHandlerValidateRequest_RejectsEmptyContent(t *testing.T) {
	handler := NewChatHandlerWithTimeout(nil, 120*time.Second, "qwen2.5:7b", "ollama")
	req := models.ChatRequest{
		Model: "qwen2.5:7b",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "   "},
		},
	}

	if err := handler.validateRequest(&req); err == nil || !strings.Contains(err.Error(), "content") {
		t.Fatalf("expected content error, got %v", err)
	}
}

func TestChatHandlerValidateRequest_RejectsAssistantAsLastMessage(t *testing.T) {
	handler := NewChatHandlerWithTimeout(nil, 120*time.Second, "qwen2.5:7b", "ollama")
	req := models.ChatRequest{
		Model: "qwen2.5:7b",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
		},
	}

	if err := handler.validateRequest(&req); err == nil || !strings.Contains(err.Error(), "last message") {
		t.Fatalf("expected last message error, got %v", err)
	}
}
