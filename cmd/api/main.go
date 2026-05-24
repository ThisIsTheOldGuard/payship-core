package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/ThisIsTheOldGuard/payship-core/internal/api"
	"github.com/ThisIsTheOldGuard/payship-core/internal/repository"
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

	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = "0.0.0.0:8080"
	}

	// Получаем файл конфига нашей БД
	poolCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		slog.Error("Failed to parse db config", "error", err)
		os.Exit(1)
	}

	// Ограничиваем пул для локальной разработки
	poolCfg.MaxConns = 10
	poolCfg.MinConns = 2

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

	// 2. Инициализация репозитория
	orderRepo := repository.NewOrderRepo(pool)

	// 3. Регистрация хендлеров (передаём репозиторий)
	http.HandleFunc("/", api.HomeHandler)
	http.HandleFunc("/order", api.OrderHandler(orderRepo))

	slog.Info("Starting server", "address", "http://localhost:8080")
	// 0.0.0.0 по причине работы с Ubuntu в wsl Windows
	if err := http.ListenAndServe(addr, nil); err != nil {
		slog.Error("Server failed", "error", err)
	}
}
