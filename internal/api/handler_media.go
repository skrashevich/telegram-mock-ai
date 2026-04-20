package api

import (
	"math/rand"
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
		From:    &b.User,
		Chat:    *chat,
		Date:    time.Now().Unix(),
		Caption: caption,
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
	case strings.Contains(method, "animation"):
		msg.Animation = &models.Animation{
			FileID:       fileID,
			FileUniqueID: fileUniqueID,
			Width:        320,
			Height:       240,
			Duration:     5,
			MimeType:     "video/mp4",
		}
		// Telegram also sets document for animations
		msg.Document = &models.Document{
			FileID:       fileID,
			FileUniqueID: fileUniqueID,
			MimeType:     "video/mp4",
		}
	default:
		msg.Document = &models.Document{
			FileID:       fileID,
			FileUniqueID: fileUniqueID,
			MimeType:     "application/octet-stream",
		}
	}

	stored := s.store.StoreMessage(msg)
	respondOK(w, stored)
}

func (s *Server) handleSendPoll(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	question := parseStringParam(r, "question")
	if question == "" {
		respondError(w, http.StatusBadRequest, "Bad Request: question is required")
		return
	}

	optionsRaw := parseStringParam(r, "options")
	if optionsRaw == "" {
		respondError(w, http.StatusBadRequest, "Bad Request: options is required")
		return
	}

	var optionTexts []string
	if err := parseJSON(optionsRaw, &optionTexts); err != nil {
		// Try parsing as InputPollOption objects
		var inputOptions []struct {
			Text string `json:"text"`
		}
		if err2 := parseJSON(optionsRaw, &inputOptions); err2 != nil {
			respondError(w, http.StatusBadRequest, "Bad Request: invalid options format")
			return
		}
		for _, o := range inputOptions {
			optionTexts = append(optionTexts, o.Text)
		}
	}

	if len(optionTexts) < 2 {
		respondError(w, http.StatusBadRequest, "Bad Request: poll must have at least 2 options")
		return
	}

	chat, exists := s.store.GetChat(chatID)
	if !exists {
		chat = s.store.CreateChat(models.Chat{ID: chatID, Type: "private"})
	}

	pollOptions := make([]models.PollOption, len(optionTexts))
	for i, t := range optionTexts {
		pollOptions[i] = models.PollOption{Text: t}
	}

	pollType := parseStringParam(r, "type")
	if pollType == "" {
		pollType = "regular"
	}

	poll := models.Poll{
		ID:          models.RandomHex(8),
		Question:    question,
		Options:     pollOptions,
		IsAnonymous: true,
		Type:        pollType,
	}

	msg := s.store.StoreMessage(models.Message{
		From: &b.User,
		Chat: *chat,
		Date: time.Now().Unix(),
		Poll: &poll,
	})

	respondOK(w, msg)
}

func (s *Server) handleSendDice(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	chat, exists := s.store.GetChat(chatID)
	if !exists {
		chat = s.store.CreateChat(models.Chat{ID: chatID, Type: "private"})
	}

	emoji := parseStringParam(r, "emoji")
	if emoji == "" {
		emoji = "🎲"
	}

	maxVal := 6
	switch emoji {
	case "🎯":
		maxVal = 6
	case "🏀":
		maxVal = 5
	case "⚽":
		maxVal = 5
	case "🎳":
		maxVal = 6
	case "🎰":
		maxVal = 64
	}

	value := rand.Intn(maxVal) + 1

	msg := s.store.StoreMessage(models.Message{
		From: &b.User,
		Chat: *chat,
		Date: time.Now().Unix(),
		Dice: &models.Dice{Emoji: emoji, Value: value},
	})

	respondOK(w, msg)
}

func (s *Server) handleSendContact(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	phoneNumber := parseStringParam(r, "phone_number")
	if phoneNumber == "" {
		respondError(w, http.StatusBadRequest, "Bad Request: phone_number is required")
		return
	}

	firstName := parseStringParam(r, "first_name")
	if firstName == "" {
		respondError(w, http.StatusBadRequest, "Bad Request: first_name is required")
		return
	}

	chat, exists := s.store.GetChat(chatID)
	if !exists {
		chat = s.store.CreateChat(models.Chat{ID: chatID, Type: "private"})
	}

	contact := models.Contact{
		PhoneNumber: phoneNumber,
		FirstName:   firstName,
		LastName:    parseStringParam(r, "last_name"),
		VCard:       parseStringParam(r, "vcard"),
	}

	msg := s.store.StoreMessage(models.Message{
		From:    &b.User,
		Chat:    *chat,
		Date:    time.Now().Unix(),
		Contact: &contact,
	})

	respondOK(w, msg)
}

func (s *Server) handleSendVenue(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	title := parseStringParam(r, "title")
	address := parseStringParam(r, "address")
	if title == "" || address == "" {
		respondError(w, http.StatusBadRequest, "Bad Request: title and address are required")
		return
	}

	lat, latOK := parseFloatParam(r, "latitude")
	lon, lonOK := parseFloatParam(r, "longitude")
	if !latOK || !lonOK {
		respondError(w, http.StatusBadRequest, "Bad Request: latitude and longitude are required")
		return
	}

	chat, exists := s.store.GetChat(chatID)
	if !exists {
		chat = s.store.CreateChat(models.Chat{ID: chatID, Type: "private"})
	}

	venue := models.Venue{
		Location: models.Location{Latitude: lat, Longitude: lon},
		Title:    title,
		Address:  address,
	}

	msg := s.store.StoreMessage(models.Message{
		From:    &b.User,
		Chat:    *chat,
		Date:    time.Now().Unix(),
		Venue:   &venue,
		Location: &venue.Location,
	})

	respondOK(w, msg)
}

func (s *Server) handleSendLocation(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	chatID, ok := parseChatID(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: chat_id is required")
		return
	}

	lat, latOK := parseFloatParam(r, "latitude")
	lon, lonOK := parseFloatParam(r, "longitude")
	if !latOK || !lonOK {
		respondError(w, http.StatusBadRequest, "Bad Request: latitude and longitude are required")
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
		Location: &models.Location{
			Latitude:  lat,
			Longitude: lon,
		},
	})

	respondOK(w, msg)
}

