package api

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/skrashevich/telegram-mock-ai/internal/bot"
	"github.com/skrashevich/telegram-mock-ai/internal/config"
	"github.com/skrashevich/telegram-mock-ai/internal/llm"
	"github.com/skrashevich/telegram-mock-ai/internal/state"
	"github.com/skrashevich/telegram-mock-ai/internal/updates"
)

// Server holds all dependencies for the Bot API HTTP server.
type Server struct {
	cfg        *config.Config
	store      *state.Store
	registry   *bot.Registry
	dispatcher *updates.Dispatcher
	llm        *llm.Client
	handlers   map[string]botHandler
}

// botHandler handles a specific Bot API method.
type botHandler func(w http.ResponseWriter, r *http.Request, b *bot.Bot)

// NewServer creates a new API server.
func NewServer(cfg *config.Config, store *state.Store, registry *bot.Registry, dispatcher *updates.Dispatcher, llmClient *llm.Client) *Server {
	s := &Server{
		cfg:        cfg,
		store:      store,
		registry:   registry,
		dispatcher: dispatcher,
		llm:        llmClient,
	}
	s.handlers = map[string]botHandler{
		// Info
		"getme":             s.handleGetMe,
		"getchat":           s.handleGetChat,
		"getchatmember":     s.handleGetChatMember,
		"getchatmembercount": s.handleGetChatMemberCount,
		"getchatmemberscount": s.handleGetChatMemberCount, // alias
		"getchatadministrators": s.handleGetChatAdministrators,
		// Updates
		"getupdates":     s.handleGetUpdates,
		"setwebhook":     s.handleSetWebhook,
		"deletewebhook":  s.handleDeleteWebhook,
		"getwebhookinfo": s.handleGetWebhookInfo,
		// Messages
		"sendmessage":      s.handleSendMessage,
		"editmessagetext":  s.handleEditMessageText,
		"deletemessage":    s.handleDeleteMessage,
		"forwardmessage":   s.handleForwardMessage,
		"copymessage":      s.handleCopyMessage,
		// Media (stubs)
		"sendphoto":     s.handleSendMedia,
		"senddocument":  s.handleSendMedia,
		"sendvideo":     s.handleSendMedia,
		"sendaudio":     s.handleSendMedia,
		"sendvoice":     s.handleSendMedia,
		"sendsticker":   s.handleSendMedia,
		"sendanimation": s.handleSendMedia,
		"sendlocation":  s.handleSendLocation,
		// Callbacks
		"answercallbackquery": s.handleAnswerCallbackQuery,
		// Chat management
		"banchatmember":      s.handleBanChatMember,
		"unbanchatmember":    s.handleUnbanChatMember,
		"restrictchatmember": s.handleRestrictChatMember,
		"promotechatmember":  s.handlePromoteChatMember,
		"leavechat":          s.handleLeaveChat,
		// Edit
		"editmessagereplymarkup": s.handleEditMessageReplyMarkup,
	}
	return s
}

// Handler returns the HTTP handler for the Bot API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/bot", func(w http.ResponseWriter, r *http.Request) {
		respondError(w, http.StatusNotFound, "Not Found: use /bot<token>/<method>")
	})
	// Catch-all pattern for /bot{token}/{method}
	mux.HandleFunc("/", s.handleBotRequest)
	return mux
}

func (s *Server) handleBotRequest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Parse /bot{token}/{method}
	if !strings.HasPrefix(path, "/bot") {
		respondError(w, http.StatusNotFound, "Not Found")
		return
	}

	rest := path[4:] // strip "/bot"
	slashIdx := strings.Index(rest, "/")
	if slashIdx < 0 {
		respondError(w, http.StatusNotFound, "Not Found: method required")
		return
	}

	token := rest[:slashIdx]
	method := rest[slashIdx+1:]

	if token == "" {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	b, ok := s.registry.Get(token)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Ensure the bot user exists in state
	if _, exists := s.store.GetUser(b.User.ID); !exists {
		s.store.CreateUser(b.User)
	}

	// On first connection, trigger welcome messages
	if s.registry.MarkConnected(token) {
		go s.sendWelcomeMessages(b)
	}

	methodLower := strings.ToLower(method)
	handler, ok := s.handlers[methodLower]
	if !ok {
		respondError(w, http.StatusBadRequest, "Bad Request: method not found")
		return
	}

	slog.Debug("API request", "method", method, "bot", b.User.Username)
	handler(w, r, b)
}
