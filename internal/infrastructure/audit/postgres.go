package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresAdapter implement AuditPort
type PostgresAdapter struct {
	pool *pgxpool.Pool
}

func NewPostgresAdapter(pool *pgxpool.Pool) *PostgresAdapter {
	return &PostgresAdapter{pool: pool}
}

// Save บันทึก audit log
func (p *PostgresAdapter) Save(ctx context.Context, log models.AuditLog) error {
	sourcesJSON, _ := json.Marshal(log.Sources)

	query := `
		INSERT INTO audit_logs 
		(request_id, user_id, username, session_id, method, path, intent, 
		 query, response_preview, sources, latency_ms, status_code, ip_address)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`

	_, err := p.pool.Exec(ctx, query,
		log.RequestID,
		log.UserID, log.Username, log.SessionID,
		log.Method, log.Path, log.Intent,
		log.Query, log.ResponsePreview, sourcesJSON,
		log.LatencyMs, log.StatusCode, log.IPAddress,
	)
	if err != nil {
		return fmt.Errorf("save audit log: %w", err)
	}
	return nil
}

// List ดึง audit logs ตาม filter
func (p *PostgresAdapter) List(ctx context.Context, filter models.AuditLogFilter) ([]models.AuditLog, error) {
	if filter.Limit == 0 {
		filter.Limit = 50
	}

	query := `
		SELECT id, request_id, user_id, username, session_id, method, path,
		       intent, query, response_preview, sources,
		       latency_ms, status_code, ip_address, created_at
		FROM audit_logs
		WHERE ($1 = '' OR user_id = $1)
		  AND ($2 = '' OR intent = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := p.pool.Query(ctx, query,
		filter.UserID, filter.Intent,
		filter.Limit, filter.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var log models.AuditLog
		var sourcesBytes []byte
		if err := rows.Scan(
			&log.ID, &log.RequestID, &log.UserID, &log.Username, &log.SessionID,
			&log.Method, &log.Path, &log.Intent, &log.Query,
			&log.ResponsePreview, &sourcesBytes,
			&log.LatencyMs, &log.StatusCode, &log.IPAddress, &log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		if sourcesBytes != nil {
			json.Unmarshal(sourcesBytes, &log.Sources)
		}
		logs = append(logs, log)
	}

	return logs, nil
}
