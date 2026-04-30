package ports

import (
	"context"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

type WebSearchPort interface {
	Search(ctx context.Context, query string, opts models.WebSearchOptions) ([]models.WebSearchResult, error)
	HealthCheck(ctx context.Context) error
}
