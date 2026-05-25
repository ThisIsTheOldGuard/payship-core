package model

import (
	"encoding/json"
	"fmt"
	"time"
)

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

func (s OrderStatus) Valid() bool {
	switch s {
	case StatusPending, StatusProcessing, StatusCompleted, StatusCancelled:
		return true
	default:
		return false
	}
}

func (s *OrderStatus) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	candidate := OrderStatus(str)
	if !candidate.Valid() {
		return fmt.Errorf("invalid status: %s", str)
	}
	*s = candidate
	return nil
}
