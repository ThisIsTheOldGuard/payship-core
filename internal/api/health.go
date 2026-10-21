package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Pinger -  интерфейс для проверки доступности БД.
type Pinger interface {
	Ping(context.Context) error
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
		SendJSONError(w, http.StatusNotFound, "Page not found")
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
func ReadyHandler(p Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*2)
		defer cancel()

		if err := p.Ping(ctx); err != nil {
			slog.Error("Readiness check failed", "error", err)
			SendJSONError(w, http.StatusServiceUnavailable, "Database is unavailable")
			return
		}

		sendJSONHealth(w, http.StatusOK, "Ready")

	}
}
