package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/skrashevich/telegram-mock-ai/internal/bot"
	"github.com/skrashevich/telegram-mock-ai/internal/models"
)

func (s *Server) handleGetUpdates(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	// Reject if webhook is set
	if b.WebhookURL != "" {
		respondError(w, http.StatusConflict, "Conflict: can't use getUpdates method while webhook is active")
		return
	}

	offset, _ := parseIntParam(r, "offset")
	limit, hasLimit := parseIntParam(r, "limit")
	if !hasLimit || limit <= 0 {
		limit = 100
	}
	timeoutSec, _ := parseIntParam(r, "timeout")
	if timeoutSec > 50 {
		timeoutSec = 50
	}
	timeout := time.Duration(timeoutSec) * time.Second

	q := s.dispatcher.GetOrCreateQueue(b.Token)

	// Confirm previously received updates
	if offset > 0 {
		q.Confirm(offset)
	}

	updates := q.Dequeue(r.Context(), offset, int(limit), timeout)
	if updates == nil {
		updates = []models.Update{}
	}

	respondOK(w, updates)
}

func (s *Server) handleSetWebhook(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	rawURL := parseStringParam(r, "url")
	if rawURL == "" {
		respondError(w, http.StatusBadRequest, "Bad Request: url is required")
		return
	}

	// Validate webhook URL scheme
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		respondError(w, http.StatusBadRequest, "Bad Request: invalid webhook URL scheme")
		return
	}

	secretToken := parseStringParam(r, "secret_token")
	var allowedUpdates []string
	if au := parseStringParam(r, "allowed_updates"); au != "" {
		if err := json.Unmarshal([]byte(au), &allowedUpdates); err != nil {
			allowedUpdates = []string{au}
		}
	}

	s.registry.SetWebhook(b.Token, rawURL, secretToken, allowedUpdates)
	respondBool(w, true)
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	s.registry.DeleteWebhook(b.Token)
	respondBool(w, true)
}

func (s *Server) handleGetWebhookInfo(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	q := s.dispatcher.GetOrCreateQueue(b.Token)
	info := models.WebhookInfo{
		URL:                b.WebhookURL,
		PendingUpdateCount: q.PendingCount(),
		AllowedUpdates:     b.AllowedUpdates,
	}
	respondOK(w, info)
}
