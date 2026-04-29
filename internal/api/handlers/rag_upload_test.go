package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

type fakeRAGIngestor struct {
	ingests int
}

func (f *fakeRAGIngestor) Ingest(_ context.Context, content, _ string) error {
	if strings.TrimSpace(content) == "" {
		return fiber.NewError(http.StatusBadRequest, "empty content")
	}
	f.ingests++
	return nil
}

func TestUpload_WithProvidedPDF(t *testing.T) {
	pdfPath := os.Getenv("UPLOAD_TEST_PDF")
	if pdfPath == "" {
		pdfPath = "/Users/nipon.k/Documents/บค.002-2568 ประกาศ เรื่องแนวทางการขออนุมัติทำงานล่วงเวลาอย่างเคร่งครัด.pdf"
	}

	if _, err := os.Stat(pdfPath); err != nil {
		t.Skipf("test pdf not found: %s", pdfPath)
	}

	app := fiber.New()
	fake := &fakeRAGIngestor{}
	handler := NewRAGHandlerWithIngestor(fake)
	app.Post("/v1/rag/upload", handler.Upload)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(pdfPath))
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	f, err := os.Open(pdfPath)
	if err != nil {
		t.Fatalf("open test pdf: %v", err)
	}
	defer f.Close()

	if _, err := io.Copy(part, f); err != nil {
		t.Fatalf("copy pdf to multipart: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/rag/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 15 * time.Second})
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(b))
	}

	var out Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if fake.ingests == 0 {
		t.Fatalf("expected at least 1 ingest call")
	}
}
