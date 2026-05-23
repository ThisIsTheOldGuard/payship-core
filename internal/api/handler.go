package api

import (
	"log/slog"
	"net/http"
)

// HomeHandler Обрабатывает запросы на главную страницу
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Request received", "method", r.Method, "path", r.URL.Path)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello! You visited the home page."))
}
