package main

import (
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ThisIsTheOldGuard/payship-core/internal/api"
	"github.com/jackc/pgx/v5/pgxpool"
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
//   - addr: адрес и порт для прослушивания (например, "0.0.0.0:8080").
type SrvConfig struct {
	addr string
}

// TestConfig содержит параметры запуска HTTP-сервера.
//
// Поля:
//   - IsTest: определяет вариант работы приложения (например, "true").
type TestConfig struct {
	IsTest bool
}

// LoadDBConfig загружает конфигурацию БД из переменных окружения.
//
// Функция читает DB_URL, MAX_CONNS, MIN_CONNS через os.Getenv.
// При отсутствии или ошибке парсинга используются значения по умолчанию:
//   - DB_URL: "postgres://admin:secret@localhost:5432/payship_core?sslmode=disable"
//   - MAX_CONNS: 10
//   - MIN_CONNS: 2
//
// Возвращает заполненную *DBConfig для передачи в NewDBPool.
func LoadDBConfig() *DBConfig {

	return &DBConfig{
		URL:             getStrEnv("DB_URL", "postgres://admin:secret@localhost:5432/payship_core?sslmode=disable"),
		MaxConns:        parseIntEnv("MAX_CONNS", 20),
		MinConns:        parseIntEnv("MIN_CONNS", 5),
		MaxConnIdleTime: parseTimeEnv("DB_MAX_CONN_IDLE_MINUTES", 5*time.Minute),
		MaxConnLifetime: parseTimeEnv("DB_MAX_CONN_LIFE_HOURS", 1*time.Hour),
	}
}

// LoadSrvConfig загружает конфигурацию HTTP-сервера из переменных окружения.
//
// Функция читает SERVER_ADDR через os.Getenv.
// При отсутствии используется значение по умолчанию: "0.0.0.0:8080".
//
// Возвращает заполненную *SrvConfig для настройки http.Server.
func LoadSrvConfig() *SrvConfig {
	return &SrvConfig{addr: getStrEnv("SERVER_ADDR", "0.0.0.0:8080")}
}

// LoadTestConfig загружает настройки для определения формата работы приложения
//
// Возвращает *TestConfig с параметрами тестовых данных
func LoadTestConfig() *TestConfig {
	return &TestConfig{parseBoolEnv("IsTest", true)}
}

func InitTestHandlers(mux *http.ServeMux, repo *pgxpool.Pool) {
	TestCfg := LoadTestConfig()
	if TestCfg.IsTest {
		mux.HandleFunc("GET /debug/db/slow", api.Debug_DB_Slow(repo))
	}
}

// parseIntEnv возвращает параметр в числовом представлении.
//
// Функция парсит текстовое значение из env файла и возвращает int значение,
// если значение не заполнено возвращает стандартное значение.
//
// Параметры:
// - key: ключ параметра.
// - defaultVal: стандартное значение, которое возвращается, если параметр не задан.
func parseIntEnv(key string, defaultVal int) int32 {
	v := os.Getenv(key)
	if v == "" {
		return int32(defaultVal)
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		slog.Warn("Invalid env var", "key", key, "error", err)
		return int32(defaultVal)
	}
	return int32(i)
}

// parseBoolEnv возвращает параметр в булевом представлении.
//
// Функция парсит текстовое значение из env файла и возвращает булево значение,
// если значение не заполнено возвращает стандартное значение.
//
// Параметры:
// - key: ключ параметра.
// - defaultVal: стандартное значение, которое возвращается, если параметр не задан.
func parseBoolEnv(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		slog.Warn("Invalid env var", "key", key, "error", err)
		return defaultVal
	}

	return b
}

// parseBoolEnv получает параметр в текстовом представлении.
//
// Функция парсит текстовое значение из env файла и возвращает значение,
// если значение не заполнено возвращает стандартное значение.
//
// Параметры:
// - key: ключ параметра.
// - defaultVal: стандартное значение, которое возвращается, если параметр не задан.
func getStrEnv(key string, defaultVal string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	return v
}

// parseTimeEnv возвращает параметр в временном представлении.
//
// Функция парсит текстовое значение из env файла и возвращает *time.Duration значение,
// если значение не заполнено возвращает стандартное значение.
//
// Параметры:
// - key: ключ параметра.
// - defaultVal: стандартное значение, которое возвращается, если параметр не задан.
func parseTimeEnv(key string, defaultVal time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	i, err := time.ParseDuration(v)
	if err != nil {
		slog.Warn("Invalid env var", "key", key, "error", err)
		return defaultVal
	}
	return i
}
