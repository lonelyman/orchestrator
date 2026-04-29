package handlers

import (
	"context"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/core/rag"
	"github.com/gofiber/fiber/v3"
)

type RAGHandler struct {
	ragEngine ragIngestor
}

type ragIngestor interface {
	Ingest(ctx context.Context, content, source string) error
}

func NewRAGHandler(ragEngine *rag.RAGEngine) *RAGHandler {
	return &RAGHandler{ragEngine: ragEngine}
}

func NewRAGHandlerWithIngestor(ragEngine ragIngestor) *RAGHandler {
	return &RAGHandler{ragEngine: ragEngine}
}

func (h *RAGHandler) Ingest(c fiber.Ctx) error {
	if h.ragEngine == nil {
		return Fail(c, 500, "rag engine not configured", "INGEST_ERROR")
	}

	var req struct {
		Content string `json:"content"`
		Source  string `json:"source"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return Fail(c, 400, "invalid request", "BAD_REQUEST")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := h.ragEngine.Ingest(ctx, req.Content, req.Source); err != nil {
		return Fail(c, 500, err.Error(), "INGEST_ERROR")
	}

	return OK(c, fiber.Map{"status": "ingested", "source": req.Source})
}

func (h *RAGHandler) Upload(c fiber.Ctx) error {
	if h.ragEngine == nil {
		return Fail(c, 500, "rag engine not configured", "INGEST_ERROR")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return Fail(c, 400, "missing file", "BAD_REQUEST")
	}
	if file.Size <= 0 {
		return Fail(c, 400, "empty file", "BAD_REQUEST")
	}
	if file.Size > rag.MaxUploadBytes {
		return Fail(c, 413, "file too large", "FILE_TOO_LARGE")
	}

	f, err := file.Open()
	if err != nil {
		return Fail(c, 500, "cannot open file", "FILE_ERROR")
	}
	defer f.Close()

	text, err := rag.Parse(file.Filename, f)
	if err != nil {
		return Fail(c, 400, err.Error(), "PARSE_ERROR")
	}
	if text == "" {
		return Fail(c, 400, "no content extracted", "PARSE_ERROR")
	}

	chunks := rag.Chunk(text, rag.DefaultChunkOptions())
	if len(chunks) == 0 {
		return Fail(c, 400, "no chunks generated", "PARSE_ERROR")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	for _, chunk := range chunks {
		if err := h.ragEngine.Ingest(ctx, chunk, file.Filename); err != nil {
			return Fail(c, 500, err.Error(), "INGEST_ERROR")
		}
	}

	return OK(c, fiber.Map{
		"file":   file.Filename,
		"chunks": len(chunks),
	})
}
