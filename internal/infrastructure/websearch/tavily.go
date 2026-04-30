package websearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

const maxTavilyErrorBodyBytes = 4 << 10

type TavilyAdapter struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewTavilyAdapter(baseURL, apiKey string, timeout time.Duration) *TavilyAdapter {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &TavilyAdapter{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  strings.TrimSpace(apiKey),
		client:  &http.Client{Timeout: timeout},
	}
}

type tavilySearchRequest struct {
	Query             string `json:"query"`
	Topic             string `json:"topic,omitempty"`
	SearchDepth       string `json:"search_depth,omitempty"`
	MaxResults        int    `json:"max_results,omitempty"`
	IncludeAnswer     bool   `json:"include_answer"`
	IncludeRawContent bool   `json:"include_raw_content"`
}

type tavilySearchResponse struct {
	Results []tavilyResult `json:"results"`
}

type tavilyResult struct {
	Title         string  `json:"title"`
	URL           string  `json:"url"`
	Content       string  `json:"content"`
	Score         float64 `json:"score"`
	PublishedDate string  `json:"published_date"`
}

func (t *TavilyAdapter) Search(ctx context.Context, query string, opts models.WebSearchOptions) ([]models.WebSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	maxResults := opts.MaxResults
	if maxResults <= 0 {
		maxResults = 5
	}
	if maxResults > 10 {
		maxResults = 10
	}

	topic := strings.TrimSpace(opts.Topic)
	if topic == "" {
		topic = "general"
	}

	reqBody := tavilySearchRequest{
		Query:             query,
		Topic:             topic,
		SearchDepth:       "basic",
		MaxResults:        maxResults,
		IncludeAnswer:     false,
		IncludeRawContent: false,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal tavily request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+"/search", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create tavily request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send tavily request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkTavilyStatus(resp); err != nil {
		return nil, err
	}

	var searchResp tavilySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("decode tavily response: %w", err)
	}

	results := make([]models.WebSearchResult, 0, len(searchResp.Results))
	for _, item := range searchResp.Results {
		publishedAt := parseTavilyDate(item.PublishedDate)
		results = append(results, models.WebSearchResult{
			Title:       item.Title,
			URL:         item.URL,
			Snippet:     item.Content,
			Source:      hostFromURL(item.URL),
			PublishedAt: publishedAt,
			Score:       item.Score,
		})
	}
	return results, nil
}

func (t *TavilyAdapter) HealthCheck(ctx context.Context) error {
	_, err := t.Search(ctx, "health check", models.WebSearchOptions{MaxResults: 1})
	return err
}

func checkTavilyStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxTavilyErrorBodyBytes))
	message := strings.TrimSpace(string(body))
	if message == "" {
		return fmt.Errorf("tavily search returned status %d", resp.StatusCode)
	}
	return fmt.Errorf("tavily search returned status %d: %s", resp.StatusCode, message)
}

func parseTavilyDate(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, time.DateOnly, "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func hostFromURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	withoutScheme := strings.TrimPrefix(strings.TrimPrefix(value, "https://"), "http://")
	host, _, _ := strings.Cut(withoutScheme, "/")
	return host
}
