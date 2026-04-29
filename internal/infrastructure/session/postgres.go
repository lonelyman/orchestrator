package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresAdapter implement SessionPort สำหรับ PostgreSQL
type PostgresAdapter struct {
	pool   *pgxpool.Pool
	expiry time.Duration
}

// NewPostgresAdapter สร้าง PostgresAdapter ใหม่
func NewPostgresAdapter(pool *pgxpool.Pool, expiry time.Duration) *PostgresAdapter {
	return &PostgresAdapter{pool: pool, expiry: expiry}
}

// CreateSession สร้าง Session ใหม่
func (p *PostgresAdapter) CreateSession(ctx context.Context, userID string) (*models.Session, error) {
	session := &models.Session{
		UserID:    userID,
		ExpiresAt: time.Now().Add(p.expiry),
		IsActive:  true,
	}

	query := `
		INSERT INTO sessions (user_id, expires_at, is_active)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`

	err := p.pool.QueryRow(ctx, query, session.UserID, session.ExpiresAt, session.IsActive).
		Scan(&session.ID, &session.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return session, nil
}

// GetSession ดึง Session จาก ID
func (p *PostgresAdapter) GetSession(ctx context.Context, sessionID string) (*models.Session, error) {
	session := &models.Session{}
	query := `
		SELECT id, user_id, created_at, expires_at, is_active
		FROM sessions
		WHERE id = $1 AND is_active = true AND expires_at > NOW()`

	err := p.pool.QueryRow(ctx, query, sessionID).
		Scan(&session.ID, &session.UserID, &session.CreatedAt, &session.ExpiresAt, &session.IsActive)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	return session, nil
}

// GetHistory ดึง messages ล่าสุด N ข้อความ (Sliding Window)
func (p *PostgresAdapter) GetHistory(ctx context.Context, sessionID string, limit int) ([]models.Message, error) {
	query := `
		SELECT id, session_id, role, content, metadata, created_at
		FROM (
			SELECT id, session_id, role, content, metadata, created_at, sequence_number
			FROM messages
			WHERE session_id = $1
			ORDER BY sequence_number DESC
			LIMIT $2
		) recent
		ORDER BY sequence_number ASC`

	rows, err := p.pool.Query(ctx, query, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		var metadataBytes []byte
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &metadataBytes, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		if metadataBytes != nil {
			json.Unmarshal(metadataBytes, &msg.Metadata)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// SaveMessage บันทึก message ลง DB
func (p *PostgresAdapter) SaveMessage(ctx context.Context, msg models.Message) error {
	metadataBytes, _ := json.Marshal(msg.Metadata)

	query := `
		INSERT INTO messages (session_id, role, content, metadata)
		VALUES ($1, $2, $3, $4)`

	_, err := p.pool.Exec(ctx, query, msg.SessionID, msg.Role, msg.Content, metadataBytes)
	if err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	return nil
}

// SaveMessages บันทึก messages หลายรายการตามลำดับใน transaction เดียว
func (p *PostgresAdapter) SaveMessages(ctx context.Context, messages []models.Message) error {
	if len(messages) == 0 {
		return nil
	}

	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin save messages transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO messages (session_id, role, content, metadata)
		VALUES ($1, $2, $3, $4)`

	for _, msg := range messages {
		metadataBytes, _ := json.Marshal(msg.Metadata)
		if _, err := tx.Exec(ctx, query, msg.SessionID, msg.Role, msg.Content, metadataBytes); err != nil {
			return fmt.Errorf("save message: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit save messages transaction: %w", err)
	}
	return nil
}

// ExpireSession ปิด Session
func (p *PostgresAdapter) ExpireSession(ctx context.Context, sessionID string) error {
	_, err := p.pool.Exec(ctx, `UPDATE sessions SET is_active = false WHERE id = $1`, sessionID)
	return err
}
