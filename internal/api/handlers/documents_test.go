package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/gofiber/fiber/v3"
)

type fakeDocumentStore struct {
	documents []models.DocumentSummary
	deleted   map[string]int64
}

func (f *fakeDocumentStore) ListDocuments(context.Context) ([]models.DocumentSummary, error) {
	return f.documents, nil
}

func (f *fakeDocumentStore) DeleteDocumentSource(_ context.Context, source string) (int64, error) {
	return f.deleted[source], nil
}

func TestDocumentHandlerList(t *testing.T) {
	store := &fakeDocumentStore{
		documents: []models.DocumentSummary{
			{Source: "policy.pdf", Chunks: 3, CreatedAt: time.Unix(1, 0), LastIngestAt: time.Unix(2, 0)},
		},
	}
	app := fiber.New()
	app.Get("/v1/rag/documents", NewDocumentHandler(store).List)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/v1/rag/documents", nil))
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var out Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Data == nil {
		t.Fatalf("expected data")
	}
}

func TestDocumentHandlerDelete(t *testing.T) {
	store := &fakeDocumentStore{deleted: map[string]int64{"policy.pdf": 3}}
	app := fiber.New()
	app.Delete("/v1/rag/documents/:source", NewDocumentHandler(store).Delete)

	resp, err := app.Test(httptest.NewRequest(http.MethodDelete, "/v1/rag/documents/policy.pdf", nil))
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestDocumentHandlerDeleteWithQuerySource(t *testing.T) {
	store := &fakeDocumentStore{deleted: map[string]int64{"บค.001.pdf": 2}}
	app := fiber.New()
	app.Delete("/v1/rag/documents", NewDocumentHandler(store).Delete)

	req := httptest.NewRequest(http.MethodDelete, "/v1/rag/documents?source=%E0%B8%9A%E0%B8%84.001.pdf", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestDocumentHandlerDeleteMissingSource(t *testing.T) {
	app := fiber.New()
	app.Delete("/v1/rag/documents", NewDocumentHandler(&fakeDocumentStore{}).Delete)

	resp, err := app.Test(httptest.NewRequest(http.MethodDelete, "/v1/rag/documents", nil))
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestDocumentHandlerDeleteNotFound(t *testing.T) {
	app := fiber.New()
	app.Delete("/v1/rag/documents/:source", NewDocumentHandler(&fakeDocumentStore{deleted: map[string]int64{}}).Delete)

	resp, err := app.Test(httptest.NewRequest(http.MethodDelete, "/v1/rag/documents/missing.pdf", nil))
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}
