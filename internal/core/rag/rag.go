package rag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/enterprise-ai/orchestrator/internal/domain/ports"
	"github.com/google/uuid"
)

// RAGEngine จัดการ Ingest และ Search เอกสาร
type RAGEngine struct {
	embedder ports.EmbedderPort
	vector   ports.VectorPort
}

// New สร้าง RAGEngine ใหม่
func New(embedder ports.EmbedderPort, vector ports.VectorPort) *RAGEngine {
	return &RAGEngine{
		embedder: embedder,
		vector:   vector,
	}
}

// Ingest รับข้อความและบันทึกลง Vector DB
func (r *RAGEngine) Ingest(ctx context.Context, content, source string) error {
	// แปลงข้อความเป็น Vector
	embedding, err := r.embedder.Embed(ctx, content)
	if err != nil {
		return fmt.Errorf("embed content: %w", err)
	}

	// สร้าง Document
	doc := models.Document{
		ID:        uuid.New().String(),
		Content:   content,
		Source:    source,
		Embedding: embedding,
		CreatedAt: time.Now(),
	}

	// บันทึกลง Vector DB
	if err := r.vector.Store(ctx, doc); err != nil {
		return fmt.Errorf("store document: %w", err)
	}

	return nil
}

// Search ค้นหาเอกสารที่เกี่ยวข้องกับคำถาม
func (r *RAGEngine) Search(ctx context.Context, query string, limit int) ([]models.Document, error) {
	// แปลงคำถามเป็น Vector
	embedding, err := r.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	// ค้นหาใน Vector DB
	docs, err := r.vector.Search(ctx, embedding, limit)
	if err != nil {
		return nil, fmt.Errorf("search vector: %w", err)
	}

	return docs, nil
}

// BuildContext รวมเอกสารที่เจอเป็น context string สำหรับส่งให้ LLM
func (r *RAGEngine) BuildContext(docs []models.Document) string {
	if len(docs) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Relevant information from organization documents:\n\n")

	for i, doc := range docs {
		sb.WriteString(fmt.Sprintf("[%d] Source: %s\n%s\n\n", i+1, doc.Source, doc.Content))
	}

	return sb.String()
}

// Embedder คืน EmbedderPort
func (r *RAGEngine) Embedder() ports.EmbedderPort {
	return r.embedder
}
