DROP INDEX IF EXISTS idx_outbox_dedup;
DROP INDEX IF EXISTS idx_outbox_status_created_at;
DROP TABLE IF EXISTS outbox_messages;