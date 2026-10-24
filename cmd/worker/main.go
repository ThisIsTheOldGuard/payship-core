package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThisIsTheOldGuard/payship-core/internal/api"
	"github.com/ThisIsTheOldGuard/payship-core/internal/config" // Твой пакет конфигов
	"github.com/ThisIsTheOldGuard/payship-core/internal/database"
	"github.com/ThisIsTheOldGuard/payship-core/internal/metrics"
	"github.com/ThisIsTheOldGuard/payship-core/internal/worker"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	_ = godotenv.Load()

	dbCfg := config.LoadDBConfig()
	WorkerCfg := config.LoadWorkerConfig()

	pool, err := database.NewDBPool(ctx, dbCfg)
	if err != nil {
		slog.Error("Failed to init DB", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	prometheus.MustRegister(metrics.NewDBPoolCollector(pool))

	inDocker := config.IsRunningInDocker()
	resolver := &worker.SmartResolver{ThisDocker: inDocker}

	slog.Info("Check connect Kafka...")
	dialer := &kafka.Dialer{
		Timeout:   5 * time.Second,
		DualStack: true,
		Resolver:  resolver,
	}

	conn, err := dialer.Dial("tcp", WorkerCfg.Kafka.Addr)
	if err != nil {
		slog.Error("FAILED to connect Kafka", "addr", WorkerCfg.Kafka.Addr, "error", err)
		os.Exit(1)
	}
	if err := conn.Close(); err != nil {
		slog.Error("Failed to close check connection", "error", err)
	}

	slog.Info("Successfully connected to Kafka")

	customDialContext := func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err == nil && !inDocker && host == "payship-kafka" && port == "9092" {
			address = net.JoinHostPort("127.0.0.1", "9094")
		}
		d := &net.Dialer{Timeout: 5 * time.Second}
		return d.DialContext(ctx, network, address)
	}

	producer := &kafka.Writer{
		Addr:         kafka.TCP(WorkerCfg.Kafka.Addr),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Transport: &kafka.Transport{
			Dial: customDialContext,
		},
	}

	defer func() {
		if err := producer.Close(); err != nil {
			slog.Error("Failed to close Kafka producer", "error", err)
		}
	}()

	relay := worker.NewOutboxRelayer(pool, producer, WorkerCfg.Kafka.BatchSize, metrics.PrometheusDBMetric{})
	//relay := worker.NewOutboxRelayer(pool, worker.NewMockProducer(), WorkerCfg.Kafka.BatchSize, metrics.PrometheusDBMetric{})
	slog.Info("Starting Outbox Relay Worker...")

	go relay.Run(ctx)

	metrics.RegisterDBMetrics()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /metrics", promhttp.Handler().ServeHTTP)
	mux.HandleFunc("GET /health", api.HealthHandler())
	mux.HandleFunc("GET /ready", api.ReadyHandler(pool))

	srv := &http.Server{
		Addr:    WorkerCfg.Worker.Addr,
		Handler: mux,
	}

	go func() {
		slog.Info("Worker metrics server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Metrics server failed", "error", err)
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutdown Worker ...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Worker forced to shutdown", "error", err)
	}

	slog.Info("Worker exited properly")

}
