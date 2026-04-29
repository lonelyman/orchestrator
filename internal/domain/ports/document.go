package ports

import (
	"context"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

// DocumentPort คือ interface สำหรับจัดการเอกสารที่ ingest เข้า RAG แล้ว
type DocumentPort interface {
	// ListDocuments แสดงเอกสารแบบ grouped by source พร้อมจำนวน chunks
	ListDocuments(ctx context.Context) ([]models.DocumentSummary, error)

	// DeleteDocumentSource ลบ chunks ทั้งหมดของ source นั้น และคืนจำนวน chunks ที่ลบ
	DeleteDocumentSource(ctx context.Context, source string) (int64, error)
}
