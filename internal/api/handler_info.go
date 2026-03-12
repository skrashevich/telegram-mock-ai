package api

import (
	"net/http"

	"github.com/skrashevich/telegram-mock-ai/internal/bot"
)

func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	respondOK(w, b.User)
}

func (s *Server) handleGetChat(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	chat, exists := s.store.GetChat(chatID)
	if !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: chat not found")
		return
	}

	respondOK(w, chat)
}

func (s *Server) handleGetChatMember(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	userID, ok := parseIntParam(r, "user_id")
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: user_id is required")
		return
	}

	member, exists := s.store.GetChatMember(chatID, userID)
	if !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: user not found in chat")
		return
	}

	respondOK(w, member)
}

func (s *Server) handleGetChatMemberCount(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	if _, exists := s.store.GetChat(chatID); !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: chat not found")
		return
	}

	count := s.store.GetChatMemberCount(chatID)
	respondOK(w, count)
}

func (s *Server) handleGetChatAdministrators(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	if _, exists := s.store.GetChat(chatID); !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: chat not found")
		return
	}

	members := s.store.GetChatMembers(chatID)
	var admins []any
	for _, m := range members {
		if m.Status == "creator" || m.Status == "administrator" {
			admins = append(admins, m)
		}
	}
	if admins == nil {
		admins = []any{}
	}
	respondOK(w, admins)
}
