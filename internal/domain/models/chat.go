package models

import "time"

// ChatMessage คือ 1 ข้อความในการสนทนา
type ChatMessage struct {
	Role    string `json:"role"`    // user, assistant, system
	Content string `json:"content"`
}

// ChatRequest - OpenAI-compatible format
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

// ChatResponse - OpenAI-compatible format
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// Session เก็บประวัติการสนทนาของแต่ละ user
type Session struct {
	ID        string        `json:"id"`
	UserID    string        `json:"user_id"`
	History   []ChatMessage `json:"history"`
	CreatedAt time.Time     `json:"created_at"`
}