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

func TestOpenAICompatibleAdapterChatWithTools_SendsToolsAndParsesToolCalls(t *testing.T) {
	var reqBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices":[{
				"finish_reason":"tool_calls",
				"message":{
					"role":"assistant",
					"tool_calls":[{
						"id":"call_1",
						"type":"function",
						"function":{"name":"web_search","arguments":"{\"query\":\"latest news\"}"}
					}]
				}
			}],
			"usage":{"prompt_tokens":10,"completion_tokens":3,"total_tokens":13}
		}`))
	}))
	defer server.Close()

	adapter := NewOpenAICompatibleAdapter(server.URL, "test-model", "")
	resp, err := adapter.ChatWithTools(context.Background(), models.LLMChatRequest{
		Messages: []models.ChatMessage{{Role: "user", Content: "search"}},
		Tools: []models.Tool{{
			Name:        "web_search",
			Description: "Search current web results",
			Parameters: map[string]models.ToolParam{
				"query": {Type: "string", Description: "search query", Required: true},
			},
		}},
		ToolChoice: "auto",
	})
	if err != nil {
		t.Fatalf("ChatWithTools() error = %v", err)
	}
	if reqBody["tool_choice"] != "auto" {
		t.Fatalf("expected tool_choice auto, got %v", reqBody["tool_choice"])
	}
	tools, ok := reqBody["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("expected one tool in request, got %#v", reqBody["tools"])
	}
	if resp.FinishReason != "tool_calls" {
		t.Fatalf("expected finish reason tool_calls, got %q", resp.FinishReason)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected one tool call, got %+v", resp.ToolCalls)
	}
	if resp.ToolCalls[0].ID != "call_1" || resp.ToolCalls[0].ToolName != "web_search" {
		t.Fatalf("unexpected tool call: %+v", resp.ToolCalls[0])
	}
	if resp.ToolCalls[0].Arguments["query"] != "latest news" {
		t.Fatalf("unexpected tool args: %+v", resp.ToolCalls[0].Arguments)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 13 {
		t.Fatalf("expected usage total 13, got %+v", resp.Usage)
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
