package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/skrashevich/telegram-mock-ai/internal/bot"
	"github.com/skrashevich/telegram-mock-ai/internal/config"
	"github.com/skrashevich/telegram-mock-ai/internal/llm"
	"github.com/skrashevich/telegram-mock-ai/internal/models"
	"github.com/skrashevich/telegram-mock-ai/internal/seed"
	"github.com/skrashevich/telegram-mock-ai/internal/state"
	"github.com/skrashevich/telegram-mock-ai/internal/updates"
)

// AdminServer provides management endpoints for state inspection and manipulation.
type AdminServer struct {
	store      *state.Store
	registry   *bot.Registry
	dispatcher *updates.Dispatcher
	llmClient  *llm.Client
	seedCfg    *config.SeedGenerate
}

// NewAdminServer creates a new admin API server.
func NewAdminServer(store *state.Store, registry *bot.Registry, dispatcher *updates.Dispatcher, llmClient *llm.Client, seedCfg *config.SeedGenerate) *AdminServer {
	return &AdminServer{
		store:      store,
		registry:   registry,
		dispatcher: dispatcher,
		llmClient:  llmClient,
		seedCfg:    seedCfg,
	}
}

// Handler returns the HTTP handler for the admin API.
func (a *AdminServer) Handler() http.Handler {
	mux := http.NewServeMux()

	// Bot management
	mux.HandleFunc("GET /api/bots", a.handleListBots)

	// User management
	mux.HandleFunc("GET /api/users", a.handleListUsers)
	mux.HandleFunc("POST /api/users", a.handleCreateUser)

	// Chat management
	mux.HandleFunc("GET /api/chats", a.handleListChats)
	mux.HandleFunc("POST /api/chats", a.handleCreateChat)
	mux.HandleFunc("POST /api/chats/{chat_id}/members", a.handleAddChatMember)
	mux.HandleFunc("GET /api/chats/{chat_id}/members", a.handleListChatMembers)
	mux.HandleFunc("GET /api/chats/{chat_id}/messages", a.handleListChatMessages)

	// Update injection
	mux.HandleFunc("POST /api/bots/{token}/updates", a.handleInjectUpdate)

	// Message injection (simulates a user sending a message)
	mux.HandleFunc("POST /api/chats/{chat_id}/messages", a.handleInjectMessage)

	// Seed generation
	mux.HandleFunc("POST /api/seed/generate", a.handleSeedGenerate)

	// Health
	mux.HandleFunc("GET /api/health", a.handleHealth)

	// State dump
	mux.HandleFunc("GET /api/state", a.handleDumpState)

	return mux
}

func (a *AdminServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *AdminServer) handleListBots(w http.ResponseWriter, r *http.Request) {
	bots := a.registry.List()
	result := make([]map[string]any, 0, len(bots))
	for _, b := range bots {
		result = append(result, map[string]any{
			"token":       b.Token,
			"user":        b.User,
			"webhook_url": b.WebhookURL,
		})
	}
	respondJSON(w, http.StatusOK, result)
}

func (a *AdminServer) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users := a.store.ListUsers()
	respondJSON(w, http.StatusOK, users)
}

func (a *AdminServer) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID        int64  `json:"id,omitempty"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name,omitempty"`
		Username  string `json:"username,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request: invalid JSON")
		return
	}
	if req.FirstName == "" {
		respondError(w, http.StatusBadRequest, "Bad Request: first_name is required")
		return
	}
	user := a.store.CreateUser(models.User{
		ID:        req.ID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  req.Username,
	})
	respondJSON(w, http.StatusCreated, user)
}

func (a *AdminServer) handleListChats(w http.ResponseWriter, r *http.Request) {
	chats := a.store.ListChats()
	respondJSON(w, http.StatusOK, chats)
}

func (a *AdminServer) handleCreateChat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID      int64   `json:"id,omitempty"`
		Type    string  `json:"type"`
		Title   string  `json:"title,omitempty"`
		Members []int64 `json:"members,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request: invalid JSON")
		return
	}
	if req.Type == "" {
		req.Type = "group"
	}

	chat := a.store.CreateChat(models.Chat{
		ID:    req.ID,
		Type:  req.Type,
		Title: req.Title,
	})

	// Add members
	for _, uid := range req.Members {
		a.store.AddChatMember(chat.ID, uid, "member")
	}

	respondJSON(w, http.StatusCreated, chat)
}

