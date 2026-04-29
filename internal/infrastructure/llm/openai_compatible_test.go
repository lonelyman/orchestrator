package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

func TestOpenAICompatibleAdapterChat_DecodesSuccessfulResponse(t *testing.T) {
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"สวัสดี"}}]}`))
	}))
	defer server.Close()

	adapter := NewOpenAICompatibleAdapter(server.URL, "test-model", "test-key")
	result, err := adapter.Chat(context.Background(), []models.ChatMessage{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if result != "สวัสดี" {
		t.Fatalf("expected response content, got %q", result)
	}
	if authHeader != "Bearer test-key" {
		t.Fatalf("expected bearer auth header, got %q", authHeader)
	}
}

func TestOpenAICompatibleAdapterChatStream_DecodesSSE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	adapter := NewOpenAICompatibleAdapter(server.URL, "test-model", "")
	var chunks []string
	err := adapter.ChatStream(context.Background(), []models.ChatMessage{{Role: "user", Content: "hello"}}, func(chunk string) {
		chunks = append(chunks, chunk)
	})
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}
	if got := strings.Join(chunks, ""); got != "hello world" {
		t.Fatalf("expected streamed content, got %q", got)
	}
}

func TestOpenAICompatibleAdapterChat_ReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "model unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	adapter := NewOpenAICompatibleAdapter(server.URL, "test-model", "")
	_, err := adapter.Chat(context.Background(), []models.ChatMessage{{Role: "user", Content: "hello"}})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "503") || !strings.Contains(err.Error(), "model unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}
