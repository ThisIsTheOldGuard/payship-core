package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ThisIsTheOldGuard/payship-core/internal/api"
	"github.com/ThisIsTheOldGuard/payship-core/internal/repository"
	"github.com/ThisIsTheOldGuard/payship-core/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// env
	err := godotenv.Load()
	if err != nil {
		slog.Error("Error loading .env file", "error", err)
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://admin:secret@localhost:5432/payship_core?sslmode=disable"
	}

	// Получаем файл конфига нашей БД
	poolCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		slog.Error("Failed to parse db config", "error", err)
		os.Exit(1)
	}

	// Ограничиваем пул для локальной разработки
	MaxConnsStr := os.Getenv("MaxConns")
	MaxConnsInt, err := strconv.Atoi(MaxConnsStr)
	if err != nil {
		slog.Info("Ошибка при конвертации: %v. Будет использовано значение по умолчанию.\n", err)
		MaxConnsInt = 10 // значение по умолчанию
	}
	poolCfg.MaxConns = int32(MaxConnsInt)

	MinConnsStr := os.Getenv("MinConns")
	MinConnsInt, err := strconv.Atoi(MinConnsStr)
	if err != nil {
		slog.Info("Ошибка при конвертации: %v. Будет использовано значение по умолчанию.\n", err)
		MinConnsInt = 2 // значение по умолчанию
	}
	poolCfg.MinConns = int32(MinConnsInt)

	// Создаем пул базы данных на основе настроек
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		slog.Error("Failed to connect to db", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Проверка подключения
	if err := pool.Ping(ctx); err != nil {
		slog.Error("Database ping failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to PostgreSQL", "pool_size", pool.Stat().TotalConns())

	// Инициализация репозитория
	orderRepo := repository.NewOrderRepo(pool)

	// Создание сервера
	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = "0.0.0.0:8080"
	}

	mux := http.NewServeMux()

	orderSvc := service.NewOrderService(orderRepo)

	mux.HandleFunc("GET /", api.HomeHandler)
	mux.HandleFunc("POST /order", api.CreateOrderHandler(orderSvc))
	mux.HandleFunc("GET /orders", api.ListOrdersHandler(orderSvc))
	mux.HandleFunc("GET /order/{id}", api.GetOrderHandler(orderSvc))
	mux.HandleFunc("POST /order/{id}/transitions", api.UpdateOrderTransitionHandler(orderSvc))

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
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
