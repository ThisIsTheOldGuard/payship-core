package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ThisIsTheOldGuard/payship-core/internal/model"
	"github.com/ThisIsTheOldGuard/payship-core/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type orderRepo struct {
	pool *pgxpool.Pool
}

// Возвращает адрес нашей таблицы, с которой будем работать
func NewOrderRepo(pool *pgxpool.Pool) service.OrderRepo {
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("orderRepo.GetByID: order not found")
		}
		return nil, fmt.Errorf("orderRepo.GetByID: %w", err)
	}

	return &order, nil
}

func (r *orderRepo) ListOrders(ctx context.Context, limit, offset int) ([]*model.Order, int, error) {

	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM orders`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("orderRepo.ListOrders.count: %w", err)
	}

	// https://pkg.go.dev/github.com/jackc/pgx#hdr-Query_Interface
	query := `SELECT id, customer_name, amount, status, created_at FROM orders
	ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	var orders []*model.Order

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, total, fmt.Errorf("orderRepo.ListOrders.query %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var order model.Order
		if err := rows.Scan(&order.ID, &order.CustomerName, &order.Amount, &order.Status, &order.CreatedAt); err != nil {
			return nil, total, fmt.Errorf("orderRepo.ListOrders.scan: %w", err)
		}
		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, total, fmt.Errorf("orderRepo.ListRows.iteration: %w", err)
	}

	return orders, total, nil
}

func (r *orderRepo) UpdateOrderTransition(ctx context.Context, id int64, status model.OrderStatus) error {

	query := `UPDATE orders SET status = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("orderRepo.UpdateOrderTransition: %w", err)
	}

	return nil

}
