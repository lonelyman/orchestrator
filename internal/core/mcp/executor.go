package mcp

import (
	"context"
	"fmt"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/enterprise-ai/orchestrator/internal/domain/ports"
)

// Executor จัดการการเรียกใช้ Tools ผ่าน MCP
type Executor struct {
	adapters map[string]ports.MCPPort // key = adapter name
}

// New สร้าง Executor ใหม่
func New() *Executor {
	return &Executor{
		adapters: make(map[string]ports.MCPPort),
	}
}

// Register ลงทะเบียน MCP Adapter
func (e *Executor) Register(name string, adapter ports.MCPPort) {
	e.adapters[name] = adapter
}

// ListAllTools รวม Tools จากทุก Adapter
func (e *Executor) ListAllTools(ctx context.Context) ([]models.Tool, error) {
	var allTools []models.Tool
	for _, adapter := range e.adapters {
		tools, err := adapter.ListTools(ctx)
		if err != nil {
			continue // ข้าม adapter ที่ error
		}
		allTools = append(allTools, tools...)
	}
	return allTools, nil
}

// Execute หา Adapter ที่รองรับ Tool แล้วเรียกใช้
func (e *Executor) Execute(ctx context.Context, call models.ToolCall) (models.ToolResult, error) {
	for _, adapter := range e.adapters {
		tools, err := adapter.ListTools(ctx)
		if err != nil {
			continue
		}
		for _, tool := range tools {
			if tool.Name == call.ToolName {
				return adapter.Execute(ctx, call)
			}
		}
	}
	return models.ToolResult{}, fmt.Errorf("tool not found: %s", call.ToolName)
}
