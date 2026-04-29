-- +goose Up
CREATE TABLE IF NOT EXISTS audit_logs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          TEXT,
    username         TEXT,
    session_id       TEXT,
    method           TEXT NOT NULL,
    path             TEXT NOT NULL,
    intent           TEXT,
    query            TEXT,
    response_preview TEXT,
    sources          JSONB,
    latency_ms       INTEGER,
    status_code      INTEGER,
    ip_address       TEXT,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_user_id
    ON audit_logs(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_created_at
    ON audit_logs(created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_audit_created_at;
DROP INDEX IF EXISTS idx_audit_user_id;
DROP TABLE IF EXISTS audit_logs;
