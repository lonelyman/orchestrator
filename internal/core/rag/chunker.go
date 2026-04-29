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
		ChunkSize:    200, // ลดจาก 500 → 200
		ChunkOverlap: 20,  // ลดจาก 50 → 20
	}
}

// Chunk ตัดข้อความยาวเป็น chunks เล็กๆ
func Chunk(text string, opts ChunkOptions) []string {
	if opts.ChunkSize <= 0 {
		opts.ChunkSize = DefaultChunkOptions().ChunkSize
	}
	if opts.ChunkOverlap < 0 {
		opts.ChunkOverlap = 0
	}
	if opts.ChunkOverlap >= opts.ChunkSize {
		opts.ChunkOverlap = opts.ChunkSize / 2
	}

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
		step := opts.ChunkSize - opts.ChunkOverlap
		if step <= 0 {
			break
		}
		start += step
		if start >= len(words) {
			break
		}
	}

	return chunks
}
