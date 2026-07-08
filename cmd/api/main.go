// Package main является точкой входа приложения payship-core.
//
// Этот пакет отвечает за оркестрацию запуска: загрузку конфигурации,
// инициализацию зависимостей (БД, репозитории, сервисы), настройку HTTP-сервера
// и обработку сигналов завершения для корректного shutdown.
//
// Пример запуска:
//
//	go run ./cmd/api
//
// или через Docker:
//
//	docker compose up
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
	"github.com/ThisIsTheOldGuard/payship-core/internal/repository"
	"github.com/ThisIsTheOldGuard/payship-core/internal/service"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// main инициализирует и запускает HTTP-сервер обработки заказов.
//
// Функция выполняет следующие шаги:
//   - Загружает переменные окружения из .env (для локальной разработки).
//   - Создаёт пул соединений с PostgreSQL с настраиваемыми лимитами.
//   - Инициализирует репозиторий, сервис и хендлеры через зависимость-инъекцию.
//   - Регистрирует маршруты в http.ServeMux.
//   - Запускает сервер в горутине и обрабатывает SIGINT/SIGTERM для graceful shutdown.
//
// Функция завершает работу программы через os.Exit при критических ошибках инициализации.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// env
	_ = godotenv.Load()

	dbCfg := LoadDBConfig()
	pool, err := NewDBPool(ctx, dbCfg)
	if err != nil {
		slog.Error("Failed to init DB", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Проверка подключения
	if err := pool.Ping(ctx); err != nil {
		slog.Error("DB ping failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to PostgreSQL", "pool_size", pool.Stat().TotalConns())

	repo := repository.NewOrderRepo(pool)
	srvCfg := LoadSrvConfig()
	orderSvc := service.NewOrderService(repo)

	mux := http.NewServeMux()

	RegisterMetrics()
	mux.HandleFunc("GET /metrics", promhttp.Handler().ServeHTTP)

	mux.HandleFunc("GET /", api.HomeHandler)
	mux.HandleFunc("POST /order", api.CreateOrderHandler(orderSvc))
	mux.HandleFunc("GET /orders", api.ListOrdersHandler(orderSvc))
	mux.HandleFunc("GET /order/{id}", api.GetOrderHandler(orderSvc))
	mux.HandleFunc("POST /order/{id}/transitions", api.UpdateOrderTransitionHandler(orderSvc))

	srv := &http.Server{
		Addr:    srvCfg.addr,
		Handler: MetricsMiddleware(mux),
	}

	// Запуск сервера
	go func() {
		slog.Info("Starting server", "address", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
		}
	}()

	// Обработка завершения / CTRL + C
	// Возможно стоит использовать https://github.com/sollniss/graceful
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutdown Server ...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited properly")
}
