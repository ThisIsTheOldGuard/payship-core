package api

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse - структура представления ошибки.
type ErrorResponse struct {
	Err string `json:"error"`
}

// Вовзможно в будущем надо сделать ответ в виде перечисления одного из внутренних статусов
type HealthResponse struct {
	Status string `json:"status"`
}

// sendJSONError отправляет стандартизированный JSON-ответ об ошибке.
//
// Вспомогательная функция устанавливает:
//   - Content-Type: application/json
//   - Указанный HTTP-статус код
//   - Тело {"error": "<сообщение>"}
//
// Параметры:
//   - w: http.ResponseWriter для отправки ответа.
//   - status: HTTP-статус код (например, 400, 404, 500).
//   - message: человекочитаемое сообщение об ошибке.
//
// Пример:
//
//	sendJSONError(w, http.StatusBadRequest, "invalid amount")
//	// Ответ: 400 {"error":"invalid amount"}
func SendJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Err: message})
}

// sendJSONError отправляет стандартизированный JSON-ответ о статусе сервиса.
//
// Вспомогательная функция устанавливает:
//   - Content-Type: application/json
//   - Указанный HTTP-статус код
//   - Тело {"status": "<сообщение>"}
//
// Параметры:
//   - w: http.ResponseWriter для отправки ответа.
//   - status: HTTP-статус код (например, 200).
//   - message: человекочитаемое сообщение о статусе.
//
// Пример:
//
//	sendJSONHealth(w, http.StatusOK, "Ready")
//	// Ответ: {"status":"Ready"}
func sendJSONHealth(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(HealthResponse{Status: message})
}
