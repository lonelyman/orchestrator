package websearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

func TestTavilyAdapterSearch(t *testing.T) {
	var authHeader string
	var reqBody tavilySearchRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		authHeader = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"results":[{
				"title":"Example",
				"url":"https://example.com/news",
				"content":"Search snippet",
				"score":0.91,
				"published_date":"2026-04-30"
			}]
		}`))
	}))
	defer server.Close()

	adapter := NewTavilyAdapter(server.URL, "tvly-test", time.Second)
	results, err := adapter.Search(context.Background(), "latest news", models.WebSearchOptions{
		MaxResults: 3,
		Topic:      "news",
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if authHeader != "Bearer tvly-test" {
		t.Fatalf("expected bearer token, got %q", authHeader)
	}
	if reqBody.Query != "latest news" || reqBody.Topic != "news" || reqBody.MaxResults != 3 {
		t.Fatalf("unexpected request body: %+v", reqBody)
	}
	if reqBody.SearchDepth != "basic" || reqBody.IncludeAnswer || reqBody.IncludeRawContent {
		t.Fatalf("unexpected request options: %+v", reqBody)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if results[0].Title != "Example" || results[0].Source != "example.com" {
		t.Fatalf("unexpected result: %+v", results[0])
	}
	if results[0].PublishedAt.IsZero() {
		t.Fatalf("expected published date")
	}
}

func TestTavilyAdapterSearch_ReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "invalid api key", http.StatusUnauthorized)
	}))
	defer server.Close()

	adapter := NewTavilyAdapter(server.URL, "bad-key", time.Second)
	_, err := adapter.Search(context.Background(), "latest news", models.WebSearchOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTavilyAdapterSearch_RejectsEmptyQuery(t *testing.T) {
	adapter := NewTavilyAdapter("http://example.com", "key", time.Second)
	_, err := adapter.Search(context.Background(), " ", models.WebSearchOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
}
