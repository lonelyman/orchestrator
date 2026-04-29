package llm

import (
	"context"
	"encoding/json"
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

func TestOllamaAdapterChatWithTools_SendsToolsAndParsesToolCalls(t *testing.T) {
	var reqBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"message":{
				"role":"assistant",
				"tool_calls":[{
					"function":{"name":"web_search","arguments":{"query":"latest news"}}
				}]
			},
			"done":true
		}`))
	}))
	defer server.Close()

	adapter := NewOllamaAdapter(server.URL, "test-model")
	resp, err := adapter.ChatWithTools(context.Background(), models.LLMChatRequest{
		Messages: []models.ChatMessage{{Role: "user", Content: "search"}},
		Tools: []models.Tool{{
			Name:        "web_search",
			Description: "Search current web results",
			Parameters: map[string]models.ToolParam{
				"query": {Type: "string", Description: "search query", Required: true},
			},
		}},
	})
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	tools, ok := reqBody["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("expected one tool in request, got %#v", reqBody["tools"])
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected one tool call, got %+v", resp.ToolCalls)
	}
	if resp.ToolCalls[0].ToolName != "web_search" {
		t.Fatalf("unexpected tool call: %+v", resp.ToolCalls[0])
	}
	if resp.ToolCalls[0].Arguments["query"] != "latest news" {
		t.Fatalf("unexpected tool args: %+v", resp.ToolCalls[0].Arguments)
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
