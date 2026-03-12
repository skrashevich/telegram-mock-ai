package llm

import "fmt"

// ProactiveMessagePrompt generates a prompt for spontaneous user messages.
func ProactiveMessagePrompt(userName, chatTitle string, recentContext string) string {
	ctx := ""
	if recentContext != "" {
		ctx = fmt.Sprintf("\nRecent chat messages for context:\n%s\n", recentContext)
	}
	chatInfo := ""
	if chatTitle != "" {
		chatInfo = fmt.Sprintf(` in the chat "%s"`, chatTitle)
	}
	return fmt.Sprintf(`You are simulating a Telegram user named "%s"%s.
Generate a realistic, spontaneous message that this user might send.%s
The message should feel natural - it could be a question, a comment, sharing something interesting, a greeting, or continuing a previous topic.
Keep it concise (1-3 sentences max).
Output ONLY the message text.`, userName, chatInfo, ctx)
}

