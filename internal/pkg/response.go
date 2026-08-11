package pkg

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Message any    `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)

	if err := enc.Encode(v); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func ReturnError(w http.ResponseWriter, statusCode int, err error) {
	WriteJSON(w, statusCode, Response{Message: err.Error()})
}

func ReturnSuccess(w http.ResponseWriter, statusCode int, message string, data any) {
	WriteJSON(w, statusCode, Response{Message: message, Data: data})
}
