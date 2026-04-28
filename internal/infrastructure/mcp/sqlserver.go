package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

// SQLServerAdapter implement MCPPort สำหรับ SQL Server
// Zero-Trust: Read-only เท่านั้น
type SQLServerAdapter struct {
	db   *sql.DB
	name string
}

// NewSQLServerAdapter สร้าง SQLServerAdapter ใหม่
func NewSQLServerAdapter(db *sql.DB, name string) *SQLServerAdapter {
	return &SQLServerAdapter{db: db, name: name}
}

// ListTools แสดงรายการ Tools ที่ใช้ได้
func (s *SQLServerAdapter) ListTools(ctx context.Context) ([]models.Tool, error) {
	return []models.Tool{
		{
			Name:        fmt.Sprintf("%s_query", s.name),
			Description: fmt.Sprintf("Query data from %s database (read-only)", s.name),
			Parameters: map[string]models.ToolParam{
				"query": {
					Type:        "string",
					Description: "SQL SELECT query to execute",
					Required:    true,
				},
			},
		},
	}, nil
}

// Execute เรียกใช้ Tool — Read-only เท่านั้น
func (s *SQLServerAdapter) Execute(ctx context.Context, call models.ToolCall) (models.ToolResult, error) {
	query, ok := call.Arguments["query"].(string)
	if !ok {
		return models.ToolResult{ToolName: call.ToolName, Error: "missing query argument"}, nil
	}

	// Zero-Trust Validation — ห้าม write operations
	if err := validateReadOnly(query); err != nil {
		return models.ToolResult{
			ToolName: call.ToolName,
			Error:    err.Error(),
		}, nil
	}

	// Execute query
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return models.ToolResult{
			ToolName: call.ToolName,
			Error:    fmt.Sprintf("query error: %v", err),
		}, nil
	}
	defer rows.Close()

	// แปลงผลลัพธ์เป็น []map[string]interface{}
	cols, err := rows.Columns()
	if err != nil {
		return models.ToolResult{ToolName: call.ToolName, Error: err.Error()}, nil
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	return models.ToolResult{
		ToolName: call.ToolName,
		Data:     results,
	}, nil
}

// Ping ตรวจสอบการเชื่อมต่อ
func (s *SQLServerAdapter) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// validateReadOnly ตรวจสอบว่า query เป็น SELECT เท่านั้น
func validateReadOnly(query string) error {
	upper := strings.ToUpper(strings.TrimSpace(query))

	// ต้องขึ้นต้นด้วย SELECT เท่านั้น
	if !strings.HasPrefix(upper, "SELECT") {
		return fmt.Errorf("only SELECT queries are allowed")
	}

	// ห้ามมี write keywords
	forbidden := []string{"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE", "EXEC", "EXECUTE"}
	for _, word := range forbidden {
		if strings.Contains(upper, word) {
			return fmt.Errorf("forbidden keyword: %s", word)
		}
	}

	return nil
}
