CREATE TABLE IF NOT EXISTS outbox_messages (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, -- современное использование https://www.crunchydata.com/blog/postgres-serials-should-be-bigint-and-how-to-migrate
    event_id UUID NOT NULL,                             -- Уникальный ID события
    topic VARCHAR(255) NOT NULL,                        -- Имя топика
    partition_key VARCHAR(255) NOT NULL,                -- Ключ для партиций
    order_id BIGINT NOT NULL,                           -- Id заказа
    payload JSONB NOT NULL,                             -- Сообщение
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',      -- PENDING | SENT | FAILED
    error TEXT DEFAULT NULL,                            -- Текст ошибки
    created_at TIMESTAMPTZ DEFAULT NOW()                -- Дата создания
    );

-- Индекс для воркера
CREATE INDEX idx_outbox_status_created_at ON outbox_messages (status, created_at);
-- Для предотвращения дублирования записей
CREATE UNIQUE INDEX idx_outbox_dedup ON outbox_messages (topic, order_id);