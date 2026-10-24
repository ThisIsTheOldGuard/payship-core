CREATE TABLE IF NOT EXISTS notification_log (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id UUID NOT NULL UNIQUE,
    order_id BIGINT NOT NULL,
    status VARCHAR(50) NOT NULL,
    error TEXT DEFAULT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
    );

CREATE INDEX idx_notification_event_id ON notification_log (event_id);