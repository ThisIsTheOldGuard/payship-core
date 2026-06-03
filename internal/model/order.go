// Package model предоставляет DTO и доменные типы для микросервиса заказов.
//
// Этот пакет содержит структуры данных, которые передаются между слоями
// приложения (repository, service, api). Все типы используют JSON-теги
// в формате snake_case для совместимости с внешними клиентами.
//
// Пример использования:
//
//	order := &model.Order{
//	    CustomerName: "Alice",
//	    Amount:       100.50,
//	    Status:       model.StatusPending,
//	}
package model

import (
	"encoding/json"
	"fmt"
	"time"
)

// Order представляет заказ в системе обработки платежей.
//
// Поля:
//   - ID: уникальный идентификатор заказа (заполняется БД при создании).
//   - CustomerName: имя клиента, размещающего заказ.
//   - Amount: сумма заказа. Временно используется float64.
//   - Status: текущий статус заказа в жизненном цикле.
//   - CreatedAt: время создания заказа в формате RFC3339.
//
// JSON-теги обеспечивают сериализацию в snake_case для API-контракта.
type Order struct {
	ID           int64       `json:"id"`
	CustomerName string      `json:"customer_name"`
	Amount       float64     `json:"amount"`
	Status       OrderStatus `json:"status"`
	CreatedAt    time.Time   `json:"created_at"`
}

// OrderStatus представляет допустимые статусы заказа в жизненном цикле.
//
// Тип основан на string для совместимости с БД и JSON.
// Допустимые значения определены константами:
//   - StatusPending: заказ создан, ожидает обработки.
//   - StatusProcessing: заказ в работе.
//   - StatusCompleted: заказ успешно завершён.
//   - StatusCancelled: заказ отменён.
//
// Пример:
//
//	var status model.OrderStatus = model.StatusPending
type OrderStatus string

const (
	// StatusPending - заказ создан, ожидает обработки.
	StatusPending OrderStatus = "pending"
	// StatusProcessing - заказ принят в работу.
	StatusProcessing OrderStatus = "processing"
	// StatusCompleted - заказ успешно завершён и оплачен.
	StatusCompleted OrderStatus = "completed"
	// StatusCancelled - заказ отменён клиентом или системой.
	StatusCancelled OrderStatus = "cancelled"
)

// Valid проверяет, является ли значение OrderStatus допустимым.
//
// Метод возвращает true, если статус входит в набор констант:
// StatusPending, StatusProcessing, StatusCompleted, StatusCancelled.
// Используется для валидации входных данных перед бизнес-логикой.
//
// Возвращает:
//   - bool: true если статус валиден, false иначе.
//
// Пример:
//
//	if !status.Valid() {
//	    return errors.New("invalid status")
//	}
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
