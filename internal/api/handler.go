package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ThisIsTheOldGuard/payship-core/internal/model"
	"github.com/ThisIsTheOldGuard/payship-core/internal/repository"
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

func OrderHandler(repo repository.OrderRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// POST
		if r.Method != http.MethodPost {
			//http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			sendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		// Разбор Json
		var req struct {
			CustomerName string  `json:"customer_name"`
			Amount       float64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			//http.Error(w, "Bad request", http.StatusBadRequest)
			sendJSONError(w, http.StatusBadRequest, "Bad request")
			return
		}

		// Валидация
		if req.CustomerName == "" {
			sendJSONError(w, http.StatusBadRequest, "customer_name is required")
			return
		}
		if req.Amount <= 0 {
			sendJSONError(w, http.StatusBadRequest, "amount must be greater than 0")
			return
		}

		// Создание заказа
		order := &model.Order{
			CustomerName: req.CustomerName,
			Amount:       req.Amount,
			Status:       "pending",
		}

		if err := repo.Create(r.Context(), order); err != nil {
			slog.Error("Failed to create order", "error", err)
			//http.Error(w, "Internal error", http.StatusInternalServerError)
			sendJSONError(w, http.StatusInternalServerError, "Internal error")
			return
		}

		// Ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(order)
	}
}
