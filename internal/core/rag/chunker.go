package rag

import "strings"

// ChunkOptions กำหนดขนาดของ chunk
type ChunkOptions struct {
	ChunkSize    int // จำนวนคำต่อ chunk
	ChunkOverlap int // จำนวนคำที่ overlap กัน
}

// DefaultChunkOptions ค่า default ที่เหมาะสม
func DefaultChunkOptions() ChunkOptions {
	return ChunkOptions{
		ChunkSize:    500,
		ChunkOverlap: 50,
	}
}

// Chunk ตัดข้อความยาวเป็น chunks เล็กๆ
func Chunk(text string, opts ChunkOptions) []string {
	// แบ่งเป็นคำ
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var chunks []string
	start := 0

	for start < len(words) {
		end := start + opts.ChunkSize
		if end > len(words) {
			end = len(words)
		}

		chunk := strings.Join(words[start:end], " ")
		chunks = append(chunks, chunk)

		// เลื่อน start โดย overlap
		start += opts.ChunkSize - opts.ChunkOverlap
		if start >= len(words) {
			break
		}
	}

	return chunks
}
