package main

import (
	"log/slog"
	"os"
	"strconv"
	"time"
)

// DBConfig содержит параметры подключения к базе данных.
//
// Поля:
//   - URL: строка подключения в формате PostgreSQL DSN.
//   - MaxConns: максимальное количество соединений в пуле.
//   - MinConns: минимальное количество соединений в пуле.
//
// Значения по умолчанию задаются через os.Getenv с фоллбэком.
type DBConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnIdleTime time.Duration
	MaxConnLifetime time.Duration
}

// SrvConfig содержит параметры запуска HTTP-сервера.
//
// Поля:
//   - addr: адрес и порт (например, "0.0.0.0:8080").
type SrvConfig struct {
	addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	RateLimitRPS           float64
	RateLimitBurst         int
	RateLimitFallbackRPS   float64
	RateLimitFallbackBurst int
}

// TestConfig содержит параметры запуска HTTP-сервера.
//
// Поля:
//   - IsTest: определяет вариант работы приложения (например, "true").
type TestConfig struct {
	IsTest bool
}

// DBConfig содержит параметры подключения к базе данных.
//
// Поля:
//   - addr: адрес и порт.
//   - PoolSize: максимальное количество соединений в пуле.
//
// Значения по умолчанию задаются через os.Getenv с фоллбэком.
type RedisConfig struct {
	addr     string
	PoolSize int
}

// LoadDBConfig загружает конфигурацию БД из переменных окружения.
//
// Функция читает DB_URL, MAX_CONNS, MIN_CONNS через os.Getenv.
// При отсутствии или ошибке парсинга используются значения по умолчанию:
//   - DB_URL: путь к базе данных (по умолчанию "postgres://admin:secret@localhost:5432/payship_core?sslmode=disable")
//   - MAX_CONNS: максимальный размер пула (по умолчанию "20")
//   - MIN_CONNS: минимальный размер пула (по умолчанию "5")
//   - DB_MAX_CONN_IDLE_MINUTES: таймаут для простаивающего соединения (по умолчанию "5m")
//   - DB_MAX_CONN_LIFE_HOURS: таймаут для открытого соединения (по умолчанию "1h")
//
// Возвращает заполненную *DBConfig для передачи в NewDBPool.
func LoadDBConfig() *DBConfig {

	return &DBConfig{
		URL:             getStrEnv("DB_URL", "postgres://admin:secret@localhost:5432/payship_core?sslmode=disable"),
		MaxConns:        getInt32Env("MAX_CONNS", 20),
		MinConns:        getInt32Env("MIN_CONNS", 5),
		MaxConnIdleTime: parseDurationEnv("DB_MAX_CONN_IDLE_MINUTES", 5*time.Minute),
		MaxConnLifetime: parseDurationEnv("DB_MAX_CONN_LIFE_HOURS", 1*time.Hour),
	}
}

// LoadRedisConfig загружает конфигурацию БД из переменных окружения.
//
// Функция читает REDIS_ADDR, REDIS_POOL_SIZE через os.Getenv.
// При отсутствии или ошибке парсинга используются значения по умолчанию:
//   - REDIS_ADDR: адрес сервиса Redis (по умолчанию "0.0.0.0:6379")
//   - REDIS_POOL_SIZE: максимальное количество соединений в пуле (по умолчанию "10")
//
// Возвращает заполненную *RedisConfig для передачи в cache.NewRedisCache.
func LoadRedisConfig() *RedisConfig {
	return &RedisConfig{
		addr:     getStrEnv("REDIS_ADDR", "0.0.0.0:6379"),
		PoolSize: getIntEnv("REDIS_POOL_SIZE", 10),
	}
}

// LoadSrvConfig загружает конфигурацию HTTP-сервера и ограничений трафика из переменных окружения.
//
// Функция читает настройки адреса, таймаутов и rate limiting через os.Getenv.
// При отсутствии или ошибке парсинга используются значения по умолчанию:
//   - SERVER_ADDR: адрес и порт для прослушивания (по умолчанию "0.0.0.0:8080").
//   - SERVER_READ_TIMEOUT: максимальное время на чтение заголовков и тела запроса (по умолчанию "5s").
//   - HTTP_WRITE_TIMEOUT: максимальное время на формирование и отправку ответа (по умолчанию "10s").
//   - HTTP_IDLE_TIMEOUT: максимальное время ожидания следующего запроса на keep-alive соединении (по умолчанию "120s").
//   - RATE_LIMIT_RPS: нормальная пропускная способность в запросах в секунду (по умолчанию 100).
//   - RATE_LIMIT_BURST: допустимый всплеск запросов в нормальном режиме (по умолчанию 200).
//   - RATE_LIMIT_FALLBACK_RPS: пропускная способность при недоступности кэша (по умолчанию 30).
//   - RATE_LIMIT_FALLBACK_BURST: допустимый всплеск запросов в режиме fallback (по умолчанию 50).
//
// Возвращает заполненную *SrvConfig для инициализации http.Server и middleware.
func LoadSrvConfig() *SrvConfig {
	return &SrvConfig{
		addr:                   getStrEnv("SERVER_ADDR", "0.0.0.0:8080"),
		ReadTimeout:            parseDurationEnv("SERVER_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:           parseDurationEnv("HTTP_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:            parseDurationEnv("HTTP_IDLE_TIMEOUT", 120*time.Second),
		RateLimitRPS:           getFloat64Env("RATE_LIMIT_RPS", 100),
		RateLimitBurst:         getIntEnv("RATE_LIMIT_BURST", 200),
		RateLimitFallbackRPS:   getFloat64Env("RATE_LIMIT_FALLBACK_RPS", 30), // Снижаем в 3 раза
		RateLimitFallbackBurst: getIntEnv("RATE_LIMIT_FALLBACK_BURST", 50),
	}
}

// LoadTestConfig загружает настройки для определения формата работы приложения
//
// Возвращает *TestConfig с параметрами тестовых данных
func LoadTestConfig() *TestConfig {
	return &TestConfig{parseBoolEnv("IsTest", true)}
}

// getStrEnv получает строковое значение из окружения.
func getStrEnv(key string, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// getIntEnv парсит значение в int.
func getIntEnv(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
		slog.Warn("Invalid env var (expected int)", "key", key, "value", v)
	}
	return defaultVal
}

// getInt32Env парсит значение в int32.
func getInt32Env(key string, defaultVal int32) int32 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 32); err == nil {
			return int32(i)
		}
		slog.Warn("Invalid env var (expected int32)", "key", key, "value", v)
	}
	return defaultVal
}

// getFloat64Env парсит значение в float64.
func getFloat64Env(key string, defaultVal float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
		slog.Warn("Invalid env var (expected float64)", "key", key, "value", v)
	}
	return defaultVal
}

// parseBoolEnv парсит значение в bool.
func parseBoolEnv(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
		slog.Warn("Invalid env var (expected bool)", "key", key, "value", v)
	}
	return defaultVal
}

// parseDurationEnv парсит значение в time.Duration (например, "5s", "1m").
func parseDurationEnv(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		slog.Warn("Invalid env var (expected duration)", "key", key, "value", v)
	}
	return defaultVal
}
