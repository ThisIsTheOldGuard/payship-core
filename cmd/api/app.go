package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewDBPool создаёт и инициализирует пул соединений с PostgreSQL.
//
// Функция парсит конфигурацию из DBConfig, устанавливает лимиты соединений
// (MaxConns/MinConns) и возвращает готовый к работе *pgxpool.Pool.
// Ошибки оборачиваются с контекстом для отладки.
//
// Параметры:
//   - ctx: контекст для отмены операции создания пула.
//   - cfg: конфигурация подключения к БД.
//
// Возвращает:
//   - *pgxpool.Pool: пул соединений при успехе.
//   - error: ошибка парсинга конфигурации или подключения.
//
// Пример:
//
//	pool, err := NewDBPool(ctx, dbCfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer pool.Close()
func NewDBPool(ctx context.Context, cfg *DBConfig) (*pgxpool.Pool, error) {
	// Конфиг БД
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Ограничения
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns

	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime

	// Создание пула
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	return pool, nil
}
