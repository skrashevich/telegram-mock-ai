package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// APIResponse is the standard Telegram Bot API response envelope.
type APIResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result,omitempty"`
	ErrorCode   int             `json:"error_code,omitempty"`
	Description string          `json:"description,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode response", "err", err)
	}
}

func respondOK(w http.ResponseWriter, result any) {
	raw, err := json.Marshal(result)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Internal error")
		return
	}
	respondJSON(w, http.StatusOK, APIResponse{
		OK:     true,
		Result: raw,
	})
}

func respondBool(w http.ResponseWriter, val bool) {
	raw, _ := json.Marshal(val)
	respondJSON(w, http.StatusOK, APIResponse{
		OK:     true,
		Result: raw,
	})
}

func respondError(w http.ResponseWriter, status int, description string) {
	respondJSON(w, status, APIResponse{
		OK:          false,
		ErrorCode:   status,
		Description: description,
	})
}
