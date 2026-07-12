package main

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// httpRequestsTotal - Prometheus-счётчик общего количества HTTP-запросов.
//
// Метрика увеличивается на 1 после завершения каждого запроса.
// Позволяет отслеживать пропускную способность сервиса в разрезе:
//   - method: HTTP-метод (GET, POST, PUT, DELETE и т.д.)
//   - path:   маршрут запроса (или шаблон маршрута r.Pattern)
//   - status: HTTP-код ответа (200, 404, 500 и т.д.)
//
// Тип: prometheus.CounterVec
// Имя метрики: "http_requests_total"
var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests received.",
	},
	[]string{"method", "path", "status"},
)

// httpRequestDuration - Prometheus-гистограмма времени обработки HTTP-запросов.
//
// Измеряет латентность каждого запроса в секундах и раскладывает значения
// по предопределённым бакетам (от 1 мс до 10 с). Позволяет строить
// графики перцентилей (p50, p95, p99) и оценивать SLA.
//
// Позволяет отслеживать время обработки конечных точек сервиса в разрезе:
//   - method: HTTP-метод
//   - path:   маршрут запроса
//   - status: HTTP-код ответа
//
// Бакеты: 1ms, 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s
//
// Тип: prometheus.HistogramVec
// Имя метрики: "http_request_duration_seconds"
var httpRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	},
	[]string{"method", "path", "status"},
)

// statusRecorder - обёртка над http.ResponseWriter, перехватывающая HTTP-статус ответа.
//
// Поля:
//   - http.ResponseWriter: встроенный оригинальный writer (делегирование).
//   - statusCode:          перехваченный HTTP-статус код.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader перехватывает HTTP-статус код и сохраняет его в statusRecorder.
//
// Переопределяет метод WriteHeader встроенного http.ResponseWriter.
// Перед делегированием вызова оригинальному writer'у, записывает
// полученный statusCode в поле r.statusCode.
//
// Параметры:
//   - statusCode: HTTP-статус код (например, 200, 404, 500).
func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// RegisterMetrics регистрирует все метрики в стандартном реестре.
//
// Вызывает prometheus.MustRegister для каждой метрики пакета.
//
// Должна быть вызвана один раз при старте приложения (обычно в init() или main()),
// до начала обработки HTTP-запросов.
//
// Регистрируемые метрики:
//   - httpRequestsTotal    (http_requests_total)
//   - httpRequestDuration  (http_request_duration_seconds)
func RegisterMetrics() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

// MetricsMiddleware - HTTP-middleware для сбора метрик и логирования запросов.
//
// Оборачивает следующий http.Handler и для каждого запроса:
//  1. Пропускает служебные пути ("/metrics", "/favicon.ico") без метрик.
//  2. Определяет path: использует шаблон маршрута,
//     если он задан, иначе - реальный URL-путь. Это предотвращает
//     взрыв кардинальности метрик из-за динамических параметров
//     (например, "/users/123" → "/users/{id}").
//  3. Обрабатывает паники: Возвращает 500 и сообщение об ошибке.
//  4. Заменяет path на "not_found" для статусов 404, чтобы не засорять
//     метрики уникальными несуществующими путями.
//  5. Вычисляет метрику httpRequestsTotal.
//  6. Записывает длительность в гистограмму httpRequestDuration.
//
// Параметры:
//   - next: следующий http.Handler в цепочке (обычно - корневой роутер).
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/metrics" || r.URL.Path == "/favicon.ico" {
			next.ServeHTTP(w, r)
			return
		}

		path := r.URL.Path
		if r.Pattern != "" {
			path = r.Pattern
		}

		slog.Info("HTTP Request Processed", "method", r.Method, "path", path)

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		defer func() {

			// Если PANIC
			if err := recover(); err != nil {
				rec.statusCode = http.StatusInternalServerError

				slog.Error("HTTP Request PANIC",
					"panic_error", err,
					"method", r.Method,
					"path", path,
				)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}

			if rec.statusCode == http.StatusNotFound {
				path = "not_found"
			}

			duration := time.Since(start).Seconds()

			statusStr := strconv.Itoa(rec.statusCode)
			httpRequestsTotal.WithLabelValues(r.Method, path, statusStr).Inc()
			httpRequestDuration.WithLabelValues(r.Method, path, statusStr).Observe(duration)

			slog.Info("HTTP Request Done",
				"method", r.Method,
				"path", path,
				"status", rec.statusCode,
				"duration_sec", duration,
			)
		}()

		next.ServeHTTP(rec, r)

	})
}
