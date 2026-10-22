package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ThisIsTheOldGuard/payship-core/internal/model"
	"github.com/jackc/pgx/v5"
)

// CreateOutboxMessage записывает доменное событие в таблицу outbox_messages
// в рамках переданной транзакции.
//
// Гарантирует атомарность: событие будет сохранено только если транзакция
// создания заказа успешно завершится (Commit). При откате (Rollback) запись
// в outbox также отменится.
//
// Параметры:
//   - ctx: контекст для отмены операции или установки таймаута.
//   - tx: активная транзакция БД (pgx.Tx), в рамках которой выполняется операция.
//   - event: структурированное доменное событие (OrderCreatedEvent).
//
// Возвращает:
//   - error: ошибка сериализации payload или выполнения SQL-запроса, nil при успехе.
func CreateOutboxMessage(ctx context.Context, tx pgx.Tx, event model.OrderCreatedEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}

	query := `
		INSERT INTO outbox_messages (event_id, topic, partition_key, order_id, payload, status) VALUES ($1, $2, $3, $4, $5, 'PENDING')`

	_, err = tx.Exec(ctx, query,
		event.EventID,
		model.TopicOrderCreated,
		strconv.FormatInt(event.OrderID, 10),
		event.OrderID,
		payload,
	)

	if err != nil {
		return fmt.Errorf("outbox.Create: %w", err)
	}

	return nil
}
