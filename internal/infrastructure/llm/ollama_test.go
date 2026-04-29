package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

func TestOllamaAdapterChat_ReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "model not loaded", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	adapter := NewOllamaAdapter(server.URL, "test-model")
	_, err := adapter.Chat(context.Background(), []models.ChatMessage{{Role: "user", Content: "hello"}})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "503") || !strings.Contains(err.Error(), "model not loaded") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOllamaAdapterChatStream_ReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "stream unavailable", http.StatusBadGateway)
	}))
	defer server.Close()

	adapter := NewOllamaAdapter(server.URL, "test-model")
	err := adapter.ChatStream(context.Background(), []models.ChatMessage{{Role: "user", Content: "hello"}}, func(string) {
		t.Fatalf("unexpected chunk")
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "502") || !strings.Contains(err.Error(), "stream unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOllamaAdapterChat_DecodesSuccessfulResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"สวัสดี"},"done":true}`))
	}))
	defer server.Close()

	adapter := NewOllamaAdapter(server.URL, "test-model")
	result, err := adapter.Chat(context.Background(), []models.ChatMessage{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if result != "สวัสดี" {
		t.Fatalf("expected response content, got %q", result)
	}
}
