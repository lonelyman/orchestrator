package ports

import (
	"context"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

// MCPPort คือ "สัญญา" ว่า MCP Bridge ใดๆ ต้องทำได้
// ไม่ว่าจะเป็น SQL Server, MySQL, หรือ API อื่นๆ
type MCPPort interface {
	// ListTools แสดงรายการ Tools ที่ใช้ได้
	ListTools(ctx context.Context) ([]models.Tool, error)

	// Execute เรียกใช้ Tool พร้อม arguments
	// Read-only เท่านั้น ห้าม INSERT/UPDATE/DELETE
	Execute(ctx context.Context, call models.ToolCall) (models.ToolResult, error)

	// Ping ตรวจสอบการเชื่อมต่อ
	Ping(ctx context.Context) error
}
