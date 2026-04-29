-- +goose Up
ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS request_id TEXT;

CREATE INDEX IF NOT EXISTS idx_audit_request_id
    ON audit_logs(request_id);

-- +goose Down
DROP INDEX IF EXISTS idx_audit_request_id;

ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS request_id;
