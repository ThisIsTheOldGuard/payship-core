package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ThisIsTheOldGuard/payship-core/internal/model"
)

// mockOrderRepo имитирует service.OrderRepo для модульного тестирования.
type mockOrderRepo struct {
	createCalled           bool
	getByIDCalled          bool
	listOrdersCalled       bool
	updateTransitionCalled bool

	createRes           error
	getByIDRes          *model.Order
	getByIDErr          error
	listOrdersRes       []*model.Order
	listOrdersTotal     int
	listOrdersErr       error
	updateTransitionRes error
}

// Метод Create имитирует метод Create репозитория.
// Устанавливает флаг createCalled в значение true и возвращает предварительно настроенные значения.
func (m *mockOrderRepo) Create(ctx context.Context, order *model.Order) error {
	m.createCalled = true
	return m.createRes
}

// Метод GetByID имитирует метод Create репозитория.
// Устанавливает флаг createCalled в значение true и возвращает предварительно настроенные значения.
func (m *mockOrderRepo) GetByID(ctx context.Context, id int64) (*model.Order, error) {
	m.getByIDCalled = true
	return m.getByIDRes, m.getByIDErr
}

// Метод ListOrders имитирует метод Create репозитория.
// Устанавливает флаг createCalled в значение true и возвращает предварительно настроенные значения.
func (m *mockOrderRepo) ListOrders(ctx context.Context, limit, offset int) ([]*model.Order, int, error) {
	m.listOrdersCalled = true
	return m.listOrdersRes, m.listOrdersTotal, m.listOrdersErr
}

// Метод UpdateOrderTransition имитирует метод Create репозитория.
// Устанавливает флаг createCalled в значение true и возвращает предварительно настроенные значения.
func (m *mockOrderRepo) UpdateOrderTransition(ctx context.Context, id int64, status model.OrderStatus) error {
	m.updateTransitionCalled = true
	return m.updateTransitionRes
}

