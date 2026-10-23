ALTER TABLE outbox_messages
    
ADD COLUMN IF NOT EXISTS attempts INT NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS available_at TIMESTAMPTZ DEFAULT NOW();

DROP INDEX IF EXISTS idx_outbox_status_created_at;
CREATE INDEX idx_outbox_ready ON outbox_messages (status, available_at);