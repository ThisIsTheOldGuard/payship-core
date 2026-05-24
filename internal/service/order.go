package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ThisIsTheOldGuard/payship-core/internal/model"
	"github.com/ThisIsTheOldGuard/payship-core/internal/repository"
)

var (
	ErrEmptyCustomer = errors.New("customer_name is required")
	ErrInvalidAmount = errors.New("amount must be greater than 0")
	ErrOrderNotFound = errors.New("order not found")
	ErrInvalidPage   = errors.New("page must be >= 1")
	ErrInvalidLimit  = errors.New("limit must be between 1 and 100")
)

type OrderService struct {
	repo repository.OrderRepo
}

func NewOrderService(repo repository.OrderRepo) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) CreateOrder(ctx context.Context, customerName string, amount float64) (*model.Order, error) {

	if customerName == "" {
		return nil, ErrEmptyCustomer
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	order := &model.Order{
		CustomerName: customerName,
		Amount:       amount,
		Status:       model.StatusPending,
	}

	if err := s.repo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("service.CreateOrder: %w", err)
	}

	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id int64) (*model.Order, error) {
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service.GetOrder: %w", err)
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

func (s *OrderService) ListOrders(ctx context.Context, limit, page int) ([]*model.Order, int, error) {
	if page < 1 {
		return nil, 0, ErrInvalidPage
	}
	if limit < 1 || limit > 100 {
		return nil, 0, ErrInvalidLimit
	}

	offset := (page - 1) * limit
	orders, count, err := s.repo.ListOrders(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("service.ListOrders: %w", err)
	}

	return orders, count, nil
}
