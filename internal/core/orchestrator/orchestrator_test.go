package orchestrator

import (
	"context"
	"strings"
	"testing"

	"github.com/enterprise-ai/orchestrator/internal/core/rag"
	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

type fakeLLM struct {
	chatWithToolsCalls []models.LLMChatRequest
	chatCalls          [][]models.ChatMessage
	toolResponses      []models.LLMChatResponse
	chatResponse       string
}

func (f *fakeLLM) Chat(_ context.Context, messages []models.ChatMessage) (string, error) {
	f.chatCalls = append(f.chatCalls, messages)
	return f.chatResponse, nil
}

func (f *fakeLLM) ChatWithTools(_ context.Context, req models.LLMChatRequest) (models.LLMChatResponse, error) {
	f.chatWithToolsCalls = append(f.chatWithToolsCalls, req)
	if len(f.toolResponses) == 0 {
		return models.LLMChatResponse{}, nil
	}
	resp := f.toolResponses[0]
	f.toolResponses = f.toolResponses[1:]
	return resp, nil
}

func (f *fakeLLM) ChatStream(_ context.Context, _ []models.ChatMessage, _ func(string)) error {
	return nil
}

func (f *fakeLLM) HealthCheck(context.Context) error {
	return nil
}

type fakeEmbedder struct{}

func (f fakeEmbedder) Embed(context.Context, string) ([]float32, error) {
	return []float32{0.1, 0.2}, nil
}

func (f fakeEmbedder) Dimensions() int {
	return 2
}

type fakeVector struct {
	docs []models.Document
}

func (f fakeVector) Store(context.Context, models.Document) error {
	return nil
}

func (f fakeVector) Search(context.Context, []float32, int) ([]models.Document, error) {
	return f.docs, nil
}

func (f fakeVector) Delete(context.Context, string) error {
	return nil
}

func TestChat_RAGIntentUsesAgentTool(t *testing.T) {
	llm := &fakeLLM{
		toolResponses: []models.LLMChatResponse{
			{
				ToolCalls: []models.ToolCall{{
					ToolName: ragSearchToolName,
					Arguments: map[string]any{
						"query": "นโยบายลา",
						"limit": float64(1),
					},
				}},
				FinishReason: "tool_calls",
			},
			{Content: "คำตอบจากเอกสาร"},
		},
	}
	engine := rag.New(fakeEmbedder{}, fakeVector{docs: []models.Document{{
		Source:  "hr.pdf",
		Content: "พนักงานลาป่วยได้ตามนโยบายบริษัท",
	}}})
	orch := New(llm, engine, nil, "ตอบภาษาไทย")

	result, intentType, err := orch.Chat(context.Background(), models.ChatRequest{
		Messages: []models.ChatMessage{{Role: "user", Content: "นโยบายลาป่วยเป็นอย่างไร"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if intentType != models.IntentRAG {
		t.Fatalf("expected RAG intent, got %s", intentType)
	}
	if result != "คำตอบจากเอกสาร" {
		t.Fatalf("unexpected result: %q", result)
	}
	if len(llm.chatWithToolsCalls) != 2 {
		t.Fatalf("expected two ChatWithTools calls, got %d", len(llm.chatWithToolsCalls))
	}
	if len(llm.chatWithToolsCalls[0].Tools) != 1 || llm.chatWithToolsCalls[0].Tools[0].Name != ragSearchToolName {
		t.Fatalf("expected rag_search tool definition, got %+v", llm.chatWithToolsCalls[0].Tools)
	}
	secondMessages := llm.chatWithToolsCalls[1].Messages
	if len(secondMessages) == 0 || !strings.Contains(secondMessages[len(secondMessages)-1].Content, "พนักงานลาป่วย") {
		t.Fatalf("expected tool result in second call, got %+v", secondMessages)
	}
}

func TestChat_RAGIntentFallsBackWhenModelDoesNotCallTool(t *testing.T) {
	llm := &fakeLLM{
		toolResponses: []models.LLMChatResponse{{Content: "ตอบโดยไม่ใช้ tool"}},
		chatResponse:  "คำตอบจาก fallback",
	}
	engine := rag.New(fakeEmbedder{}, fakeVector{docs: []models.Document{{
		Source:  "hr.pdf",
		Content: "ข้อมูลจากเอกสารที่ fallback ต้องใช้",
	}}})
	orch := New(llm, engine, nil, "ตอบภาษาไทย")

	result, _, err := orch.Chat(context.Background(), models.ChatRequest{
		Messages: []models.ChatMessage{{Role: "user", Content: "นโยบายบริษัทคืออะไร"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if result != "คำตอบจาก fallback" {
		t.Fatalf("unexpected result: %q", result)
	}
	if len(llm.chatCalls) != 1 {
		t.Fatalf("expected fallback Chat call, got %d", len(llm.chatCalls))
	}
	if !strings.Contains(llm.chatCalls[0][0].Content, "ข้อมูลจากเอกสารที่ fallback ต้องใช้") {
		t.Fatalf("expected fallback RAG context, got %q", llm.chatCalls[0][0].Content)
	}
}

type fakeWebSearch struct {
	results []models.WebSearchResult
}

func (f fakeWebSearch) Search(context.Context, string, models.WebSearchOptions) ([]models.WebSearchResult, error) {
	return f.results, nil
}

func (f fakeWebSearch) HealthCheck(context.Context) error {
	return nil
}

func TestChat_WebSearchIntentUsesWebSearchTool(t *testing.T) {
	llm := &fakeLLM{
		toolResponses: []models.LLMChatResponse{
			{
				ToolCalls: []models.ToolCall{{
					ToolName: webSearchToolName,
					Arguments: map[string]any{
						"query":       "ข่าว AI ล่าสุด",
						"max_results": float64(1),
						"topic":       "news",
					},
				}},
				FinishReason: "tool_calls",
			},
			{Content: "คำตอบจากเว็บพร้อมแหล่งที่มา"},
		},
	}
	orch := New(llm, nil, nil, "ตอบภาษาไทย")
	orch.RegisterTool(NewWebSearchTool(fakeWebSearch{results: []models.WebSearchResult{{
		Title:   "AI News",
		URL:     "https://example.com/ai",
		Snippet: "latest AI news",
		Source:  "example.com",
	}}}, 3))

	result, intentType, err := orch.Chat(context.Background(), models.ChatRequest{
		Messages: []models.ChatMessage{{Role: "user", Content: "ข่าว AI ล่าสุดคืออะไร"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if intentType != models.IntentWebSearch {
		t.Fatalf("expected web search intent, got %s", intentType)
	}
	if result != "คำตอบจากเว็บพร้อมแหล่งที่มา" {
		t.Fatalf("unexpected result: %q", result)
	}
	if len(llm.chatWithToolsCalls) != 2 {
		t.Fatalf("expected two ChatWithTools calls, got %d", len(llm.chatWithToolsCalls))
	}
	if len(llm.chatWithToolsCalls[0].Tools) != 1 || llm.chatWithToolsCalls[0].Tools[0].Name != webSearchToolName {
		t.Fatalf("expected web_search tool definition, got %+v", llm.chatWithToolsCalls[0].Tools)
	}
}

func TestChat_WebSearchIntentReturnsDisabledMessageWhenToolUnavailable(t *testing.T) {
	llm := &fakeLLM{chatResponse: "should not be used"}
	orch := New(llm, nil, nil, "ตอบภาษาไทย")

	result, intentType, err := orch.Chat(context.Background(), models.ChatRequest{
		Messages: []models.ChatMessage{{Role: "user", Content: "ข่าวล่าสุดวันนี้"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if intentType != models.IntentWebSearch {
		t.Fatalf("expected web search intent, got %s", intentType)
	}
	if !strings.Contains(result, "ยังไม่ได้เปิดใช้งาน web search") {
		t.Fatalf("unexpected result: %q", result)
	}
	if len(llm.chatCalls) != 0 || len(llm.chatWithToolsCalls) != 0 {
		t.Fatalf("expected no LLM calls, got chat=%d tool=%d", len(llm.chatCalls), len(llm.chatWithToolsCalls))
	}
}
