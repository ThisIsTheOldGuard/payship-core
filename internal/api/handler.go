package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ThisIsTheOldGuard/payship-core/internal/model"
	"github.com/ThisIsTheOldGuard/payship-core/internal/service"
)

// Для валидации также можно использовать https://github.com/go-playground/validator

type ErrorResponse struct {
	Error string `json:"error"`
}

// sendJSONError — вспомогательная функция для единого формата ошибок
func sendJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// HomeHandler Обрабатывает запросы на главную страницу
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Request received", "method", r.Method, "path", r.URL.Path)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello! You visited the home page."))
}

func CreateOrderHandler(svc *service.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Разбор Json
		var req struct {
			CustomerName string  `json:"customer_name"`
			Amount       float64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		order, err := svc.CreateOrder(r.Context(), req.CustomerName, req.Amount)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrEmptyCustomer), errors.Is(err, service.ErrInvalidAmount):
				sendJSONError(w, http.StatusBadRequest, err.Error())
			default:
				slog.Error("Failed to create order", "error", err)
				sendJSONError(w, http.StatusInternalServerError, "internal error")
			}
			return
		}

		// Ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(order)

	}
}

func GetOrderHandler(svc *service.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			sendJSONError(w, http.StatusBadRequest, "invalid order id")
			return
		}

		order, err := svc.GetOrder(r.Context(), id)
		if err != nil {
			if errors.Is(err, service.ErrOrderNotFound) {
				sendJSONError(w, http.StatusNotFound, err.Error())
				return
			}
			slog.Error("Failed to get order", "error", err)
			sendJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(order)
	}
}

func ListOrdersHandler(svc *service.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := 1
		limit := 20
		if p := r.URL.Query().Get("page"); p != "" {
			val, err := strconv.Atoi(p)
			if err != nil {
				sendJSONError(w, http.StatusBadRequest, "invalid page value")
			}
			page = val
		}
		if l := r.URL.Query().Get("limit"); l != "" {
			val, err := strconv.Atoi(l)
			if err != nil {
				sendJSONError(w, http.StatusBadRequest, "invalid limit value")
			}
			limit = val
		}

		orders, total, err := svc.ListOrders(r.Context(), limit, page)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrOrderNotFound),
				errors.Is(err, service.ErrInvalidPage),
				errors.Is(err, service.ErrInvalidLimit):
				sendJSONError(w, http.StatusNotFound, err.Error())
			default:
				slog.Error("Failed to list orders", "error", err)
				sendJSONError(w, http.StatusInternalServerError, "internal error")
			}
			return
		}

		response := &model.OrderListResponse{
			Items: orders,
			Total: total,
			Page:  page,
			Limit: limit,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
}

func UpdateOrderTransitionHandler(svc *service.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			sendJSONError(w, http.StatusBadRequest, "invalid order id")
			return
		}

		// заготовка на будущее, например дял создания новой таблицы status с id
		var transition struct {
			Name string `json:"name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&transition); err != nil {
			sendJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		if err := svc.UpdateOrderTransition(r.Context(), id, transition.Name); err != nil {
			switch {
			case errors.Is(err, service.ErrNotValidTransition),
				errors.Is(err, service.ErrInvalidTransition):
				sendJSONError(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, service.ErrOrderNotFound):
				sendJSONError(w, http.StatusNotFound, err.Error())
			default:
				slog.Error("Failed to update status order", "error", err)
				sendJSONError(w, http.StatusInternalServerError, "internal error")
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)

	}
}
