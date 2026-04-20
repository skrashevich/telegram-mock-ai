package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/skrashevich/telegram-mock-ai/internal/bot"
	"github.com/skrashevich/telegram-mock-ai/internal/models"
)

func (s *Server) handleBanChatMember(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
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

	member, exists := s.store.UpdateChatMemberStatus(chatID, userID, "kicked")
	if !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: user not found in chat")
		return
	}

	// Generate chat_member update
	chat, _ := s.store.GetChat(chatID)
	if chat != nil {
		update := models.Update{
			ChatMember: &models.ChatMemberUpdated{
				Chat: *chat,
				From: b.User,
				Date: time.Now().Unix(),
				OldChatMember: models.ChatMember{
					User:   member.User,
					Status: "member",
				},
				NewChatMember: *member,
			},
		}
		s.dispatcher.Dispatch(b.Token, update)
	}

	respondBool(w, true)
}

func (s *Server) handleUnbanChatMember(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
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

	_, exists := s.store.UpdateChatMemberStatus(chatID, userID, "left")
	if !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: user not found in chat")
		return
	}

	respondBool(w, true)
}

func (s *Server) handleRestrictChatMember(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
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

	_, exists := s.store.UpdateChatMemberStatus(chatID, userID, "restricted")
	if !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: user not found in chat")
		return
	}

	respondBool(w, true)
}

func (s *Server) handlePromoteChatMember(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
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

	_, exists := s.store.UpdateChatMemberStatus(chatID, userID, "administrator")
	if !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: user not found in chat")
		return
	}

	respondBool(w, true)
}

func (s *Server) handleLeaveChat(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	s.store.RemoveChatMember(chatID, b.User.ID)
	respondBool(w, true)
}

func (s *Server) handlePinChatMessage(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	messageID, ok := parseIntParam(r, "message_id")
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: message_id is required")
		return
	}

	if _, exists := s.store.GetMessage(chatID, int(messageID)); !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: message not found")
		return
	}

	respondBool(w, true)
}

func (s *Server) handleUnpinChatMessage(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	if _, exists := s.store.GetChat(chatID); !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: chat not found")
		return
	}

	respondBool(w, true)
}

func (s *Server) handleUnpinAllChatMessages(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	if _, exists := s.store.GetChat(chatID); !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: chat not found")
		return
	}

	respondBool(w, true)
}

func (s *Server) handleSetChatTitle(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	title := parseStringParam(r, "title")
	if title == "" {
		respondError(w, http.StatusBadRequest, "Bad Request: title is required")
		return
	}

	if !s.store.UpdateChatTitle(chatID, title) {
		respondError(w, http.StatusBadRequest, "Bad Request: chat not found")
		return
	}

	respondBool(w, true)
}

func (s *Server) handleSetChatDescription(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	if _, exists := s.store.GetChat(chatID); !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: chat not found")
		return
	}

	respondBool(w, true)
}

func (s *Server) handleSetChatPermissions(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	if _, exists := s.store.GetChat(chatID); !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: chat not found")
		return
	}

	respondBool(w, true)
}

func (s *Server) handleExportChatInviteLink(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	if _, exists := s.store.GetChat(chatID); !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: chat not found")
		return
	}

	link := fmt.Sprintf("https://t.me/+mock_%d", chatID)
	respondOK(w, link)
}

func (s *Server) handleSetMessageReaction(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	messageID, ok := parseIntParam(r, "message_id")
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: message_id is required")
		return
	}

	if _, exists := s.store.GetMessage(chatID, int(messageID)); !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: message not found")
		return
	}

	respondBool(w, true)
}

func (s *Server) handleGetChatMenuButton(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	respondOK(w, map[string]string{"type": "default"})
}

func (s *Server) handleSetChatMenuButton(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	respondBool(w, true)
}

func (s *Server) handleGetMyDefaultAdministratorRights(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	respondOK(w, map[string]any{})
}

func (s *Server) handleSetMyDefaultAdministratorRights(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	respondBool(w, true)
}

func (s *Server) handleEditMessageReplyMarkup(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	messageID, ok := parseIntParam(r, "message_id")
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: message_id is required")
		return
	}

	msg, exists := s.store.GetMessage(chatID, int(messageID))
	if !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: message not found")
		return
	}

	if msg.From == nil || msg.From.ID != b.User.ID {
		respondError(w, http.StatusBadRequest, "Bad Request: message can't be edited")
		return
	}

	// Clear or update reply markup
	var replyMarkup *models.InlineKeyboard
	rm := parseStringParam(r, "reply_markup")
	if rm != "" {
		var kb models.InlineKeyboard
		if err := parseJSON(rm, &kb); err == nil {
			replyMarkup = &kb
		}
	}

	updated, ok2 := s.store.UpdateMessageReplyMarkup(chatID, int(messageID), replyMarkup)
	if !ok2 {
		respondError(w, http.StatusBadRequest, "Bad Request: message not found")
		return
	}

	respondOK(w, updated)
}
