-- +goose Up
CREATE TABLE IF NOT EXISTS documents (
    id         TEXT PRIMARY KEY,
    content    TEXT NOT NULL,
    source     TEXT NOT NULL,
    embedding  vector(768),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_documents_created_at
    ON documents(created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_documents_created_at;
DROP TABLE IF EXISTS documents;
