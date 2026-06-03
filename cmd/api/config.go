package main

import (
	"log/slog"
	"os"
	"strconv"
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
	URL      string
	MaxConns int32
	MinConns int32
}

// SrvConfig содержит параметры запуска HTTP-сервера.
//
// Поля:
//   - addr: адрес и порт для прослушивания (например, "0.0.0.0:8080").
type SrvConfig struct {
	addr string
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
		URL:      getStrEnv("DB_URL", "postgres://admin:secret@localhost:5432/payship_core?sslmode=disable"),
		MaxConns: parseIntEnv("MAX_CONNS", 10),
		MinConns: parseIntEnv("MIN_CONNS", 2)}
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

func getStrEnv(key string, defaultVal string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	return v
}
