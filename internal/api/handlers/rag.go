package handlers

import (
	"context"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/core/rag"
	"github.com/gofiber/fiber/v3"
)

type RAGHandler struct {
	ragEngine        ragIngestor
	maxUploadBytes   int64
	parseOptions     rag.ParseOptions
	ragIngestTimeout time.Duration
}

type ragIngestor interface {
	Ingest(ctx context.Context, content, source string) error
}

func NewRAGHandler(ragEngine *rag.RAGEngine) *RAGHandler {
	return NewRAGHandlerWithLimit(ragEngine, rag.MaxUploadBytes, 120*time.Second)
}

func NewRAGHandlerWithIngestor(ragEngine ragIngestor) *RAGHandler {
	return NewRAGHandlerWithLimit(ragEngine, rag.MaxUploadBytes, 120*time.Second)
}

func NewRAGHandlerWithLimit(ragEngine ragIngestor, maxUploadBytes int64, ragIngestTimeout time.Duration) *RAGHandler {
	return NewRAGHandlerWithOptions(ragEngine, rag.ParseOptions{MaxBytes: maxUploadBytes}, ragIngestTimeout)
}

func NewRAGHandlerWithOptions(ragEngine ragIngestor, parseOptions rag.ParseOptions, ragIngestTimeout time.Duration) *RAGHandler {
	maxUploadBytes := parseOptions.MaxBytes
	if maxUploadBytes <= 0 {
		maxUploadBytes = rag.MaxUploadBytes
	}
	parseOptions.MaxBytes = maxUploadBytes
	if ragIngestTimeout <= 0 {
		ragIngestTimeout = 120 * time.Second
	}
	return &RAGHandler{
		ragEngine:        ragEngine,
		maxUploadBytes:   maxUploadBytes,
		parseOptions:     parseOptions,
		ragIngestTimeout: ragIngestTimeout,
	}
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

	ctx, cancel := context.WithTimeout(context.Background(), h.ragIngestTimeout)
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
	if file.Size > h.maxUploadBytes {
		return Fail(c, 413, "file too large", "FILE_TOO_LARGE")
	}

	f, err := file.Open()
	if err != nil {
		return Fail(c, 500, "cannot open file", "FILE_ERROR")
	}
	defer f.Close()

	text, err := rag.ParseWithOptions(file.Filename, f, h.parseOptions)
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

	ctx, cancel := context.WithTimeout(context.Background(), h.ragIngestTimeout)
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
