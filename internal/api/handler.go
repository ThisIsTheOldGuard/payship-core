// Package api предоставляет HTTP-хендлеры для микросервиса заказов.
//
// Этот пакет отвечает за:
//   - Парсинг HTTP-запросов (JSON, path/query-параметры).
//   - Валидацию синтаксиса входных данных.
//   - Маппинг ошибок сервиса в HTTP-статусы и JSON-ответы.
//   - Логирование через slog.
//
// Хендлеры не содержат бизнес-логики и делегируют её сервисному слою.
//
// Пример использования:
//
//	mux.HandleFunc("POST /order", api.CreateOrderHandler(svc))
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ThisIsTheOldGuard/payship-core/internal/domain"
	"github.com/ThisIsTheOldGuard/payship-core/internal/model"
)

// Для валидации также можно использовать https://github.com/go-playground/validator

// CreateOrderHandler создаёт хендлер для эндпоинта POST /order.
//
// Функция-фабрика принимает service.OrderService через замыкание
// для инъекции зависимости. Возвращённый хендлер:
//   - Декодирует JSON-тело в структуру заказа.
//   - Валидирует обязательные поля (customer_name, amount).
//   - Вызывает сервис для создания заказа.
//   - Возвращает 201 Created с JSON-телом или ошибку.
//
// Параметры:
//   - *Dependencies: структура зависимостей
//
// Возвращает:
//   - http.HandlerFunc: готовый хендлер для регистрации в mux.
//
// Пример:
//
//	$ curl -X POST http://localhost:8080/order \
//	  -H "Content-Type: application/json" \
//	  -d '{"customer_name":"Alice","amount":100}'
//	//Ответ: {"id":1,"customer_name":"Alice","amount":100,"status":"pending",...}
func CreateOrderHandler(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Разбор Json
		var req struct {
			CustomerName string  `json:"customer_name"`
			Amount       float64 `json:"amount"`
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&req); err != nil {
			SendJSONError(w, http.StatusBadRequest, "invalid JSON body or unknown fields")
			return
		}

		order, err := deps.Svc.CreateOrder(r.Context(), req.CustomerName, req.Amount)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrEmptyCustomer), errors.Is(err, domain.ErrInvalidAmount):
				SendJSONError(w, http.StatusBadRequest, err.Error())
			default:
				slog.Error("Failed to create order", "error", err)
				SendJSONError(w, http.StatusInternalServerError, "internal error")
			}
			return
		}

		// Cache
		cacheKey := fmt.Sprintf("order:id:%d", order.ID)
		_ = deps.Cache.Delete(r.Context(), cacheKey)
		_ = deps.Cache.Delete(r.Context(), "orders:page:1:limit:10")
		_ = deps.Cache.Delete(r.Context(), "orders:page:1:limit:20")

		// Ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(order)

	}
}

// GetOrderHandler создаёт хендлер для эндпоинта GET /order/{id}.
//
// Функция-фабрика принимает service.OrderService через замыкание.
// Возвращённый хендлер:
//   - Извлекает id из path-параметра через r.PathValue("id").
//   - Парсит id в int64, возвращает 400 при ошибке.
//   - Вызывает сервис для получения заказа.
//   - Возвращает 200 OK с заказом или ошибку.
//
// Параметры:
//   - *Dependencies: структура зависимостей
//
// Возвращает:
//   - http.HandlerFunc: готовый хендлер для регистрации в mux.
//
// Пример:
//
//	$ curl http://localhost:8080/order/42
//	//Ответ: {"id":42,"customer_name":"Bob","amount":50.0,...}
func GetOrderHandler(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			SendJSONError(w, http.StatusBadRequest, "invalid order id")
			return
		}

		cacheKey := fmt.Sprintf("order:id:%d", id)
		if cachedData, err := deps.Cache.Get(r.Context(), cacheKey); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "max-age=30, public")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(cachedData))

			return
		}

		order, err := deps.Svc.GetOrder(r.Context(), id)
		if err != nil {
			if errors.Is(err, domain.ErrOrderNotFound) {
				SendJSONError(w, http.StatusNotFound, err.Error())
				return
			}
			slog.Error("Failed to get order", "error", err)
			SendJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}

		jsonData, _ := json.Marshal(order)
		_ = deps.Cache.Set(r.Context(), cacheKey, string(jsonData), 30*time.Second)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(order)
	}
}

