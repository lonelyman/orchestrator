package embedder

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNomicAdapterEmbed_ReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "embed model unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	adapter := NewNomicAdapter(server.URL, "test-embed")
	_, err := adapter.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "503") || !strings.Contains(err.Error(), "embed model unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNomicAdapterEmbed_DecodesSuccessfulResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embedding":[0.1,0.2,0.3]}`))
	}))
	defer server.Close()

	adapter := NewNomicAdapter(server.URL, "test-embed")
	embedding, err := adapter.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(embedding) != 3 {
		t.Fatalf("expected 3 dimensions, got %d", len(embedding))
	}
}
