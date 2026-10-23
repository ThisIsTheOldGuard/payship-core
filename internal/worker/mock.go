package worker

import (
	"context"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type MockProducer struct{}

func NewMockProducer() *MockProducer {
	return &MockProducer{}
}

func (m *MockProducer) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	for _, msg := range msgs {
		slog.Info("MOCK: Message sent successfully", "topic", msg.Topic, "key", string(msg.Key))
	}
	return nil
}
