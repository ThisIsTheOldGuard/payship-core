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
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ThisIsTheOldGuard/payship-core/internal/domain"
	"github.com/ThisIsTheOldGuard/payship-core/internal/model"
	"github.com/ThisIsTheOldGuard/payship-core/internal/service"
)

// Для валидации также можно использовать https://github.com/go-playground/validator

// ErrorResponse - структура представления ошибки.
type ErrorResponse struct {
	Error string `json:"error"`
}

// sendJSONError отправляет стандартизированный JSON-ответ об ошибке.
//
// Вспомогательная функция устанавливает:
//   - Content-Type: application/json
//   - Указанный HTTP-статус код
//   - Тело {"error": "<сообщение>"}
//
// Параметры:
//   - w: http.ResponseWriter для отправки ответа.
//   - status: HTTP-статус код (например, 400, 404, 500).
//   - message: человекочитаемое сообщение об ошибке.
//
// Пример:
//
//	sendJSONError(w, http.StatusBadRequest, "invalid amount")
//	// Ответ: 400 {"error":"invalid amount"}
func sendJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// CreateOrderHandler создаёт хендлер для эндпоинта POST /order.
//
// Функция-фабрика принимает *service.OrderService через замыкание
// для инъекции зависимости. Возвращённый хендлер:
//   - Декодирует JSON-тело в структуру заказа.
//   - Валидирует обязательные поля (customer_name, amount).
//   - Вызывает сервис для создания заказа.
//   - Возвращает 201 Created с JSON-телом или ошибку.
//
// Параметры:
//   - svc: экземпляр сервиса для бизнес-логики.
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
			case errors.Is(err, domain.ErrEmptyCustomer), errors.Is(err, domain.ErrInvalidAmount):
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

// GetOrderHandler создаёт хендлер для эндпоинта GET /order/{id}.
//
// Функция-фабрика принимает *service.OrderService через замыкание.
// Возвращённый хендлер:
//   - Извлекает id из path-параметра через r.PathValue("id").
//   - Парсит id в int64, возвращает 400 при ошибке.
//   - Вызывает сервис для получения заказа.
//   - Возвращает 200 OK с заказом или ошибку.
//
// Параметры:
//   - svc: экземпляр сервиса для бизнес-логики.
//
// Возвращает:
//   - http.HandlerFunc: готовый хендлер для регистрации в mux.
//
// Пример:
//
//	$ curl http://localhost:8080/order/42
//	//Ответ: {"id":42,"customer_name":"Bob","amount":50.0,...}
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
			if errors.Is(err, domain.ErrOrderNotFound) {
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

// ListOrdersHandler создаёт хендлер для эндпоинта GET /orders.
//
// Функция-фабрика принимает *service.OrderService через замыкание.
// Возвращённый хендлер:
//   - Парсит query-параметры page и limit.
//   - Валидирует диапазон значений (page>=1, 1<=limit<=100).
//   - Вызывает сервис для получения страницы заказов.
//   - Возвращает 200 OK с пагинированным ответом или ошибку.
//
// Параметры:
//   - svc: экземпляр сервиса для бизнес-логики.
//
// Возвращает:
//   - http.HandlerFunc: готовый хендлер для регистрации в mux.
//
// Пример:
//
//	$ curl "http://localhost:8080/orders?page=1&limit=10"
//	//Ответ: {"items":[...],"total":150,"page":1,"limit":10}
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
			case errors.Is(err, domain.ErrOrderNotFound),
				errors.Is(err, domain.ErrInvalidPage),
				errors.Is(err, domain.ErrInvalidLimit):
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
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}
}

// UpdateOrderTransitionHandler создаёт хендлер для эндпоинта
// POST /order/{id}/transitions.
//
// Функция-фабрика принимает *service.OrderService через замыкание.
// Возвращённый хендлер:
//   - Извлекает id из path-параметра и парсит в int64.
//   - Декодирует JSON-тело {"name": "new_status"}.
//   - Валидирует значение статуса через сервис.
//   - Вызывает сервис для обновления с проверкой статус-машины.
//   - Возвращает 200 OK при успехе или ошибку.
//
// Параметры:
//   - svc: экземпляр сервиса для бизнес-логики.
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
			Status string `json:"status"`
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&transition); err != nil {
			sendJSONError(w, http.StatusBadRequest, "invalid JSON body or unknown fields")
			return
		}

		if err := svc.UpdateOrderTransition(r.Context(), id, transition.Status); err != nil {
			switch {
			case errors.Is(err, domain.ErrNotValidTransition),
				errors.Is(err, domain.ErrInvalidTransition):
				sendJSONError(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, domain.ErrOrderNotFound):
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
