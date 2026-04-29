package models

import "time"

// AuditLog บันทึกทุก request สำหรับ Compliance
type AuditLog struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Username        string    `json:"username"`
	SessionID       string    `json:"session_id"`
	Method          string    `json:"method"`
	Path            string    `json:"path"`
	Intent          string    `json:"intent,omitempty"`
	Query           string    `json:"query,omitempty"`
	ResponsePreview string    `json:"response_preview,omitempty"`
	Sources         []string  `json:"sources,omitempty"`
	LatencyMs       int64     `json:"latency_ms"`
	StatusCode      int       `json:"status_code"`
	IPAddress       string    `json:"ip_address"`
	CreatedAt       time.Time `json:"created_at"`
}

// AuditLogFilter สำหรับ query logs
type AuditLogFilter struct {
	UserID    string    `query:"user_id"`
	Intent    string    `query:"intent"`
	StartDate time.Time `query:"start_date"`
	EndDate   time.Time `query:"end_date"`
	Limit     int       `query:"limit"`
	Offset    int       `query:"offset"`
}
