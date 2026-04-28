package models

// Tool คือเครื่องมือที่ AI สามารถเรียกใช้ได้
type Tool struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Parameters  map[string]ToolParam `json:"parameters"`
}

// ToolParam คือ parameter ของ Tool
type ToolParam struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ToolCall คือการเรียกใช้ Tool จาก AI
type ToolCall struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult คือผลลัพธ์จากการเรียกใช้ Tool
type ToolResult struct {
	ToolName string      `json:"tool_name"`
	Data     interface{} `json:"data"`
	Error    string      `json:"error,omitempty"`
}
