package kafka

import (
	"context"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

// MessageWriter - общий интерфейс для отправки сообщений.
type MessageWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// SmartResolver решает проблему DNS при локальной разработке vs Docker.
type SmartResolver struct {
	ThisDocker bool
}

func (r *SmartResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	if !r.ThisDocker && host == "payship-kafka" {
		host = "127.0.0.1"
	}
	return net.DefaultResolver.LookupHost(ctx, host)
}

// NewProducer создает и проверяет соединение с Kafka.
// При неудачном подключение завершает работу сервиса
func NewProducer(addr string, inDocker bool) MessageWriter {
	resolver := &SmartResolver{ThisDocker: inDocker}

	dialer := &kafka.Dialer{
		Timeout:   5 * time.Second,
		DualStack: true,
		Resolver:  resolver,
	}

	slog.Info("Checking connection to Kafka...", "addr", addr)
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		slog.Error("FAILED to connect to Kafka", "addr", addr, "error", err)
		slog.Error("Make sure Kafka is running and KAFKA_ADDR is correct")
		os.Exit(1) // Немедленный выход при старте
	}
	if err := conn.Close(); err != nil {
		slog.Warn("Failed to close check connection", "error", err) // Не фатально
	}
	slog.Info("Successfully connected to Kafka")

	customDialContext := func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err == nil && !inDocker && host == "payship-kafka" && port == "9092" {
			address = net.JoinHostPort("127.0.0.1", "9094")
		}
		d := &net.Dialer{Timeout: 5 * time.Second}
		return d.DialContext(ctx, network, address)
	}

	return &kafka.Writer{
		Addr:         kafka.TCP(addr),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Transport: &kafka.Transport{
			Dial: customDialContext,
		},
	}
}