// ListOrdersHandler создаёт хендлер для эндпоинта GET /orders.
//
// Функция-фабрика принимает service.OrderService через замыкание.
// Возвращённый хендлер:
//   - Парсит query-параметры page и limit.
//   - Валидирует диапазон значений (page>=1, 1<=limit<=100).
//   - Вызывает сервис для получения страницы заказов.
//   - Возвращает 200 OK с пагинированным ответом или ошибку.
//
// Параметры:
//   - *Dependencies: структура зависимостей
//
// Возвращает:
//   - http.HandlerFunc: готовый хендлер для регистрации в mux.
//
// Пример:
//
//	$ curl "http://localhost:8080/orders?page=1&limit=10"
//	//Ответ: {"items":[...],"total":150,"page":1,"limit":10}
func ListOrdersHandler(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		page := 1
		limit := 10
		if p := r.URL.Query().Get("page"); p != "" {
			val, err := strconv.Atoi(p)
			if err != nil {
				SendJSONError(w, http.StatusBadRequest, "invalid page value")
			}
			page = val
		}
		if l := r.URL.Query().Get("limit"); l != "" {
			val, err := strconv.Atoi(l)
			if err != nil {
				SendJSONError(w, http.StatusBadRequest, "invalid limit value")
			}
			limit = val
		}

		cacheKey := fmt.Sprintf("orders:page:%d:limit:%d", page, limit)

		if cachedData, err := deps.Cache.Get(r.Context(), cacheKey); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "max-age=30, public")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(cachedData))

			return
		}

		orders, total, err := deps.Svc.ListOrders(r.Context(), limit, page)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrOrdersNotFound),
				errors.Is(err, domain.ErrInvalidPage),
				errors.Is(err, domain.ErrInvalidLimit):
				SendJSONError(w, http.StatusNotFound, err.Error())
			default:
				slog.Error("Failed to list orders", "error", err)
				SendJSONError(w, http.StatusInternalServerError, "internal error")
			}
			return
		}

		response := &model.OrderListResponse{
			Items: orders,
			Total: total,
			Page:  page,
			Limit: limit,
		}

		jsonData, _ := json.Marshal(response)
		_ = deps.Cache.Set(r.Context(), cacheKey, string(jsonData), 30*time.Second)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}
}

// UpdateOrderTransitionHandler создаёт хендлер для эндпоинта
// POST /order/{id}/transitions.
//
// Функция-фабрика принимает service.OrderService через замыкание.
// Возвращённый хендлер:
//   - Извлекает id из path-параметра и парсит в int64.
//   - Декодирует JSON-тело {"name": "new_status"}.
//   - Валидирует значение статуса через сервис.
//   - Вызывает сервис для обновления с проверкой статус-машины.
//   - Возвращает 200 OK при успехе или ошибку.
//
// Параметры:
//   - *Dependencies: структура зависимостей
//
// Возвращает:
//   - http.HandlerFunc: готовый хендлер для регистрации в mux.
//
// Пример:
//
//	$ curl -X POST http://localhost:8080/order/1/transitions \
//	  -H "Content-Type: application/json" \
//	  -d '{"status":"processing"}'
//	//Ответ: 201
func UpdateOrderTransitionHandler(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			SendJSONError(w, http.StatusBadRequest, "invalid order id")
			return
		}

		// заготовка на будущее, например дял создания новой таблицы status с id
		var transition struct {
			Status string `json:"status"`
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&transition); err != nil {
			SendJSONError(w, http.StatusBadRequest, "invalid JSON body or unknown fields")
			return
		}

		if err := deps.Svc.UpdateOrderTransition(r.Context(), id, transition.Status); err != nil {
			switch {
			case errors.Is(err, domain.ErrNotValidTransition),
				errors.Is(err, domain.ErrInvalidTransition):
				SendJSONError(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, domain.ErrOrderNotFound):
				SendJSONError(w, http.StatusNotFound, err.Error())
			default:
				slog.Error("Failed to update status order", "error", err)
				SendJSONError(w, http.StatusInternalServerError, "internal error")
			}
			return
		}

		cacheKey := fmt.Sprintf("order:id:%d", id)
		_ = deps.Cache.Delete(r.Context(), cacheKey)
		_ = deps.Cache.Delete(r.Context(), "orders:page:1:limit:10")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)

	}
}