// TestGetOrder проверяет получение заказа по идентификатору.
func TestGetOrder(t *testing.T) {
	tests := []struct {
		name         string
		id           int64
		mockRepo     mockOrderRepo
		wantErr      error
		wantAnyError bool
		checkData    func(t *testing.T, order *model.Order)
	}{
		{
			name: "success",
			id:   123,
			mockRepo: mockOrderRepo{
				getByIDRes: &model.Order{ID: 123, CustomerName: "Anya", Amount: 150.0, Status: model.StatusPending},
				getByIDErr: nil,
			},
			wantErr: nil,
			checkData: func(t *testing.T, order *model.Order) {
				if order == nil {
					t.Fatal("expected non-nil order")
				}
				if order.ID != 123 {
					t.Errorf("got ID %d, want 123", order.ID)
				}
				if order.CustomerName != "Anya" {
					t.Errorf("got CustomerName %s, want Anya", order.CustomerName)
				}
			},
		},
		{
			name: "order not found",
			id:   999,
			mockRepo: mockOrderRepo{
				getByIDRes: nil,
				getByIDErr: ErrOrderNotFound,
			},
			wantErr: ErrOrderNotFound,
			checkData: func(t *testing.T, order *model.Order) {
				if order != nil {
					t.Error("expected nil order for not found case")
				}
			},
		},
		{
			name: "database error (any)",
			id:   1,
			mockRepo: mockOrderRepo{
				getByIDRes: nil,
				getByIDErr: errors.New("Ogo oshibka!"),
			},
			wantAnyError: true,
			checkData: func(t *testing.T, order *model.Order) {
				if order != nil {
					t.Error("expected nil order on DB failure")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.mockRepo
			svc := NewOrderService(&mock)
			ctx := context.Background()

			order, err := svc.GetOrder(ctx, tt.id)

			if tt.wantAnyError {
				if err == nil {
					t.Error("expected an error, got nil")
				}
			} else if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("got error %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}

			if tt.checkData != nil && err == nil {
				tt.checkData(t, order)
			}
		})
	}
}

// TestCreateOrder проверяет создание нового заказа с валидацией бизнес-правил.
func TestCreateOrder(t *testing.T) {
	tests := []struct {
		name      string
		customer  string
		amount    float64
		mockRepo  mockOrderRepo
		wantErr   error
		checkMock func(t *testing.T, m *mockOrderRepo)
	}{{
		name:     "success",
		customer: "Anya",
		amount:   150.0,
		mockRepo: mockOrderRepo{createRes: nil},
		wantErr:  nil,
		checkMock: func(t *testing.T, m *mockOrderRepo) {
			if !m.createCalled {
				t.Errorf("expected Create to be called")
			}
		},
	},
		{
			name:     "invalid amount (negative)",
			customer: "Alexander",
			amount:   -10.0,
			mockRepo: mockOrderRepo{},
			wantErr:  ErrInvalidAmount,
			checkMock: func(t *testing.T, m *mockOrderRepo) {
				if m.createCalled {
					t.Errorf("expected Create NOT to be called on invalid amount")
				}
			},
		},
		{
			name:     "invalid amount (zero)",
			customer: "Matthew",
			amount:   0.0,
			mockRepo: mockOrderRepo{},
			wantErr:  ErrInvalidAmount,
			checkMock: func(t *testing.T, m *mockOrderRepo) {
				if m.createCalled {
					t.Errorf("expected Create NOT to be called on zero amount")
				}
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewOrderService(&tt.mockRepo)
			ctx := context.Background()

			_, err := svc.CreateOrder(ctx, tt.customer, tt.amount)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got error %v, want %v", err, tt.wantErr)
			}
			tt.checkMock(t, &tt.mockRepo)
		})
	}
}

// TestUpdateOrderTransition проверяет статус-машину и переходы между состояниями заказа.
func TestUpdateOrderTransition(t *testing.T) {
	tests := []struct {
		name         string
		targetStatus string
		mockRepo     mockOrderRepo
		wantErr      error
		checkMock    func(t *testing.T, m *mockOrderRepo)
	}{
		{
			name:         "valid transition: pending -> processing",
			targetStatus: "processing",
			mockRepo: mockOrderRepo{
				getByIDRes:          &model.Order{ID: 1, Status: model.StatusPending},
				updateTransitionRes: nil,
			},
			wantErr: nil,
			checkMock: func(t *testing.T, m *mockOrderRepo) {
				if !m.updateTransitionCalled {
					t.Error("expected UpdateStatus to be called")
				}
			},
		},
		{
			name:         "invalid transition: pending -> completed",
			targetStatus: "completed",
			mockRepo: mockOrderRepo{
				getByIDRes: &model.Order{ID: 1, Status: model.StatusPending},
			},
			wantErr: ErrInvalidTransition,
			checkMock: func(t *testing.T, m *mockOrderRepo) {
				if m.updateTransitionCalled {
					t.Error("expected UpdateStatus NOT to be called for forbidden transition")
				}
			},
		},
		{
			name:         "invalid status value",
			targetStatus: "trash",
			mockRepo:     mockOrderRepo{},
			wantErr:      ErrNotValidTransition,
			checkMock: func(t *testing.T, m *mockOrderRepo) {
				if m.getByIDCalled || m.updateTransitionCalled {
					t.Error("expected early return for unknown status value")
				}
			},
		},
		{
			name:         "order not found",
			targetStatus: "processing",
			mockRepo: mockOrderRepo{
				getByIDErr: ErrOrderNotFound,
			},
			wantErr: ErrOrderNotFound,
			checkMock: func(t *testing.T, m *mockOrderRepo) {
				if m.updateTransitionCalled {
					t.Error("expected UpdateStatus NOT to be called when order missing")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mock := tt.mockRepo

			svc := NewOrderService(&mock)
			ctx := context.Background()

			err := svc.UpdateOrderTransition(ctx, 1, tt.targetStatus)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got error %v, want %v", err, tt.wantErr)
			}
			tt.checkMock(t, &mock)
		})
	}
}

// TestListOrders проверяет пагинацию и валидацию параметров списка заказов.
func TestListOrders(t *testing.T) {
	tests := []struct {
		name      string
		page      int
		limit     int
		mockRepo  mockOrderRepo
		wantErr   error
		checkData func(t *testing.T, items []*model.Order, total int)
		checkMock func(t *testing.T, m *mockOrderRepo)
	}{
		{name: "invalid page (zero",
			page:     0,
			limit:    20,
			mockRepo: mockOrderRepo{},
			wantErr:  ErrInvalidPage,
			checkMock: func(t *testing.T, m *mockOrderRepo) {
				if m.listOrdersCalled {
					t.Error("expected ListOrders NOT to be called for invalid page")
				}
			},
		},
		{
			name:     "invalid page (neagtive)",
			page:     -5,
			limit:    20,
			mockRepo: mockOrderRepo{},
			wantErr:  ErrInvalidPage,
			checkMock: func(t *testing.T, m *mockOrderRepo) {
				if m.listOrdersCalled {
					t.Error("expected ListOrders NOT to be called for negative page")
				}
			},
		},
		{
			name:     "invalid limit (zero)",
			page:     1,
			limit:    0,
			mockRepo: mockOrderRepo{},
			wantErr:  ErrInvalidLimit,
			checkMock: func(t *testing.T, m *mockOrderRepo) {
				if m.listOrdersCalled {
					t.Error("expected ListOrders NOT to be called for invalid limit")
				}
			},
		},
		{
			name:     "invalid limit (too large)",
			page:     1,
			limit:    101,
			mockRepo: mockOrderRepo{},
			wantErr:  ErrInvalidLimit,
			checkMock: func(t *testing.T, m *mockOrderRepo) {
				if m.listOrdersCalled {
					t.Error("expected ListOrders NOT to be called for limit > 100")
				}
			},
		},
		{
			name:  "success: page=1, limit=20",
			page:  1,
			limit: 20,
			mockRepo: mockOrderRepo{
				listOrdersRes: []*model.Order{
					{ID: 1, CustomerName: "Matthew", Amount: 100, Status: model.StatusPending},
					{ID: 2, CustomerName: "Anastasia", Amount: 250, Status: model.StatusProcessing}},
				listOrdersTotal: 2,
			},
			wantErr: nil,
			checkData: func(t *testing.T, items []*model.Order, total int) {
				if len(items) != 2 {
					t.Errorf("got %d items, want 2", len(items))
				}
				if total != 2 {
					t.Errorf("got total=%d, want 2", total)
				}
			},
			checkMock: func(t *testing.T, m *mockOrderRepo) {
				if !m.listOrdersCalled {
					t.Error("expected ListOrders to be called")
				}
			},
		},
		{
			name:  "success: empty page (offset > total)",
			page:  10,
			limit: 10,
			mockRepo: mockOrderRepo{
				listOrdersRes:   []*model.Order{},
				listOrdersTotal: 5,
			},
			wantErr: nil,
			checkData: func(t *testing.T, items []*model.Order, total int) {
				if len(items) != 0 {
					t.Errorf("expected empty items slice, got %d", len(items))
				}
				if total != 5 {
					t.Errorf("got total=%d, want 5", total)
				}
			},
			checkMock: func(t *testing.T, m *mockOrderRepo) {
				if !m.listOrdersCalled {
					t.Error("expected ListOrders to be called even for empty page")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.mockRepo
			svc := NewOrderService(&mock)
			ctx := context.Background()

			items, total, err := svc.ListOrders(ctx, tt.limit, tt.page)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got error %v, want %v", err, tt.wantErr)
			}

			if tt.checkData != nil && err == nil {
				tt.checkData(t, items, total)
			}
			if tt.checkMock != nil {
				tt.checkMock(t, &mock)
			}
		})
	}
}
