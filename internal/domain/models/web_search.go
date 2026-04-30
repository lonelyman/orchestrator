package models

import "time"

type WebSearchOptions struct {
	MaxResults int           `json:"max_results"`
	Topic      string        `json:"topic,omitempty"`
	Recency    time.Duration `json:"recency,omitempty"`
	Region     string        `json:"region,omitempty"`
	SafeSearch bool          `json:"safe_search"`
}

type WebSearchResult struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Snippet     string    `json:"snippet"`
	Source      string    `json:"source,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
	Score       float64   `json:"score,omitempty"`
}
