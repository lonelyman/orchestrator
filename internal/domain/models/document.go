package models

import "time"

// Document คือเอกสารที่ถูก Ingest เข้าระบบ RAG
type Document struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`   // เนื้อหาของ chunk
	Source    string    `json:"source"`    // ชื่อไฟล์หรือแหล่งที่มา
	Embedding []float32 `json:"embedding"` // Vector ที่ได้จาก Embedder
	CreatedAt time.Time `json:"created_at"`
}

// SearchResult คือผลลัพธ์จากการค้นหา Vector
type SearchResult struct {
	Document Document `json:"document"`
	Score    float64  `json:"score"` // ความใกล้เคียง 0-1
}
