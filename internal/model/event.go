package model

import (
	"time"

	"github.com/google/uuid"
)

// OrderCreatedEvent описывает факт создания заказа.
type OrderCreatedEvent struct {
	EventID   uuid.UUID   `json:"event_id"`
	OrderID   int64       `json:"order_id"`
	Customer  string      `json:"customer_name"`
	Amount    float64     `json:"amount"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
}

// TopicOrderCreated - имя топика в Kafka.
const TopicOrderCreated = "payship.orders.created"
