package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ThisIsTheOldGuard/payship-core/internal/model"
	"github.com/ThisIsTheOldGuard/payship-core/internal/repository"
)

// HomeHandler Обрабатывает запросы на главную страницу
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Request received", "method", r.Method, "path", r.URL.Path)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello! You visited the home page."))
}

func OrderHandler(repo repository.OrderRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			CustomerName string  `json:"customer_name"`
			Amount       float64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		order := &model.Order{
			CustomerName: req.CustomerName,
			Amount:       req.Amount,
			Status:       "pending",
		}

		if err := repo.Create(r.Context(), order); err != nil {
			slog.Error("Failed to create order", "error", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(order)
	}
}
