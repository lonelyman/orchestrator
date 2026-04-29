package rag

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOCRImageWithOllama(t *testing.T) {
	imagePath := filepath.Join(t.TempDir(), "page-1.png")
	if err := os.WriteFile(imagePath, []byte("fake-png"), 0o600); err != nil {
		t.Fatalf("write image: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var req ollamaOCRRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "ocr-model" {
			t.Fatalf("unexpected model: %s", req.Model)
		}
		if req.Prompt != "extract text" {
			t.Fatalf("unexpected prompt: %s", req.Prompt)
		}
		if req.Stream {
			t.Fatalf("expected stream=false")
		}
		if len(req.Images) != 1 {
			t.Fatalf("expected one image, got %d", len(req.Images))
		}
		gotImage, err := base64.StdEncoding.DecodeString(req.Images[0])
		if err != nil {
			t.Fatalf("decode image: %v", err)
		}
		if string(gotImage) != "fake-png" {
			t.Fatalf("unexpected image payload: %q", gotImage)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":"ข้อความ OCR"}`))
	}))
	defer server.Close()

	text, err := ocrImageWithOllama(server.Client(), server.URL, "ocr-model", "extract text", imagePath)
	if err != nil {
		t.Fatalf("ocrImageWithOllama() error = %v", err)
	}
	if text != "ข้อความ OCR" {
		t.Fatalf("unexpected OCR text: %q", text)
	}
}

func TestSortPageImages(t *testing.T) {
	images := []string{
		"/tmp/page-10.png",
		"/tmp/page-2.png",
		"/tmp/page-1.png",
	}

	sortPageImages(images)

	want := []string{
		"/tmp/page-1.png",
		"/tmp/page-2.png",
		"/tmp/page-10.png",
	}
	if !reflect.DeepEqual(images, want) {
		t.Fatalf("unexpected order: %#v", images)
	}
}
