package main

import (
	"log/slog"
	"os"
	"strconv"
)

type DBConfig struct {
	URL      string
	MaxConns int32
	MinConns int32
}

type SrvConfig struct {
	addr string
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

func LoadDBConfig() *DBConfig {

	return &DBConfig{
		URL:      getStrEnv("DB_URL", "postgres://admin:secret@localhost:5432/payship_core?sslmode=disable"),
		MaxConns: parseIntEnv("MAX_CONNS", 10),
		MinConns: parseIntEnv("MIN_CONNS", 2)}
}

func LoadSrvConfig() *SrvConfig {
	return &SrvConfig{addr: getStrEnv("SERVER_ADDR", "0.0.0.0:8080")}
}
