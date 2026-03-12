package proactive

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/skrashevich/telegram-mock-ai/internal/bot"
	"github.com/skrashevich/telegram-mock-ai/internal/config"
	"github.com/skrashevich/telegram-mock-ai/internal/llm"
	"github.com/skrashevich/telegram-mock-ai/internal/models"
	"github.com/skrashevich/telegram-mock-ai/internal/state"
	"github.com/skrashevich/telegram-mock-ai/internal/updates"
)

// Engine generates proactive events on a timer.
type Engine struct {
	cfg        config.ProactiveConfig
	store      *state.Store
	llmClient  *llm.Client
	dispatcher *updates.Dispatcher
	registry   *bot.Registry
	llmTimeout time.Duration
}

// NewEngine creates a new proactive event engine.
func NewEngine(
	cfg config.ProactiveConfig,
	store *state.Store,
	llmClient *llm.Client,
	dispatcher *updates.Dispatcher,
	registry *bot.Registry,
	llmTimeout time.Duration,
) *Engine {
	return &Engine{
		cfg:        cfg,
		store:      store,
		llmClient:  llmClient,
		dispatcher: dispatcher,
		registry:   registry,
		llmTimeout: llmTimeout,
	}
}

// Run starts the proactive event loop. Blocks until ctx is cancelled.
func (e *Engine) Run(ctx context.Context) {
	slog.Info("proactive engine started",
		"interval_min", e.cfg.IntervalMin,
		"interval_max", e.cfg.IntervalMax,
	)

	for {
		delay := e.randomDelay()
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
			e.tick(ctx)
		case <-ctx.Done():
			timer.Stop()
			slog.Info("proactive engine stopped")
			return
		}
	}
}

func (e *Engine) randomDelay() time.Duration {
	min := e.cfg.IntervalMin
	max := e.cfg.IntervalMax
	if max <= min {
		return min
	}
	return min + time.Duration(rand.Int63n(int64(max-min)))
}

func (e *Engine) tick(ctx context.Context) {
	bots := e.registry.List()
	if len(bots) == 0 {
		return
	}

	// Pick a random bot
	b := bots[rand.Intn(len(bots))]

	// Get chats where this bot is a member
	botChats := e.store.GetUserChats(b.User.ID)
	if len(botChats) == 0 {
		return
	}

	// Pick a random chat
	chat := botChats[rand.Intn(len(botChats))]

	// Get non-bot members
	nonBotMembers := e.store.GetNonBotChatMembers(chat.ID)
	if len(nonBotMembers) == 0 {
		return
	}

	// Pick scenario
	scenario := e.pickScenario()

	switch scenario {
	case "user_message":
		e.generateMessage(ctx, b, chat, nonBotMembers)
	case "new_member":
		e.generateNewMember(ctx, b, chat)
	case "member_left":
		e.generateMemberLeft(ctx, b, chat, nonBotMembers)
	case "photo_message":
		e.generatePhotoMessage(ctx, b, chat, nonBotMembers)
	case "sticker_message":
		e.generateStickerMessage(b, chat, nonBotMembers)
	default:
		e.generateMessage(ctx, b, chat, nonBotMembers)
	}
}

func (e *Engine) pickScenario() string {
	scenarios := e.cfg.Scenarios
	if len(scenarios) == 0 {
		return "user_message"
	}

	totalWeight := 0.0
	for _, s := range scenarios {
		totalWeight += s.Weight
	}

	r := rand.Float64() * totalWeight
	cumulative := 0.0
	for _, s := range scenarios {
		cumulative += s.Weight
		if r <= cumulative {
			return s.Type
		}
	}
	return scenarios[len(scenarios)-1].Type
}

func (e *Engine) generateMessage(ctx context.Context, b *bot.Bot, chat models.Chat, members []models.ChatMember) {
	sender := members[rand.Intn(len(members))].User

	// Get recent messages for context
	recent := e.store.GetChatMessages(chat.ID, 10)
	recentContext := ""
	for _, m := range recent {
		if m.Text != "" && m.From != nil {
			recentContext += m.From.FirstName + ": " + m.Text + "\n"
		}
	}

	prompt := llm.ProactiveMessagePrompt(sender.FirstName, chat.Title, recentContext)

	llmCtx, cancel := context.WithTimeout(ctx, e.llmTimeout)
	defer cancel()

	text, err := e.llmClient.Complete(llmCtx, []llm.ChatMessage{
		{Role: "system", Content: prompt},
		{Role: "user", Content: "Generate a message."},
	})
	if err != nil {
		slog.Error("proactive: LLM generation failed", "err", err)
		return
	}

	if text == "" {
		return
	}

	if len(text) > 4096 {
		text = text[:4096]
	}

	msg := e.store.StoreMessage(models.Message{
		From: &sender,
		Chat: chat,
		Date: time.Now().Unix(),
		Text: text,
	})

	e.dispatcher.Dispatch(b.Token, models.Update{Message: msg})
	slog.Info("proactive: sent message", "chat", chat.Title, "from", sender.FirstName, "bot", b.User.Username)
}

