package dtos

import (
	"encoding/json"
	"net/http"
)

// ApiResponse API 표준 응답 구조체
type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ResponseData structure
type ResponseData struct {
	Data interface{} `json:"data"`
	Meta interface{} `json:"meta"`
}

// Message returns map data
func Message(status int, message string) map[string]interface{} {
	return map[string]interface{}{"status": status, "message": message}
}

// Respond returns basic response structure
func Respond(w http.ResponseWriter, data map[string]interface{}) {
	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// JSON 인코딩 실패 시 HTTP 500 응답
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}