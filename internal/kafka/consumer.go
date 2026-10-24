package kafka

import (
	"context"
	"net"
	"time"

	"github.com/segmentio/kafka-go"
)

// NewReader создает и настраивает Kafka Reader с правильной сетевой логикой (SmartResolver).
func NewReader(brokers []string, topic, groupID string, inDocker bool) *kafka.Reader {
	resolver := &SmartResolver{ThisDocker: inDocker}

	customDialFunc := func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err == nil && !inDocker && host == "payship-kafka" && port == "9092" {
			address = net.JoinHostPort("127.0.0.1", "9094")
		}
		d := &net.Dialer{Timeout: 5 * time.Second}
		return d.DialContext(ctx, network, address)
	}

	dialer := &kafka.Dialer{
		Timeout:   5 * time.Second,
		DualStack: true,
		Resolver:  resolver,
		DialFunc:  customDialFunc,
	}

	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
		Dialer:   dialer,
	})
}
