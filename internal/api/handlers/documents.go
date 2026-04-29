package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/gofiber/fiber/v3"
)

type DocumentHandler struct {
	store   documentStore
	timeout time.Duration
}

type documentStore interface {
	ListDocuments(ctx context.Context) ([]models.DocumentSummary, error)
	DeleteDocumentSource(ctx context.Context, source string) (int64, error)
}

func NewDocumentHandler(store documentStore) *DocumentHandler {
	return NewDocumentHandlerWithTimeout(store, 10*time.Second)
}

func NewDocumentHandlerWithTimeout(store documentStore, timeout time.Duration) *DocumentHandler {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &DocumentHandler{store: store, timeout: timeout}
}

func (h *DocumentHandler) List(c fiber.Ctx) error {
	if h.store == nil {
		return Fail(c, 500, "document store not configured", "DOCUMENT_ERROR")
	}

	ctx, cancel := context.WithTimeout(c.Context(), h.timeout)
	defer cancel()

	documents, err := h.store.ListDocuments(ctx)
	if err != nil {
		return Fail(c, 500, err.Error(), "DOCUMENT_ERROR")
	}
	if documents == nil {
		documents = []models.DocumentSummary{}
	}

	return OK(c, fiber.Map{
		"documents": documents,
		"count":     len(documents),
	})
}

func (h *DocumentHandler) Delete(c fiber.Ctx) error {
	if h.store == nil {
		return Fail(c, 500, "document store not configured", "DOCUMENT_ERROR")
	}

	source := strings.TrimSpace(c.Params("source"))
	if source == "" {
		source = strings.TrimSpace(c.Query("source"))
	}
	if source == "" {
		return Fail(c, 400, "source is required", "BAD_REQUEST")
	}

	ctx, cancel := context.WithTimeout(c.Context(), h.timeout)
	defer cancel()

	deleted, err := h.store.DeleteDocumentSource(ctx, source)
	if err != nil {
		return Fail(c, 500, err.Error(), "DOCUMENT_ERROR")
	}
	if deleted == 0 {
		return Fail(c, 404, "document not found", "DOCUMENT_NOT_FOUND")
	}

	return OK(c, fiber.Map{
		"source":         source,
		"deleted_chunks": deleted,
	})
}
