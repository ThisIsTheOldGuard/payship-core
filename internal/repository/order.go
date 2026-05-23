package repository

import (
	"context"
	"fmt"

	"github.com/ThisIsTheOldGuard/payship-core/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepo interface {
	Create(ctx context.Context, order *model.Order) error
	GetByID(ctx context.Context, id int64) (*model.Order, error)
}

type orderRepo struct {
	pool *pgxpool.Pool
}

// Возвращает адрес нашей таблицы, с которой будем работать
func NewOrderRepo(pool *pgxpool.Pool) OrderRepo {
	return &orderRepo{pool: pool}
}

// Реализация метода Create - создания заказа
// Мутируем данные полученной таблицы для ответа. По информации из интернета, является нормальным для database/sql и pgx
func (r *orderRepo) Create(ctx context.Context, order *model.Order) error {

	query := `INSERT INTO orders (customer_name, amount, status) VALUES ($1, $2, $3) RETURNING id, created_at`
	err := r.pool.QueryRow(ctx, query, order.CustomerName, order.Amount, order.Status).Scan(&order.ID, &order.CreatedAt)

	if err != nil {
		return fmt.Errorf("orderRepo.Create: %w", err)
	}

	return nil

}

func (r *orderRepo) GetByID(ctx context.Context, id int64) (*model.Order, error) {

	query := `SELECT id, customer_name, amount, status, created_at FROM orders WHERE id = $1`
	var order model.Order
	err := r.pool.QueryRow(ctx, query, id).Scan(&order.ID, &order.CustomerName, &order.Amount, &order.Status, &order.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("orderRepo.GetByID: %w", err)
	}

	return &order, nil
}
