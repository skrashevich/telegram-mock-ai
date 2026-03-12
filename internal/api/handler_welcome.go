package api

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/skrashevich/telegram-mock-ai/internal/bot"
	"github.com/skrashevich/telegram-mock-ai/internal/llm"
	"github.com/skrashevich/telegram-mock-ai/internal/models"
)

// sendWelcomeMessages sends a message to each chat the bot is in.
// The first message is sent immediately; the rest are spread over 30 seconds.
func (s *Server) sendWelcomeMessages(b *bot.Bot) {
	chats := s.store.GetUserChats(b.User.ID)
	if len(chats) == 0 {
		return
	}

	// Shuffle so the "first" chat is random
	rand.Shuffle(len(chats), func(i, j int) { chats[i], chats[j] = chats[j], chats[i] })

	// Send first message immediately
	s.sendWelcomeToChat(b, chats[0])

	if len(chats) <= 1 {
		return
	}

	// Spread remaining messages over 30 seconds
	remaining := chats[1:]
	interval := 30 * time.Second / time.Duration(len(remaining))

	for _, chat := range remaining {
		time.Sleep(interval)
		s.sendWelcomeToChat(b, chat)
	}
}

// sendWelcomeToChat generates and sends a single welcome message to a chat.
func (s *Server) sendWelcomeToChat(b *bot.Bot, chat models.Chat) {
	nonBotMembers := s.store.GetNonBotChatMembers(chat.ID)
	if len(nonBotMembers) == 0 {
		return
	}

	sender := nonBotMembers[rand.Intn(len(nonBotMembers))].User
	text := generateWelcomeText(sender.FirstName, chat.Title)

	// Try LLM generation if available
	if s.llm != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		recent := s.store.GetChatMessages(chat.ID, 10)
		recentContext := ""
		for _, m := range recent {
			if m.Text != "" && m.From != nil {
				recentContext += m.From.FirstName + ": " + m.Text + "\n"
			}
		}

		prompt := llm.ProactiveMessagePrompt(sender.FirstName, chat.Title, recentContext)
		if generated, err := s.llm.Complete(ctx, []llm.ChatMessage{
			{Role: "system", Content: prompt},
			{Role: "user", Content: "Generate a message."},
		}); err == nil && generated != "" {
			if len(generated) > 4096 {
				generated = generated[:4096]
			}
			text = generated
		}
	}

	msg := s.store.StoreMessage(models.Message{
		From: &sender,
		Chat: chat,
		Date: time.Now().Unix(),
		Text: text,
	})

	s.dispatcher.Dispatch(b.Token, models.Update{Message: msg})
	slog.Info("welcome: sent message", "chat", chat.Title, "from", sender.FirstName, "bot", b.User.Username)
}

// generateWelcomeText returns a fallback welcome message when LLM is unavailable.
func generateWelcomeText(_, _ string) string {
	templates := []string{
		"Привет всем!",
		"Всем привет, как дела?",
		"Кто тут?",
		"Доброго времени суток!",
		"Привет! Что нового?",
		"Здарова!",
		"Ку-ку!",
		"О, привет!",
	}
	return templates[rand.Intn(len(templates))]
}
