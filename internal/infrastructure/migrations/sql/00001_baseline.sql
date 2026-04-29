-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE sessions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE messages (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role       TEXT NOT NULL,
    content    TEXT NOT NULL,
    metadata   JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE audit_logs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id       TEXT,
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
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE documents (
    id         TEXT PRIMARY KEY,
    content    TEXT NOT NULL,
    source     TEXT NOT NULL,
    embedding  vector(768),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_expires
    ON sessions(user_id, expires_at DESC);

CREATE INDEX idx_messages_session_created
    ON messages(session_id, created_at ASC);

CREATE INDEX idx_audit_request_id
    ON audit_logs(request_id);

CREATE INDEX idx_audit_user_created
    ON audit_logs(user_id, created_at DESC);

CREATE INDEX idx_audit_created
    ON audit_logs(created_at DESC);

CREATE INDEX idx_documents_created
    ON documents(created_at DESC);

CREATE INDEX idx_documents_source_created
    ON documents(source, created_at DESC);

CREATE INDEX idx_documents_embedding_hnsw
    ON documents USING hnsw (embedding vector_cosine_ops);

-- +goose Down
DROP INDEX idx_documents_embedding_hnsw;
DROP INDEX idx_documents_source_created;
DROP INDEX idx_documents_created;
DROP INDEX idx_audit_created;
DROP INDEX idx_audit_user_created;
DROP INDEX idx_audit_request_id;
DROP INDEX idx_messages_session_created;
DROP INDEX idx_sessions_user_expires;

DROP TABLE documents;
DROP TABLE audit_logs;
DROP TABLE messages;
DROP TABLE sessions;

DROP EXTENSION IF EXISTS pgcrypto;
DROP EXTENSION IF EXISTS vector;
