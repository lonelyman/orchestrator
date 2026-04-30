package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/core/rag"
	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

const (
	agentMaxIterations      = 3
	agentToolTimeout        = 15 * time.Second
	agentMaxToolResultRunes = 8_000
	ragSearchToolName       = "rag_search"
	defaultRAGSearchLimit   = 3
	maxRAGSearchLimit       = 5
)

type ToolExecutor func(ctx context.Context, call models.ToolCall) (models.ToolResult, error)

type RegisteredTool struct {
	Definition models.Tool
	Execute    ToolExecutor
	Timeout    time.Duration
}

type ToolRegistry struct {
	tools map[string]RegisteredTool
	order []string
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]RegisteredTool)}
}

func (r *ToolRegistry) Register(tool RegisteredTool) {
	name := strings.TrimSpace(tool.Definition.Name)
	if name == "" || tool.Execute == nil {
		return
	}
	tool.Definition.Name = name
	if tool.Timeout <= 0 {
		tool.Timeout = agentToolTimeout
	}
	if _, exists := r.tools[name]; !exists {
		r.order = append(r.order, name)
	}
	r.tools[name] = tool
}

func (r *ToolRegistry) Definitions() []models.Tool {
	if r == nil || len(r.tools) == 0 {
		return nil
	}
	defs := make([]models.Tool, 0, len(r.order))
	for _, name := range r.order {
		defs = append(defs, r.tools[name].Definition)
	}
	return defs
}

func (r *ToolRegistry) Execute(ctx context.Context, call models.ToolCall) (models.ToolResult, error) {
	if r == nil {
		return models.ToolResult{}, fmt.Errorf("tool registry not configured")
	}
	name := strings.TrimSpace(call.ToolName)
	tool, ok := r.tools[name]
	if !ok {
		return models.ToolResult{ToolName: name, Error: "tool not registered"}, fmt.Errorf("tool not registered: %s", name)
	}

	timeout := tool.Timeout
	if timeout <= 0 {
		timeout = agentToolTimeout
	}
	toolCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	result, err := tool.Execute(toolCtx, call)
	slog.Info("agent tool executed",
		"tool", name,
		"latency_ms", time.Since(start).Milliseconds(),
		"error", err,
	)
	if err != nil {
		return models.ToolResult{ToolName: name, Error: err.Error()}, err
	}
	result.ToolName = name
	return result, nil
}

func NewRAGSearchTool(engine *rag.RAGEngine) RegisteredTool {
	return RegisteredTool{
		Definition: models.Tool{
			Name:        ragSearchToolName,
			Description: "Search uploaded organization documents. Use this before answering questions about policies, procedures, announcements, benefits, leave, OT, or internal documents.",
			Parameters: map[string]models.ToolParam{
				"query": {
					Type:        "string",
					Description: "The document search query in the user's language.",
					Required:    true,
				},
				"limit": {
					Type:        "integer",
					Description: "Maximum number of matching chunks to return. Default 3, maximum 5.",
					Required:    false,
				},
			},
		},
		Timeout: agentToolTimeout,
		Execute: func(ctx context.Context, call models.ToolCall) (models.ToolResult, error) {
			if engine == nil {
				return models.ToolResult{}, fmt.Errorf("rag engine not configured")
			}
			query, _ := call.Arguments["query"].(string)
			query = strings.TrimSpace(query)
			if query == "" {
				return models.ToolResult{}, fmt.Errorf("query is required")
			}

			limit := parseToolLimit(call.Arguments["limit"], defaultRAGSearchLimit, maxRAGSearchLimit)
			docs, err := engine.Search(ctx, query, limit)
			if err != nil {
				return models.ToolResult{}, err
			}

			items := make([]map[string]any, 0, len(docs))
			for _, doc := range docs {
				items = append(items, map[string]any{
					"source":  doc.Source,
					"content": doc.Content,
				})
			}
			return models.ToolResult{
				ToolName: ragSearchToolName,
				Data: map[string]any{
					"query":     query,
					"documents": items,
					"count":     len(items),
				},
			}, nil
		},
	}
}

func parseToolLimit(value any, defaultLimit, maxLimit int) int {
	limit := defaultLimit
	switch v := value.(type) {
	case int:
		limit = v
	case int64:
		limit = int(v)
	case float64:
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			limit = int(v)
		}
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			limit = int(parsed)
		}
	}
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

func formatToolResultForLLM(result models.ToolResult) string {
	payload, err := json.Marshal(result)
	if err != nil {
		payload = []byte(fmt.Sprintf(`{"tool_name":%q,"error":"marshal tool result failed"}`, result.ToolName))
	}

	text := string(payload)
	runes := []rune(text)
	if len(runes) > agentMaxToolResultRunes {
		text = string(runes[:agentMaxToolResultRunes]) + "...[truncated]"
	}
	return "Tool result. Use this result to answer the user in Thai. If documents is empty, say no matching information was found in uploaded documents.\n\n" + text
}
