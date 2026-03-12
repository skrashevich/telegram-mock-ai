package updates

import (
	"log/slog"
	"sync"

	"github.com/skrashevich/telegram-mock-ai/internal/bot"
	"github.com/skrashevich/telegram-mock-ai/internal/models"
	"github.com/skrashevich/telegram-mock-ai/internal/webhook"
)

// Dispatcher routes updates to the correct delivery channel (queue or webhook).
type Dispatcher struct {
	mu       sync.RWMutex
	queues   map[string]*Queue // token -> queue
	registry *bot.Registry
	webhook  *webhook.Client
}

// NewDispatcher creates a new update dispatcher.
func NewDispatcher(registry *bot.Registry, webhookClient *webhook.Client) *Dispatcher {
	return &Dispatcher{
		queues:   make(map[string]*Queue),
		registry: registry,
		webhook:  webhookClient,
	}
}

// Dispatch sends an update to the appropriate bot.
func (d *Dispatcher) Dispatch(token string, update models.Update) {
	b, ok := d.registry.Get(token)
	if !ok {
		slog.Warn("dispatch to unknown bot", "token", token[:min(10, len(token))]+"...")
		return
	}

	if b.WebhookURL != "" {
		if err := d.webhook.Send(b.WebhookURL, b.SecretToken, update); err != nil {
			slog.Error("webhook delivery failed", "url", b.WebhookURL, "err", err)
		}
		return
	}

	q := d.GetOrCreateQueue(token)
	q.Enqueue(update)
}

// GetOrCreateQueue returns the queue for a bot token, creating one if needed.
func (d *Dispatcher) GetOrCreateQueue(token string) *Queue {
	d.mu.RLock()
	q, ok := d.queues[token]
	d.mu.RUnlock()
	if ok {
		return q
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if q, ok := d.queues[token]; ok {
		return q
	}
	q = NewQueue()
	d.queues[token] = q
	return q
}

// DispatchToAll sends an update to all bots that are members of a specific chat.
// Excludes the sender bot (identified by excludeToken).
func (d *Dispatcher) DispatchToAll(excludeToken string, update models.Update) {
	bots := d.registry.List()
	for _, b := range bots {
		if b.Token == excludeToken {
			continue
		}
		d.Dispatch(b.Token, update)
	}
}
