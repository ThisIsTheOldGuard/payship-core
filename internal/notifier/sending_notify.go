package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

type OrderEvent struct {
	EventID string `json:"event_id"`
	OrderID int64  `json:"order_id"`
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
}

type Consumer struct {
	reader  *kafka.Reader
	pool    *pgxpool.Pool
	metrics DBMetrics
}

func NewConsumer(reader *kafka.Reader, pool *pgxpool.Pool, metrics DBMetrics) *Consumer {
	return &Consumer{
		reader:  reader,
		pool:    pool,
		metrics: metrics,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	slog.Info("Starting Kafka Consumer...", "topic", c.reader.Config().Topic, "group", c.reader.Config().GroupID)

	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("Consumer context canceled, stopping...")
				return
			}
			slog.Error("Failed to read message from Kafka", "error", err)
			continue
		}

		if err := c.processMessage(ctx, msg); err != nil {
			slog.Error("Failed to process message", "offset", msg.Offset, "error", err)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			slog.Error("Failed to commit Kafka offset", "offset", msg.Offset, "error", err)
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) error {

	startwork := time.Now()
	defer func() {
		c.metrics.ObserveQueryDuration("outbox_work", time.Since(startwork))
	}()

	var event OrderEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	start := time.Now()
	query := `
		INSERT INTO notification_log (event_id, order_id, status)
		VALUES ($1, $2, 'SENT')
		ON CONFLICT (event_id) DO NOTHING
	`

	result, err := c.pool.Exec(ctx, query, event.EventID, event.OrderID)
	c.metrics.ObserveQueryDuration("notify_sending", time.Since(start))
	if err != nil {
		return fmt.Errorf("db insert failed: %w", err)
	}

	if result.RowsAffected() == 0 {
		slog.Info("Duplicate message detected, skipping", "event_id", event.EventID, "order_id", event.OrderID)
		return nil
	}

	slog.Info("Отправили уведомление",
		"order_id", event.OrderID,
		"event_id", event.EventID,
		"mock_email", fmt.Sprintf("customer_%d@example.com", event.OrderID))

	time.Sleep(100 * time.Millisecond)
	return nil
}

func (c *Consumer) Close() error {
	slog.Info("Closing Kafka Consumer...")
	return c.reader.Close()
}
