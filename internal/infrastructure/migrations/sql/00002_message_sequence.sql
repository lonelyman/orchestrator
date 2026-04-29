-- +goose Up
ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS sequence_number BIGSERIAL;

DROP INDEX IF EXISTS idx_messages_session_created;

CREATE INDEX IF NOT EXISTS idx_messages_session_created
    ON messages(session_id, sequence_number ASC);

-- +goose Down
DROP INDEX IF EXISTS idx_messages_session_created;

CREATE INDEX IF NOT EXISTS idx_messages_session_created
    ON messages(session_id, created_at ASC);

ALTER TABLE messages
    DROP COLUMN IF EXISTS sequence_number;
