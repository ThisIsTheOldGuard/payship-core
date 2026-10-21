package main

import (
	"net/http"

	"github.com/ThisIsTheOldGuard/payship-core/internal/api"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"
)

// RateLimitMiddleware ограничивает частоту запросов к API.
func RateLimitMiddleware(next http.Handler, limiter *rate.Limiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			api.SendJSONError(w, http.StatusTooManyRequests, "too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RateLimitMiddleware активирует тестовый хэндлер для тестирования задержек БД.
func InitTestHandlers(mux *http.ServeMux, repo *pgxpool.Pool) {
	TestCfg := LoadTestConfig()
	if TestCfg.IsTest {
		mux.HandleFunc("GET /debug/db/slow", api.Debug_DB_Slow(repo))
	}
}
