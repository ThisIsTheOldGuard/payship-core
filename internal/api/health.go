package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Вовзможно в будущем надо сделать ответ в виде перечисления одного из внутренних статусов
type HealthResponse struct {
	Status string `json:"status"`
}

// sendJSONError отправляет стандартизированный JSON-ответ о статусе сервиса.
//
// Вспомогательная функция устанавливает:
//   - Content-Type: application/json
//   - Указанный HTTP-статус код
//   - Тело {"status": "<сообщение>"}
//
// Параметры:
//   - w: http.ResponseWriter для отправки ответа.
//   - status: HTTP-статус код (например, 200).
//   - message: человекочитаемое сообщение о статусе.
//
// Пример:
//
//	sendJSONHealth(w, http.StatusOK, "Ready")
//	// Ответ: {"status":"Ready"}
func sendJSONHealth(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(HealthResponse{Status: message})
}

// RootHandler обрабатывает запросы на корневой путь "/".
//
// Метод возвращает простой текстовый ответ для проверки
// работоспособности сервера в формате HealthResponse
//
// Пример:
//
//	$ curl http://localhost:8080/
//	//Ответ: {"status":"API is running"}
func RootHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sendJSONHealth(w, http.StatusOK, "API is running")
	}
}

// HomeHandler обрабатывает все неподходящие запросы.
//
// Метод возвращает ошибку недоступности страницы в представлении ErrorResponse.
//
// Пример:
//
//	$ curl http://localhost:8080/nonexistent-page
//	//Ответ: {"error":"Page not found"}
func NotFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sendJSONError(w, http.StatusNotFound, "Page not found")
	}
}

// HealthHandler проверяет доступ к API.
//
// # Метод возвращает доступность сервиса в формате HealthResponse
//
// Пример:
//
//	$ curl http://localhost:8080/health
//	//Ответ: {"status":"OK"}
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sendJSONHealth(w, http.StatusOK, "OK")
	}
}

// ReadyHandler проверяет доступ к БД.
//
// # Метод возвращает состояние подключение к БД в формате HealthResponse
//
// Параметры:
//   - pool: pgxpool.Pool база данных к которой подключено приложение
//
// Пример:
//
//	$ curl http://localhost:8080/ready
//	//Ответ: {"status":"Ready"}
func ReadyHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			slog.Error("Readiness check failed", "error", err)
			sendJSONError(w, http.StatusServiceUnavailable, "Database is unavailable")
			return
		}

		sendJSONHealth(w, http.StatusOK, "Ready")

	}
}
