package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ThisIsTheOldGuard/payship-core/internal/domain"
	"github.com/ThisIsTheOldGuard/payship-core/internal/model"
)

type mockOrderService struct {
	createFunc           func(ctx context.Context, name string, amount float64) (*model.Order, error)
	getFunc              func(ctx context.Context, id int64) (*model.Order, error)
	listFunc             func(ctx context.Context, page, limit int) ([]*model.Order, int, error)
	updateTransitionFunc func(ctx context.Context, id int64, status string) error
}

func (m *mockOrderService) CreateOrder(ctx context.Context, name string, amount float64) (*model.Order, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name, amount)
	}
	return nil, nil
}

func (m *mockOrderService) GetOrder(ctx context.Context, id int64) (*model.Order, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockOrderService) ListOrders(ctx context.Context, page, limit int) ([]*model.Order, int, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, page, limit)
	}
	return nil, 0, nil
}

func (m *mockOrderService) UpdateOrderTransition(ctx context.Context, id int64, status string) error {
	if m.updateTransitionFunc != nil {
		return m.updateTransitionFunc(ctx, id, status)
	}
	return nil
}

func checkOrder(wantOrder *model.Order, w *httptest.ResponseRecorder) error {

	var got model.Order

	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		return fmt.Errorf("failed to decode order JSON: %v\nBody: %s", err, w.Body.String())
	}

	if got.ID != wantOrder.ID {
		return fmt.Errorf("got id %d, want %d", got.ID, wantOrder.ID)
	}
	if got.CustomerName != wantOrder.CustomerName {
		return fmt.Errorf("got customer_name %q, want %q", got.CustomerName, wantOrder.CustomerName)
	}
	if got.Amount != wantOrder.Amount {
		return fmt.Errorf("got amount %.2f, want %.2f", got.Amount, wantOrder.Amount)
	}
	if got.Status != wantOrder.Status {
		return fmt.Errorf("got status %q, want %q", got.Status, wantOrder.Status)
	}

	return nil
}

func TestCreateOrderHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		reqBody    string
		mockSvc    *mockOrderService
		wantStatus int
		wantOrder  *model.Order
		wantErr    ErrorResponse
	}{
		{
			name:    "valid request",
			reqBody: `{"customer_name":"Matthew","amount":150.50}`,
			mockSvc: &mockOrderService{
				createFunc: func(ctx context.Context, name string, amount float64) (*model.Order, error) {
					return &model.Order{
						ID:           42,
						CustomerName: name,
						Amount:       amount,
						Status:       model.StatusPending,
						CreatedAt:    time.Now(),
					}, nil
				},
			},
			wantStatus: http.StatusCreated,
			wantOrder: &model.Order{
				ID:           42,
				CustomerName: "Matthew",
				Amount:       150.50,
				Status:       model.StatusPending,
			},
		},
		{
			name:       "invalid JSON request",
			reqBody:    `{"broken":nil}`,
			mockSvc:    &mockOrderService{},
			wantStatus: http.StatusBadRequest,
			wantErr:    ErrorResponse{Err: "invalid JSON body or unknown fields"},
		},
		{
			name:    "empty customer name",
			reqBody: `{"customer_name":"","amount":150.50}`,
			mockSvc: &mockOrderService{
				createFunc: func(ctx context.Context, name string, amount float64) (*model.Order, error) {
					return nil, domain.ErrEmptyCustomer
				},
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    ErrorResponse{Err: "customer_name is required"},
		},
		{
			name:    "amount is zero",
			reqBody: `{"customer_name":"Matthew","amount":-10}`,
			mockSvc: &mockOrderService{
				createFunc: func(ctx context.Context, name string, amount float64) (*model.Order, error) {
					return nil, domain.ErrInvalidAmount
				},
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    ErrorResponse{Err: "amount must be greater than 0"},
		},
		{
			name:    "internal error",
			reqBody: `{"customer_name":"Matthew","amount":150.50}`,
			mockSvc: &mockOrderService{
				createFunc: func(ctx context.Context, name string, amount float64) (*model.Order, error) {
					return nil, errors.New("database timeout")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    ErrorResponse{Err: "internal error"},
		},
	}

	for _, tt := range tests {

		testLogger := slog.New(slog.NewTextHandler(t.Output(), nil))
		slog.SetDefault(testLogger)

		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/order", bytes.NewReader([]byte(tt.reqBody)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler := CreateOrderHandler(tt.mockSvc)

			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d\nBody: %s", w.Code, tt.wantStatus, w.Body.String())
				return
			}

			if tt.wantOrder != nil {
				err := checkOrder(tt.wantOrder, w)
				if err != nil {
					t.Error(err.Error())
				}
			}

			if tt.wantErr != (ErrorResponse{}) {
				var gotErr ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&gotErr); err != nil {
					t.Fatalf("failed to decode error JSON: %v", err)
				}
				if gotErr != tt.wantErr {
					t.Errorf("got error %+v, want %+v", gotErr, tt.wantErr)
				}
			}
		})
	}
}

func TestGetOrderHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		reqID      string
		mockSvc    *mockOrderService
		wantStatus int
		wantOrder  *model.Order
		wantErr    ErrorResponse
	}{
		{
			name:  "valid request",
			reqID: "42",
			mockSvc: &mockOrderService{
				getFunc: func(ctx context.Context, id int64) (*model.Order, error) {
					return &model.Order{
						ID:           42,
						CustomerName: "Matthew",
						Amount:       150.50,
						Status:       model.StatusPending,
						CreatedAt:    time.Now(),
					}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantOrder: &model.Order{
				ID:           42,
				CustomerName: "Matthew",
				Amount:       150.50,
				Status:       model.StatusPending,
			},
		},
		{
			name:       "invalid reqID",
			reqID:      "sorok_dva",
			mockSvc:    &mockOrderService{},
			wantStatus: http.StatusBadRequest,
			wantErr:    ErrorResponse{Err: "invalid order id"},
		},
		{
			name:  "order not found",
			reqID: "54321",
			mockSvc: &mockOrderService{
				getFunc: func(ctx context.Context, id int64) (*model.Order, error) {
					return nil, domain.ErrOrderNotFound
				},
			},
			wantStatus: http.StatusNotFound,
			wantErr:    ErrorResponse{Err: "order not found"},
		},
		{
			name:  "internal error",
			reqID: "42",
			mockSvc: &mockOrderService{
				getFunc: func(ctx context.Context, id int64) (*model.Order, error) {
					return nil, errors.New("database timeout")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    ErrorResponse{Err: "internal error"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := http.NewServeMux()
			mux.HandleFunc("GET /order/{id}", GetOrderHandler(tt.mockSvc))

			route := fmt.Sprintf("/order/%s", tt.reqID)

			req := httptest.NewRequest(http.MethodGet, route, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d\nBody: %s", w.Code, tt.wantStatus, w.Body.String())
				return
			}

			if tt.wantOrder != nil {
				err := checkOrder(tt.wantOrder, w)
				if err != nil {
					t.Error(err.Error())
				}
			}

			if tt.wantErr != (ErrorResponse{}) {
				var gotErr ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&gotErr); err != nil {
					t.Fatalf("failed to decode error JSON: %v", err)
				}
				if gotErr != tt.wantErr {
					t.Errorf("got error %+v, want %+v", gotErr, tt.wantErr)
				}
			}
		})
	}
}
