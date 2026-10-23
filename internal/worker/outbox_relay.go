package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

type MessageWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type OutboxRelayer struct {
	db *pgxpool.Pool
	//kafkaWriter *kafka.Writer
	kafkaWriter MessageWriter
	batchSize   int
	metrics     DBMetrics
}

// NewOutboxRelayer создает экземпляр воркера.
// func NewOutboxRelayer(db *pgxpool.Pool, writer *kafka.Writer, batchSize int, metrics DBMetrics) *OutboxRelayer {
func NewOutboxRelayer(db *pgxpool.Pool, writer MessageWriter, batchSize int, metrics DBMetrics) *OutboxRelayer {
	if batchSize <= 0 {
		batchSize = 100
	}
	return &OutboxRelayer{
		db:          db,
		kafkaWriter: writer,
		batchSize:   batchSize,
		metrics:     metrics,
	}
}

// Run запускает бесконечный цикл обработки outbox
func (r *OutboxRelayer) Run(ctx context.Context) {

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Stop Outbox Relayer...")
			return
		case <-ticker.C:

			if ctx.Err() != nil {
				return
			}

			_, err := r.processOutboxBatch(ctx)
			if err != nil {
				slog.Error("Process Outbox Batch Error", "error", err)
			}
		}
	}
}

// https://habr.com/ru/companies/otus/articles/942122/
func (r *OutboxRelayer) processOutboxBatch(ctx context.Context) (processed int, err error) {

	startwork := time.Now()
	defer func() {
		r.metrics.ObserveQueryDuration("outbox_work", time.Since(startwork))
	}()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("worker.outbox begin tx: %w", err)
	}

	// Connection Leak via Canceled Context смотреть в ./internal/repository/order.go CreateWithOutbox
	// https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool@v5.9.2#Pool.Begin
	defer func() {
		if err != nil {
			rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer rollbackCancel()

			if rollbackErr := tx.Rollback(rollbackCtx); rollbackErr != nil {
				slog.Error("Worker.outbox rollback failed", "error", rollbackErr)
			}
		}
	}()

	start := time.Now()
	query := `
		SELECT id, event_id, topic, partition_key, payload, attempts
		FROM outbox_messages
		WHERE status = 'PENDING' AND available_at <= now()
		ORDER BY created_at ASC
		FOR UPDATE SKIP LOCKED 
		LIMIT $1`

	rows, err := tx.Query(ctx, query, r.batchSize)
	r.metrics.ObserveQueryDuration("outbox_select_pending", time.Since(start))
	if err != nil {
		return 0, fmt.Errorf("worker.outbox.query: %w", err)
	}

	defer rows.Close()

	type outboxRow struct {
		id       int64
		eventID  uuid.UUID
		topic    string
		pkey     string
		payload  []byte
		attempts int
	}

	var batch []outboxRow
	for rows.Next() {
		var r outboxRow
		if err := rows.Scan(&r.id, &r.eventID, &r.topic, &r.pkey, &r.payload, &r.attempts); err != nil {
			return 0, err
		}
		batch = append(batch, r)
	}

	if err := rows.Err(); err != nil {
		return 0, err
	}

	if len(batch) == 0 {
		commitCtx, commitCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer commitCancel()
		return 0, tx.Commit(commitCtx)
	}

	var idsSent []int64
	var failedIDs []int64
	var failedAttempts []int

	for _, event := range batch {
		err := r.kafkaWriter.WriteMessages(ctx, kafka.Message{
			Topic: event.topic,
			Key:   []byte(event.pkey),
			Value: event.payload,
		})

		if err != nil {
			failedIDs = append(failedIDs, event.id)
			failedAttempts = append(failedAttempts, event.attempts+1)
		} else {
			idsSent = append(idsSent, event.id)
		}
	}

	// В будущем надо написать отдельного уборщика, который бы чистил эту таблицу
	// DELETE FROM outbox_messages
	// WHERE status = 'SENT'
	// AND created_at < NOW() - INTERVAL '7 days'; -- Храним историю 7 дней
	if len(idsSent) > 0 {

		start := time.Now()
		query := `
			UPDATE outbox_messages
			SET status = 'SENT', attempts = 0, error = NULL
			WHERE id = ANY($1)`

		_, err = tx.Exec(ctx, query, idsSent)
		r.metrics.ObserveQueryDuration("outbox_select_pending", time.Since(start))
		if err != nil {
			return 0, fmt.Errorf("worker.outbox.set sent status: %w", err)
		}
	}

	if len(failedIDs) > 0 {
		start := time.Now()
		query := `
			UPDATE outbox_messages AS o
			SET
				attempts = t.new_attempts,
				available_at = now() + (power(2, t.new_attempts) * interval '100 milliseconds'),
				status = CASE WHEN t.new_attempts >= 10 THEN 'FAILED' ELSE o.status END,
				error = 'Kafka send failed: ' || $3
				FROM unnest($1::bigint[], $2::int[]) AS t(id, new_attempts)
			WHERE o.id = t.id`
		_, err = tx.Exec(ctx, query, failedIDs, failedAttempts, "timeout or broker error")
		r.metrics.ObserveQueryDuration("outbox_update_retry", time.Since(start))
		if err != nil {
			return 0, fmt.Errorf("Worker.outbox.set update retry: %w", err)
		}
	}

	commitCtx, commitCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer commitCancel()

	if err := tx.Commit(commitCtx); err != nil {
		return 0, fmt.Errorf("Worker.outbox commit failed: %w", err)
	}

	return len(batch), nil
}
