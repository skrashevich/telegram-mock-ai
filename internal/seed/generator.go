package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/skrashevich/telegram-mock-ai/internal/bot"
	"github.com/skrashevich/telegram-mock-ai/internal/llm"
	"github.com/skrashevich/telegram-mock-ai/internal/models"
	"github.com/skrashevich/telegram-mock-ai/internal/state"
)

// GenerateRequest holds parameters for seed generation.
type GenerateRequest struct {
	UsersCount    int    `json:"users_count"`
	GroupsCount   int    `json:"groups_count"`
	ChannelsCount int    `json:"channels_count"`
	Locale        string `json:"locale"`
}

// GenerateResult holds the outcome of seed generation.
type GenerateResult struct {
	Users []models.User `json:"users"`
	Chats []models.Chat `json:"chats"`
}

// Generator handles LLM-powered seed data creation.
type Generator struct {
	llmClient  *llm.Client
	store      *state.Store
	registry   *bot.Registry
	maxRetries int
}

// NewGenerator creates a seed data generator.
func NewGenerator(llmClient *llm.Client, store *state.Store, registry *bot.Registry, maxRetries int) *Generator {
	if maxRetries <= 0 {
		maxRetries = 2
	}
	return &Generator{
		llmClient:  llmClient,
		store:      store,
		registry:   registry,
		maxRetries: maxRetries,
	}
}

// Generate calls the LLM to produce seed data, parses it, validates it,
// and populates the store. Returns the created entities.
func (g *Generator) Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
	if g.llmClient == nil {
		return nil, fmt.Errorf("LLM client is not configured")
	}

	// Apply defaults
	if req.UsersCount <= 0 {
		req.UsersCount = 5
	}
	if req.GroupsCount < 0 {
		req.GroupsCount = 0
	}
	if req.ChannelsCount < 0 {
		req.ChannelsCount = 0
	}

	// Build prompt
	prompt := llm.SeedGenerationPrompt(
		req.UsersCount, req.GroupsCount, req.ChannelsCount, req.Locale,
	)

	// Call LLM with retries
	var raw string
	var lastErr error
	for attempt := 0; attempt < g.maxRetries; attempt++ {
		raw, lastErr = g.llmClient.Complete(ctx, []llm.ChatMessage{
			{Role: "system", Content: prompt},
			{Role: "user", Content: "Generate the seed data now."},
		})
		if lastErr == nil && raw != "" {
			break
		}
		slog.Warn("seed generation: LLM attempt failed",
			"attempt", attempt+1, "err", lastErr)
	}
	if lastErr != nil && raw == "" {
		return nil, fmt.Errorf("LLM generation failed after %d attempts: %w",
			g.maxRetries, lastErr)
	}

	// Parse JSON from response (strip markdown fences if present)
	raw = StripCodeFences(raw)

	var parsed llmSeedResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w\nraw: %s",
			err, truncate(raw, 500))
	}

	// Validate
	if err := parsed.validate(); err != nil {
		return nil, fmt.Errorf("LLM response validation failed: %w", err)
	}

	// Materialize into store
	return g.materialize(&parsed), nil
}

// llmSeedResponse is the expected JSON structure from the LLM.
type llmSeedResponse struct {
	Users    []llmUser `json:"users"`
	Groups   []llmChat `json:"groups"`
	Channels []llmChat `json:"channels"`
}

type llmUser struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type llmChat struct {
	Title         string `json:"title"`
	MemberIndices []int  `json:"member_indices"`
}

func (r *llmSeedResponse) validate() error {
	if len(r.Users) == 0 {
		return fmt.Errorf("no users generated")
	}
	for i, g := range r.Groups {
		for _, idx := range g.MemberIndices {
			if idx < 0 || idx >= len(r.Users) {
				return fmt.Errorf("group[%d] has invalid member_index %d (users count: %d)",
					i, idx, len(r.Users))
			}
		}
	}
	for i, ch := range r.Channels {
		for _, idx := range ch.MemberIndices {
			if idx < 0 || idx >= len(r.Users) {
				return fmt.Errorf("channel[%d] has invalid member_index %d (users count: %d)",
					i, idx, len(r.Users))
			}
		}
	}
	return nil
}

func (g *Generator) materialize(parsed *llmSeedResponse) *GenerateResult {
	result := &GenerateResult{}

	// Create users (let store auto-generate IDs)
	createdUsers := make([]*models.User, len(parsed.Users))
	for i, u := range parsed.Users {
		user := g.store.CreateUser(models.User{
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Username:  strings.ToLower(u.Username),
		})
		createdUsers[i] = user
		result.Users = append(result.Users, *user)
		slog.Debug("seed: created user", "id", user.ID, "name", user.FirstName)
	}

	// Helper to create a chat and add members + bots
	firstChat := true
	createChat := func(lc llmChat, chatType string) {
		chat := g.store.CreateChat(models.Chat{
			Type:  chatType,
			Title: lc.Title,
		})

		for _, idx := range lc.MemberIndices {
			g.store.AddChatMember(chat.ID, createdUsers[idx].ID, "member")
		}

		// Add all registered bots to the chat.
		// First chat gets bots as administrators so they have full access.
		botStatus := "member"
		if firstChat {
			botStatus = "administrator"
			firstChat = false
		}
		for _, b := range g.registry.List() {
			g.store.AddChatMember(chat.ID, b.User.ID, botStatus)
		}

		result.Chats = append(result.Chats, *chat)
		slog.Debug("seed: created chat", "id", chat.ID, "title", chat.Title,
			"type", chatType, "members", len(lc.MemberIndices))
	}

	// Create groups
	for _, gc := range parsed.Groups {
		createChat(gc, "group")
	}

	// Create channels
	for _, cc := range parsed.Channels {
		createChat(cc, "channel")
	}

	slog.Info("seed generation complete",
		"users", len(result.Users),
		"chats", len(result.Chats),
	)

	return result
}

// StripCodeFences removes markdown code fences that LLMs sometimes add.
func StripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	// Try to find JSON object boundaries if there's extra text
	if idx := strings.Index(s, "{"); idx > 0 {
		// Check if there's text before the first brace (e.g. "```json\n{")
		prefix := s[:idx]
		if strings.Contains(prefix, "```") || strings.TrimSpace(prefix) == "" {
			s = s[idx:]
		}
	}
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
