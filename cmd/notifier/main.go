package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThisIsTheOldGuard/payship-core/internal/api"
	"github.com/ThisIsTheOldGuard/payship-core/internal/config"
	"github.com/ThisIsTheOldGuard/payship-core/internal/database"
	"github.com/ThisIsTheOldGuard/payship-core/internal/kafka" // 👈 Инфраструктура
	"github.com/ThisIsTheOldGuard/payship-core/internal/metrics"
	"github.com/ThisIsTheOldGuard/payship-core/internal/model"
	"github.com/ThisIsTheOldGuard/payship-core/internal/notifier" // 👈 Бизнес-логика
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	_ = godotenv.Load()

	dbCfg := config.LoadDBConfig()
	KafkaCfg := config.LoadKafkaConfig()
	NotifierCfg := config.LoadNotifierConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := database.NewDBPool(ctx, dbCfg)
	if err != nil {
		slog.Error("Failed to init DB", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	prometheus.MustRegister(metrics.NewDBPoolCollector(pool))

	inDocker := config.IsRunningInDocker()
	reader := kafka.NewReader(
		[]string{KafkaCfg.Addr},
		model.TopicOrderCreated,
		"notifier-service-group-v1",
		inDocker,
	)

	defer func() {
		if err := reader.Close(); err != nil {
			slog.Error("Failed to close Kafka reader", "error", err)
		}
	}()

	consumer := notifier.NewConsumer(reader, pool, metrics.PrometheusDBMetric{})
	slog.Info("Notifier service initialized", "topic", model.TopicOrderCreated)

	go consumer.Start(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /metrics", promhttp.Handler().ServeHTTP)
	mux.HandleFunc("GET /health", api.HealthHandler())
	mux.HandleFunc("GET /ready", api.ReadyHandler(pool))

	srv := &http.Server{
		Addr:    NotifierCfg.Addr,
		Handler: mux,
	}

	metrics.RegisterDBMetrics()

	go func() {
		slog.Info("Notifier metrics server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Metrics server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutdown Notifier ...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Worker forced to shutdown", "error", err)
	}

	pool.Close()
	slog.Info("Notifier exited properly")
}
