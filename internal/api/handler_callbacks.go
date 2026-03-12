package api

import (
	"net/http"

	"github.com/skrashevich/telegram-mock-ai/internal/bot"
)

func (s *Server) handleAnswerCallbackQuery(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	callbackQueryID := parseStringParam(r, "callback_query_id")
	if callbackQueryID == "" {
		respondError(w, http.StatusBadRequest, "Bad Request: callback_query_id is required")
		return
	}

	// In a mock, we just acknowledge the callback query
	respondBool(w, true)
}