func (e *Engine) generateNewMember(ctx context.Context, b *bot.Bot, chat models.Chat) {
	// Create a new simulated user
	user := e.store.CreateUser(models.User{
		FirstName: randomName(),
		Username:  "user_" + models.RandomHex(4),
	})

	e.store.AddChatMember(chat.ID, user.ID, "member")

	msg := e.store.StoreMessage(models.Message{
		From:           user,
		Chat:           chat,
		Date:           time.Now().Unix(),
		NewChatMembers: []models.User{*user},
	})

	e.dispatcher.Dispatch(b.Token, models.Update{Message: msg})
	slog.Info("proactive: new member", "chat", chat.Title, "user", user.FirstName)
}

func (e *Engine) generateMemberLeft(ctx context.Context, b *bot.Bot, chat models.Chat, members []models.ChatMember) {
	if len(members) <= 1 {
		return // Don't remove the last member
	}

	leaver := members[rand.Intn(len(members))].User
	e.store.RemoveChatMember(chat.ID, leaver.ID)

	msg := e.store.StoreMessage(models.Message{
		From:           &leaver,
		Chat:           chat,
		Date:           time.Now().Unix(),
		LeftChatMember: &leaver,
	})

	e.dispatcher.Dispatch(b.Token, models.Update{Message: msg})
	slog.Info("proactive: member left", "chat", chat.Title, "user", leaver.FirstName)
}

func (e *Engine) generatePhotoMessage(ctx context.Context, b *bot.Bot, chat models.Chat, members []models.ChatMember) {
	sender := members[rand.Intn(len(members))].User

	prompt := llm.ProactiveMessagePrompt(sender.FirstName, chat.Title, "")

	llmCtx, cancel := context.WithTimeout(ctx, e.llmTimeout)
	defer cancel()

	caption, err := e.llmClient.Complete(llmCtx, []llm.ChatMessage{
		{Role: "system", Content: prompt},
		{Role: "user", Content: "Generate a short photo caption (what the user is sharing a photo of). Just the caption, nothing else."},
	})
	if err != nil {
		caption = ""
	}

	fileID := "AgACAgIAAxkBAAI" + models.RandomHex(16)
	fileUniqueID := "AQADAgAT" + models.RandomHex(8)

	msg := e.store.StoreMessage(models.Message{
		From:    &sender,
		Chat:    chat,
		Date:    time.Now().Unix(),
		Caption: caption,
		Photo: []models.PhotoSize{
			{FileID: fileID, FileUniqueID: fileUniqueID, Width: 1280, Height: 960},
			{FileID: fileID + "_s", FileUniqueID: fileUniqueID + "_s", Width: 320, Height: 240},
		},
	})

	e.dispatcher.Dispatch(b.Token, models.Update{Message: msg})
	slog.Info("proactive: photo message", "chat", chat.Title, "from", sender.FirstName)
}

func (e *Engine) generateStickerMessage(b *bot.Bot, chat models.Chat, members []models.ChatMember) {
	sender := members[rand.Intn(len(members))].User
	emojis := []string{"😀", "😂", "❤️", "👍", "🔥", "🎉", "😎", "🤔", "😊", "👋"}
	emoji := emojis[rand.Intn(len(emojis))]

	fileID := "CAACAgIAAxkBAAI" + models.RandomHex(16)
	fileUniqueID := "AgADAgAT" + models.RandomHex(8)

	msg := e.store.StoreMessage(models.Message{
		From: &sender,
		Chat: chat,
		Date: time.Now().Unix(),
		Sticker: &models.Sticker{
			FileID:       fileID,
			FileUniqueID: fileUniqueID,
			Type:         "regular",
			Width:        512,
			Height:       512,
			Emoji:        emoji,
		},
	})

	e.dispatcher.Dispatch(b.Token, models.Update{Message: msg})
	slog.Info("proactive: sticker message", "chat", chat.Title, "from", sender.FirstName, "emoji", emoji)
}

var firstNames = []string{
	"Alex", "Maria", "Ivan", "Elena", "Dmitry", "Anna", "Sergey", "Olga",
	"Nikolai", "Natasha", "Andrey", "Ekaterina", "Pavel", "Julia", "Maxim",
	"Tatiana", "Viktor", "Svetlana", "Roman", "Irina",
}

func randomName() string {
	return firstNames[rand.Intn(len(firstNames))]
}
