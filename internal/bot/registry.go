package bot

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/skrashevich/telegram-mock-ai/internal/models"
)

// Bot represents a registered bot with its configuration.
type Bot struct {
	Token          string
	User           models.User
	WebhookURL     string
	SecretToken    string
	AllowedUpdates []string
}

// Registry manages registered bots by their token.
type Registry struct {
	mu           sync.RWMutex
	bots         map[string]*Bot
	autoRegister bool
	botIDGen     *models.IDGenerator
}

// NewRegistry creates a new bot registry.
func NewRegistry(autoRegister bool) *Registry {
	return &Registry{
		bots:         make(map[string]*Bot),
		autoRegister: autoRegister,
		botIDGen:     models.NewIDGenerator(100000),
	}
}

// Register adds a bot explicitly.
func (r *Registry) Register(token, username, firstName string) *Bot {
	r.mu.Lock()
	defer r.mu.Unlock()

	if b, ok := r.bots[token]; ok {
		return b
	}

	id := r.botIDGen.Next()
	if username == "" {
		username = fmt.Sprintf("bot_%d", id)
	}
	if firstName == "" {
		firstName = fmt.Sprintf("Bot %d", id)
	}

	b := &Bot{
		Token: token,
		User: models.User{
			ID:        id,
			IsBot:     true,
			FirstName: firstName,
			Username:  username,
		},
	}
	r.bots[token] = b
	return b
}

// Get returns a bot by token. If autoRegister is enabled, creates one on first access.
func (r *Registry) Get(token string) (*Bot, bool) {
	r.mu.RLock()
	b, ok := r.bots[token]
	r.mu.RUnlock()

	if ok {
		return b, true
	}

	if !r.autoRegister {
		return nil, false
	}

	// Auto-register with deterministic ID from token hash
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if b, ok := r.bots[token]; ok {
		return b, true
	}

	id := tokenToID(token)
	b = &Bot{
		Token: token,
		User: models.User{
			ID:        id,
			IsBot:     true,
			FirstName: fmt.Sprintf("Bot %d", id),
			Username:  fmt.Sprintf("bot_%d", id),
		},
	}
	r.bots[token] = b
	return b, true
}

// List returns all registered bots.
func (r *Registry) List() []*Bot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Bot, 0, len(r.bots))
	for _, b := range r.bots {
		result = append(result, b)
	}
	return result
}

// SetWebhook sets the webhook URL for a bot.
func (r *Registry) SetWebhook(token, url, secretToken string, allowedUpdates []string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.bots[token]
	if !ok {
		return false
	}
	b.WebhookURL = url
	b.SecretToken = secretToken
	b.AllowedUpdates = allowedUpdates
	return true
}

// DeleteWebhook removes the webhook for a bot.
func (r *Registry) DeleteWebhook(token string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.bots[token]
	if !ok {
		return false
	}
	b.WebhookURL = ""
	b.SecretToken = ""
	b.AllowedUpdates = nil
	return true
}

// tokenToID generates a deterministic user ID from a bot token.
func tokenToID(token string) int64 {
	h := sha256.Sum256([]byte(token))
	return int64(binary.BigEndian.Uint32(h[:4]))%900000 + 100000
}
