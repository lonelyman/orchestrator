package vector

import (
	"context"
	"fmt"
	"log"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"
)

type PgvectorAdapter struct {
	pool *pgxpool.Pool
}

func NewPgvectorAdapter(pool *pgxpool.Pool) *PgvectorAdapter {
	return &PgvectorAdapter{pool: pool}
}

func (p *PgvectorAdapter) InitSchema(ctx context.Context) error {
	queries := []string{
		`CREATE EXTENSION IF NOT EXISTS vector`,
		`CREATE TABLE IF NOT EXISTS documents (
			id         TEXT PRIMARY KEY,
			content    TEXT NOT NULL,
			source     TEXT NOT NULL,
			embedding  vector(768),
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
	}
	for _, q := range queries {
		if _, err := p.pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

func (p *PgvectorAdapter) Store(ctx context.Context, doc models.Document) error {
	query := `
		INSERT INTO documents (id, content, source, embedding, created_at)
		VALUES ($1, $2, $3, $4::vector, $5)
		ON CONFLICT (id) DO UPDATE
		SET content = $2, source = $3, embedding = $4::vector`

	_, err := p.pool.Exec(ctx, query,
		doc.ID,
		doc.Content,
		doc.Source,
		pgvector.NewVector(doc.Embedding),
		doc.CreatedAt,
	)
	return err
}

func (p *PgvectorAdapter) Search(ctx context.Context, embedding []float32, limit int) ([]models.Document, error) {
	log.Printf("Search called: embedding_len=%d, limit=%d", len(embedding), limit)

	if len(embedding) == 0 {
		return nil, fmt.Errorf("search: empty embedding")
	}
	if limit <= 0 {
		limit = 3
	}

	query := `
		SELECT id, content, source, created_at
		FROM documents
		WHERE embedding IS NOT NULL
		ORDER BY embedding <=> $1::vector
		LIMIT $2`

	rows, err := p.pool.Query(ctx, query, pgvector.NewVector(embedding), limit)
	if err != nil {
		log.Printf("Search query error: %v", err)
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()

	var results []models.Document
	for rows.Next() {
		var doc models.Document
		if err := rows.Scan(&doc.ID, &doc.Content, &doc.Source, &doc.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		var docsWithEmbedding int
		if err := p.pool.QueryRow(ctx, `SELECT COUNT(embedding) FROM documents`).Scan(&docsWithEmbedding); err != nil {
			return nil, fmt.Errorf("search count embeddings: %w", err)
		}
		if docsWithEmbedding == 0 {
			log.Printf("Search results: 0 docs, rows_err=<nil>")
			return results, nil
		}

		log.Printf("Search similarity returned 0 rows despite docs_with_embedding=%d, fallback to latest docs", docsWithEmbedding)

		fallbackQuery := `
			SELECT id, content, source, created_at
			FROM documents
			WHERE embedding IS NOT NULL
			ORDER BY created_at DESC
			LIMIT $1`

		fallbackRows, err := p.pool.Query(ctx, fallbackQuery, limit)
		if err != nil {
			return nil, fmt.Errorf("search fallback: %w", err)
		}
		defer fallbackRows.Close()

		for fallbackRows.Next() {
			var doc models.Document
			if err := fallbackRows.Scan(&doc.ID, &doc.Content, &doc.Source, &doc.CreatedAt); err != nil {
				return nil, fmt.Errorf("scan fallback: %w", err)
			}
			results = append(results, doc)
		}
		if err := fallbackRows.Err(); err != nil {
			return nil, err
		}
	}

	log.Printf("Search results: %d docs, rows_err=%v", len(results), rows.Err())
	return results, nil
}

func (p *PgvectorAdapter) Delete(ctx context.Context, id string) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM documents WHERE id = $1`, id)
	return err
}
