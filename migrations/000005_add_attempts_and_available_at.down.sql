DROP INDEX IF EXISTS idx_outbox_ready;
CREATE INDEX idx_outbox_status_created_at ON outbox_messages (status, created_at);

ALTER TABLE outbox_messages DROP COLUMN IF EXISTS available_at;
ALTER TABLE outbox_messages DROP COLUMN IF EXISTS attempts;