package ports

import (
	"context"
	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

// VectorPort คือ "สัญญา" ว่า Vector DB ใดๆ ต้องทำได้
// ตอนนี้ใช้ pgvector แต่เปลี่ยนได้ในอนาคต
type VectorPort interface {
	// Store บันทึก Document พร้อม Vector ลง DB
	Store(ctx context.Context, doc models.Document) error

	// Search ค้นหา Document ที่ใกล้เคียงกับ Vector ที่ให้มา
	Search(ctx context.Context, embedding []float32, limit int) ([]models.Document, error)

	// Delete ลบ Document ออกจาก DB
	Delete(ctx context.Context, id string) error
}