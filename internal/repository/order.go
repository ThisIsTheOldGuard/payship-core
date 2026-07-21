// Package repository реализует слой доступа к данным для микросервиса заказов.
//
// Этот пакет предоставляет конкретную реализацию интерфейса service.OrderRepo
// с использованием PostgreSQL и драйвера pgxpool. Все методы принимают
// context.Context для поддержки таймаутов и отмены запросов.
//
// Пример использования:
//
//	repo := repository.NewOrderRepo(pool)
//	order, err := repo.GetByID(ctx, 123)
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ThisIsTheOldGuard/payship-core/internal/domain"
	"github.com/ThisIsTheOldGuard/payship-core/internal/model"
	"github.com/ThisIsTheOldGuard/payship-core/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// orderRepo - структура для описания методов получателя
type orderRepo struct {
	pool    *pgxpool.Pool
	metrics DBMetrics
}

// NewOrderRepo создаёт новую реализацию репозитория заказов.
//
// Функция принимает настроенный *pgxpool.Pool и возвращает экземпляр,
// удовлетворяющий интерфейсу service.OrderRepo. Пул соединений должен
// быть инициализирован и проверен (Ping) перед передачей.
//
// Параметры:
//   - pool: пул соединений с PostgreSQL.
//
// Возвращает:
//   - service.OrderRepository: реализация репозитория.
//
// Пример:
//
//	pool, _ := pgxpool.New(ctx, dsn)
//	repo := NewOrderRepo(pool)
func NewOrderRepo(pool *pgxpool.Pool, metrics DBMetrics) service.OrderRepository {
	return &orderRepo{pool: pool, metrics: metrics}
}

// Create сохраняет новый заказ в базе данных.
//
// Метод выполняет INSERT с RETURNING id для заполнения поля ID
// входной структуры. Операция выполняется в контексте переданного ctx.
//
// Параметры:
//   - ctx: контекст для отмены/таймаута запроса.
//   - order: указатель на модель заказа; поле ID и CreatedAt будет заполнено после успеха.
//
// Возвращает:
//   - error: ошибка выполнения запроса или нарушения ограничений БД.
//
// Пример:
//
//	order := &model.Order{CustomerName: "Bob", Amount: 50.0, Status: model.StatusPending}
//	if err := repo.Create(ctx, order); err != nil {
//	    return err
//	}
//	fmt.Println("Created order ID:", order.ID)
func (r *orderRepo) Create(ctx context.Context, order *model.Order) error {

	start := time.Now()
	defer func() {
		r.metrics.ObserveQueryDuration("create_order", time.Since(start))
	}()

	query := `INSERT INTO orders (customer_name, amount, status) VALUES ($1, $2, $3) RETURNING id, created_at`
	err := r.pool.QueryRow(ctx, query, order.CustomerName, order.Amount, order.Status).Scan(&order.ID, &order.CreatedAt)

	if err != nil {
		return fmt.Errorf("orderRepo.Create: %w", err)
	}

	return nil

}

// GetByID возвращает заказ по его уникальному идентификатору.
//
// Метод выполняет SELECT по первичному ключу.
//
// Параметры:
//   - ctx: контекст для отмены/таймаута запроса.
//   - id: уникальный идентификатор заказа.
//
// Возвращает:
//   - *model.Order: указатель на найденный заказ.
//   - error: если заказ отсутствует, или ошибка БД.
//
// Пример:
//
//	order, err := repo.GetByID(ctx, 42)
//	if errors.Is(err, repository.ErrOrderNotFound) {
//	    // обработать 404
//	}
func (r *orderRepo) GetByID(ctx context.Context, id int64) (*model.Order, error) {

	start := time.Now()
	defer func() {
		r.metrics.ObserveQueryDuration("get_order", time.Since(start))
	}()

	query := `SELECT id, customer_name, amount, status, created_at FROM orders WHERE id = $1`
	var order model.Order
	err := r.pool.QueryRow(ctx, query, id).Scan(&order.ID, &order.CustomerName, &order.Amount, &order.Status, &order.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("orderRepo.GetByID: %w", err)
	}

	return &order, nil
}

// ListOrders возвращает страницу заказов с поддержкой пагинации.
//
// Метод выполняет два запроса:
//  1. COUNT(*) для получения общего количества записей.
//  2. SELECT с LIMIT/OFFSET для выборки данных страницы.
//
// Параметры:
//   - ctx: контекст для отмены/таймаута запроса.
//   - offset: смещение первой записи (0-based).
//   - limit: максимальное количество записей на странице.
//
// Возвращает:
//   - []*model.Order: срез заказов на текущей странице.
//   - int: общее количество заказов (для расчёта страниц).
//   - error: ошибка выполнения запросов.
//
// Пример:
//
//	orders, total, err := repo.List(ctx, 0, 20)
//	pages := (total + 19) / 20 // округление вверх
func (r *orderRepo) ListOrders(ctx context.Context, limit, offset int) ([]*model.Order, int, error) {

	start := time.Now()
	defer func() {
		r.metrics.ObserveQueryDuration("select_orders", time.Since(start))
	}()

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

// UpdateOrderTransition обновляет статус существующего заказа.
//
// Метод выполняет UPDATE по первичному ключу. Если заказ с указанным ID
// не существует, операция завершается без ошибки (обновлено 0 строк).
//
// Параметры:
//   - ctx: контекст для отмены/таймаута запроса.
//   - id: уникальный идентификатор заказа.
//   - status: новый допустимый статус заказа.
//
// Возвращает:
//   - error: ошибка выполнения запроса.
//
// Пример:
//
//	err := repo.UpdateStatus(ctx, 123, model.StatusCompleted)
//	if err != nil {
//	    // обработать ошибку БД
//	}
func (r *orderRepo) UpdateOrderTransition(ctx context.Context, id int64, status model.OrderStatus) error {

	start := time.Now()
	defer func() {
		r.metrics.ObserveQueryDuration("update_order", time.Since(start))
	}()

	query := `UPDATE orders SET status = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("orderRepo.UpdateOrderTransition: %w", err)
	}

	return nil

}
