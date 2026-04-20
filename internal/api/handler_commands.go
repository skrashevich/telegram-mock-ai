package api

import (
	"encoding/json"
	"net/http"

	"github.com/skrashevich/telegram-mock-ai/internal/bot"
	"github.com/skrashevich/telegram-mock-ai/internal/models"
)

func (s *Server) handleSetMyCommands(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	raw := parseStringParam(r, "commands")
	if raw == "" {
		respondError(w, http.StatusBadRequest, "Bad Request: commands is required")
		return
	}

	var cmds []models.BotCommand
	if err := json.Unmarshal([]byte(raw), &cmds); err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request: invalid commands format")
		return
	}

	b.SetCommands(cmds)
	respondBool(w, true)
}

func (s *Server) handleGetMyCommands(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	respondOK(w, b.GetCommands())
}

func (s *Server) handleDeleteMyCommands(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	b.DeleteCommands()
	respondBool(w, true)
}