func (a *AdminServer) handleAddChatMember(w http.ResponseWriter, r *http.Request) {
	chatID, err := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request: invalid chat_id")
		return
	}

	var req struct {
		UserID int64  `json:"user_id"`
		Status string `json:"status,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request: invalid JSON")
		return
	}
	if req.Status == "" {
		req.Status = "member"
	}

	member := a.store.AddChatMember(chatID, req.UserID, req.Status)
	if member == nil {
		respondError(w, http.StatusBadRequest, "Bad Request: user not found")
		return
	}

	respondJSON(w, http.StatusCreated, member)
}

func (a *AdminServer) handleListChatMembers(w http.ResponseWriter, r *http.Request) {
	chatID, err := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request: invalid chat_id")
		return
	}
	members := a.store.GetChatMembers(chatID)
	respondJSON(w, http.StatusOK, members)
}

func (a *AdminServer) handleListChatMessages(w http.ResponseWriter, r *http.Request) {
	chatID, err := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request: invalid chat_id")
		return
	}
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	messages := a.store.GetChatMessages(chatID, limit)
	respondJSON(w, http.StatusOK, messages)
}

func (a *AdminServer) handleInjectUpdate(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "Bad Request: token is required")
		return
	}

	_, ok := a.registry.Get(token)
	if !ok {
		respondError(w, http.StatusNotFound, "Not Found: bot not registered")
		return
	}

	var update models.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request: invalid JSON")
		return
	}

	a.dispatcher.Dispatch(token, update)
	respondJSON(w, http.StatusOK, map[string]string{"status": "dispatched"})
}

func (a *AdminServer) handleInjectMessage(w http.ResponseWriter, r *http.Request) {
	chatID, err := strconv.ParseInt(r.PathValue("chat_id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request: invalid chat_id")
		return
	}

	var req struct {
		UserID int64  `json:"user_id"`
		Text   string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request: invalid JSON")
		return
	}

	chat, exists := a.store.GetChat(chatID)
	if !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: chat not found")
		return
	}

	user, exists := a.store.GetUser(req.UserID)
	if !exists {
		respondError(w, http.StatusBadRequest, "Bad Request: user not found")
		return
	}

	msg := a.store.StoreMessage(models.Message{
		From: user,
		Chat: *chat,
		Date: time.Now().Unix(),
		Text: req.Text,
	})

	// Dispatch to all bots in this chat
	update := models.Update{Message: msg}
	bots := a.registry.List()
	for _, b := range bots {
		a.dispatcher.Dispatch(b.Token, update)
	}

	slog.Info("admin: injected message", "chat_id", chatID, "from", user.FirstName)
	respondJSON(w, http.StatusCreated, msg)
}

func (a *AdminServer) handleSeedGenerate(w http.ResponseWriter, r *http.Request) {
	if a.llmClient == nil {
		respondError(w, http.StatusServiceUnavailable, "LLM is not enabled; cannot generate seed data")
		return
	}

	// Parse optional request body
	var req seed.GenerateRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Bad Request: invalid JSON")
			return
		}
	}

	// Apply defaults from config if fields are zero
	if a.seedCfg != nil {
		if req.UsersCount == 0 {
			req.UsersCount = a.seedCfg.UsersCount
		}
		if req.GroupsCount == 0 {
			req.GroupsCount = a.seedCfg.GroupsCount
		}
		if req.ChannelsCount == 0 {
			req.ChannelsCount = a.seedCfg.ChannelsCount
		}
		if req.Locale == "" {
			req.Locale = a.seedCfg.Locale
		}
	}

	maxRetries := 2
	if a.seedCfg != nil && a.seedCfg.MaxRetries > 0 {
		maxRetries = a.seedCfg.MaxRetries
	}

	gen := seed.NewGenerator(a.llmClient, a.store, a.registry, maxRetries)
	result, err := gen.Generate(r.Context(), req)
	if err != nil {
		slog.Error("admin: seed generation failed", "err", err)
		respondError(w, http.StatusInternalServerError, "Seed generation failed: "+err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, result)
}

func (a *AdminServer) handleDumpState(w http.ResponseWriter, r *http.Request) {
	state := map[string]any{
		"bots":  a.registry.List(),
		"users": a.store.ListUsers(),
		"chats": a.store.ListChats(),
	}
	respondJSON(w, http.StatusOK, state)
}
