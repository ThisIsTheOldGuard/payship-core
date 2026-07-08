package main

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests received.",
	},
	[]string{"method", "path", "status"},
)

var httpRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	},
	[]string{"method", "path", "status"},
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func RegisterMetrics() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

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
