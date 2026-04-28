package ports

import "context"

// EmbedderPort คือ "สัญญา" ว่า Embedder ใดๆ ต้องทำได้
// ไม่ว่าจะเป็น Nomic, OpenAI, หรือ อื่นๆ
type EmbedderPort interface {
	// Embed แปลงข้อความเป็น Vector
	Embed(ctx context.Context, text string) ([]float32, error)

	// Dimensions บอกขนาดของ Vector ที่ผลิตออกมา
	// nomic-embed-text = 768 dimensions
	Dimensions() int
}
