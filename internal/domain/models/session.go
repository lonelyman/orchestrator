package models

import "time"

// Session คือการสนทนาหนึ่งครั้ง
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IsActive  bool      `json:"is_active"`
}

// Message คือข้อความหนึ่งใน Session
type Message struct {
	ID        string         `json:"id"`
	SessionID string         `json:"session_id"`
	Role      string         `json:"role"` // user, assistant
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"` // RAG sources, intent, tokens
	CreatedAt time.Time      `json:"created_at"`
}

// SessionHistory คือประวัติการสนทนาที่ดึงมาใช้กับ LLM
type SessionHistory struct {
	Session  Session   `json:"session"`
	Messages []Message `json:"messages"`
}