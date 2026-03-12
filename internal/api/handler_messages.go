package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/skrashevich/telegram-mock-ai/internal/bot"
	"github.com/skrashevich/telegram-mock-ai/internal/llm"
	"github.com/skrashevich/telegram-mock-ai/internal/models"
)

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	text := parseStringParam(r, "text")
	if text == "" {
		respondError(w, http.StatusBadRequest, "Bad Request: message text is empty")
		return
	}
	if len(text) > 4096 {
		respondError(w, http.StatusBadRequest, "Bad Request: message is too long")
		return
	}

	chat, exists := s.store.GetChat(chatID)
	if !exists {
		// Auto-create private chat for the bot
		chat = s.store.CreateChat(models.Chat{
			ID:   chatID,
			Type: "private",
		})
	}

	// Parse reply_markup if provided
	var replyMarkup *models.InlineKeyboard
	if rm := parseStringParam(r, "reply_markup"); rm != "" {
		var kb models.InlineKeyboard
		if err := json.Unmarshal([]byte(rm), &kb); err == nil {
			replyMarkup = &kb
		}
	}

	// Parse reply_to_message_id
	var replyTo *models.Message
	if replyToID, ok := parseIntParam(r, "reply_to_message_id"); ok && replyToID > 0 {
		replyTo, _ = s.store.GetMessage(chatID, int(replyToID))
	}

	msg := s.store.StoreMessage(models.Message{
		From:           &b.User,
		Chat:           *chat,
		Date:           time.Now().Unix(),
		Text:           text,
		ReplyToMessage: replyTo,
		ReplyMarkup:    replyMarkup,
	})

	respondOK(w, msg)

	// Async: generate LLM reply if enabled
	if s.cfg.LLM.Enabled && s.llm != nil {
		go s.generateReply(b, msg)
	}
}

func (s *Server) generateReply(b *bot.Bot, botMsg *models.Message) {
	chatID := botMsg.Chat.ID

	// Pick a non-bot user from the chat to be the "responder"
	nonBotMembers := s.store.GetNonBotChatMembers(chatID)
	if len(nonBotMembers) == 0 {
		slog.Debug("no non-bot members in chat, skipping reply", "chat_id", chatID)
		return
	}
	responder := nonBotMembers[rand.Intn(len(nonBotMembers))].User

	// Simulate typing delay
	delay := s.cfg.LLM.ResponseDelayMin +
		time.Duration(rand.Int63n(int64(s.cfg.LLM.ResponseDelayMax-s.cfg.LLM.ResponseDelayMin)))
	time.Sleep(delay)

	// Build chat history for context
	recentMsgs := s.store.GetChatMessages(chatID, 20)
	chatHistory := make([]llm.ChatMessage, 0, len(recentMsgs))
	for _, m := range recentMsgs {
		role := "user"
		if m.From != nil && m.From.IsBot {
			role = "assistant"
		}
		if m.Text != "" {
			chatHistory = append(chatHistory, llm.ChatMessage{
				Role:    role,
				Content: m.Text,
			})
		}
	}

	systemPrompt := s.cfg.LLM.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = defaultReplyPrompt(responder)
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.LLM.Timeout)
	defer cancel()

	replyText, err := s.llm.GenerateReply(ctx, systemPrompt, chatHistory)
	if err != nil {
		slog.Error("LLM reply generation failed", "err", err)
		return
	}

	if replyText == "" {
		return
	}

	// Truncate if too long
	if len(replyText) > 4096 {
		replyText = replyText[:4096]
	}

	replyMsg := s.store.StoreMessage(models.Message{
		From: &responder,
		Chat: botMsg.Chat,
		Date: time.Now().Unix(),
		Text: replyText,
	})

	update := models.Update{
		Message: replyMsg,
	}
	s.dispatcher.Dispatch(b.Token, update)
}

func (s *Server) handleEditMessageText(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
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

	text := parseStringParam(r, "text")
	if text == "" {
		respondError(w, http.StatusBadRequest, "Bad Request: message text is empty")
		return
	}

	msg, exists := s.store.GetMessage(chatID, int(messageID))
	if !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: message not found")
		return
	}

	// Check that the bot owns this message
	if msg.From == nil || msg.From.ID != b.User.ID {
		respondError(w, http.StatusBadRequest, "Bad Request: message can't be edited")
		return
	}

	// Parse reply_markup if provided
	var replyMarkup *models.InlineKeyboard
	if rm := parseStringParam(r, "reply_markup"); rm != "" {
		var kb models.InlineKeyboard
		if err := json.Unmarshal([]byte(rm), &kb); err == nil {
			replyMarkup = &kb
		}
	}

	updated, ok2 := s.store.UpdateMessageText(chatID, int(messageID), text, replyMarkup)
	if !ok2 {
		respondError(w, http.StatusBadRequest, "Bad Request: message not found")
		return
	}

	respondOK(w, updated)
}

func (s *Server) handleDeleteMessage(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
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

	if !s.store.DeleteMessage(chatID, int(messageID)) {
		respondError(w, http.StatusBadRequest, "Bad Request: message not found")
		return
	}

	respondBool(w, true)
}

func (s *Server) handleForwardMessage(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	fromChatID, ok := parseIntParam(r, "from_chat_id")
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: from_chat_id is required")
		return
	}

	messageID, ok := parseIntParam(r, "message_id")
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: message_id is required")
		return
	}

	original, exists := s.store.GetMessage(fromChatID, int(messageID))
	if !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: message not found")
		return
	}

	chat, exists := s.store.GetChat(chatID)
	if !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: chat not found")
		return
	}

	forwarded := s.store.StoreMessage(models.Message{
		From:        &b.User,
		Chat:        *chat,
		Date:        time.Now().Unix(),
		Text:        original.Text,
		ForwardFrom: original.From,
		ForwardDate: original.Date,
	})

	respondOK(w, forwarded)
}

func (s *Server) handleCopyMessage(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	fromChatID, ok := parseIntParam(r, "from_chat_id")
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: from_chat_id is required")
		return
	}

	messageID, ok := parseIntParam(r, "message_id")
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: message_id is required")
		return
	}

	original, exists := s.store.GetMessage(fromChatID, int(messageID))
	if !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: message not found")
		return
	}

	chat, exists := s.store.GetChat(chatID)
	if !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: chat not found")
		return
	}

	copied := s.store.StoreMessage(models.Message{
		From: &b.User,
		Chat: *chat,
		Date: time.Now().Unix(),
		Text: original.Text,
	})

	// copyMessage returns MessageId object
	respondOK(w, map[string]int{"message_id": copied.MessageID})
}

func defaultReplyPrompt(user models.User) string {
	name := user.FirstName
	if user.LastName != "" {
		name += " " + user.LastName
	}
	return `You are simulating a Telegram user named "` + name + `".
You are having a conversation with a bot. Respond naturally and realistically as this user would.
Keep your responses concise and conversational.
Output ONLY the message text, no JSON, no formatting markers.
Use the language that the bot writes in.`
}
