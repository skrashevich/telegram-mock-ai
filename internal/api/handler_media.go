package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/skrashevich/telegram-mock-ai/internal/bot"
	"github.com/skrashevich/telegram-mock-ai/internal/models"
)

// handleSendMedia handles sendPhoto, sendDocument, sendVideo, sendAudio, sendVoice, sendSticker, sendAnimation.
// Returns stub file_id values without storing actual files.
func (s *Server) handleSendMedia(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	chat, exists := s.store.GetChat(chatID)
	if !exists {
		chat = s.store.CreateChat(models.Chat{
			ID:   chatID,
			Type: "private",
		})
	}

	caption := parseStringParam(r, "caption")
	fileID := "AgACAgIAAxkBAAI" + models.RandomHex(16)
	fileUniqueID := "AQADAgAT" + models.RandomHex(8)

	msg := models.Message{
		From: &b.User,
		Chat: *chat,
		Date: time.Now().Unix(),
		Text: caption,
	}

	// Determine media type from the URL path (case-insensitive)
	method := strings.ToLower(r.URL.Path)
	switch {
	case strings.Contains(method, "photo"):
		msg.Photo = []models.PhotoSize{
			{FileID: fileID, FileUniqueID: fileUniqueID, Width: 800, Height: 600},
			{FileID: fileID + "_thumb", FileUniqueID: fileUniqueID + "_t", Width: 320, Height: 240},
		}
	case strings.Contains(method,"document"):
		msg.Document = &models.Document{
			FileID:       fileID,
			FileUniqueID: fileUniqueID,
			FileName:     "document.pdf",
			MimeType:     "application/pdf",
		}
	case strings.Contains(method,"video"):
		msg.Video = &models.Video{
			FileID:       fileID,
			FileUniqueID: fileUniqueID,
			Width:        1920,
			Height:       1080,
			Duration:     30,
		}
	case strings.Contains(method,"audio"):
		msg.Audio = &models.Audio{
			FileID:       fileID,
			FileUniqueID: fileUniqueID,
			Duration:     180,
			Title:        "audio",
		}
	case strings.Contains(method,"voice"):
		msg.Voice = &models.Voice{
			FileID:       fileID,
			FileUniqueID: fileUniqueID,
			Duration:     5,
		}
	case strings.Contains(method,"sticker"):
		msg.Sticker = &models.Sticker{
			FileID:       fileID,
			FileUniqueID: fileUniqueID,
			Type:         "regular",
			Width:        512,
			Height:       512,
			Emoji:        "😀",
		}
	default:
		// Animation or other - treat as document
		msg.Document = &models.Document{
			FileID:       fileID,
			FileUniqueID: fileUniqueID,
			MimeType:     "video/mp4",
		}
	}

	stored := s.store.StoreMessage(msg)
	respondOK(w, stored)
}

func (s *Server) handleSendLocation(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	chat, exists := s.store.GetChat(chatID)
	if !exists {
		chat = s.store.CreateChat(models.Chat{
			ID:   chatID,
			Type: "private",
		})
	}

	msg := s.store.StoreMessage(models.Message{
		From: &b.User,
		Chat: *chat,
		Date: time.Now().Unix(),
	})

	respondOK(w, msg)
}

