package main

import (
	"log/slog"
	"net/http"

	"github.com/ThisIsTheOldGuard/payship-core/internal/api"
)

func main() {

	http.HandleFunc("/", api.HomeHandler)
	slog.Info("Starting server", "address", "http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		slog.Error("Server failed", "error", err)
	}
}
