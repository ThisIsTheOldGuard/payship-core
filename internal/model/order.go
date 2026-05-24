package model

import "time"

type Order struct {
	ID           int64       `json:"id"`
	CustomerName string      `json:"customer_name"`
	Amount       float64     `json:"amount"`
	Status       OrderStatus `json:"status"`
	CreatedAt    time.Time   `json:"created_at"`
}

type OrderStatus string

const (
	StatusPending    OrderStatus = "pending"
	StatusProcessing OrderStatus = "processing"
	StatusCompleted  OrderStatus = "completed"
	StatusCancelled  OrderStatus = "cancelled"
)
