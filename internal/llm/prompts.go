package llm

import "fmt"

// SeedGenerationPrompt builds a system prompt for generating seed data as JSON.
func SeedGenerationPrompt(usersCount, groupsCount, channelsCount int, locale string) string {
	localeHint := ""
	if locale != "" {
		localeHint = fmt.Sprintf("\nUse names and titles appropriate for the \"%s\" locale/language.", locale)
	}

	return fmt.Sprintf(`You are a data generator for a Telegram chat simulator.
Generate realistic seed data as a single JSON object with the following structure:
{
  "users": [
    {"first_name": "...", "last_name": "...", "username": "..."}
  ],
  "groups": [
    {"title": "...", "member_indices": [0, 1, 2]}
  ],
  "channels": [
    {"title": "...", "member_indices": [0, 1]}
  ]
}

Requirements:
- Generate exactly %d users with realistic first names, last names, and usernames.
- Generate exactly %d groups. Each group must have between 2 and %d members (use diverse subsets, not all users in every group).
- Generate exactly %d channels. Each channel must have between 1 and %d members.
- member_indices are zero-based indices into the users array.
- Usernames must be lowercase, use underscores or digits, and be unique.
- Group/channel titles should be realistic chat names (hobbies, work, friends, neighborhood, etc.).%s

Output ONLY valid JSON. No markdown, no code fences, no explanation.`,
		usersCount,
		groupsCount, max(usersCount, 2),
		channelsCount, max(usersCount, 1),
		localeHint,
	)
}

// stylePresets maps style names to behavioral instructions for message generation.
var stylePresets = map[string]string{
	"normal": "The message should feel natural - it could be a question, a comment, sharing something interesting, a greeting, or continuing a previous topic.",
	"spam": `Generate a SPAM message. It should look like typical Telegram spam: crypto/forex scams, "earn money fast" schemes, suspicious links (use fake domains like example-profit.xyz), pyramid schemes, fake giveaways, or unsolicited ads. Make it convincing but obviously spam. Mix languages if appropriate.`,
	"toxic": `Generate a TOXIC/OFFENSIVE message. It should contain profanity, insults, hate speech, trolling, or aggressive behavior typical of toxic chat members. Include swear words, personal attacks, or inflammatory statements. This is for testing content moderation bots.`,
	"flood": `Generate a FLOOD message. It should be repetitive, excessively long, contain repeated characters (like "АААААА"), emoji spam (dozens of emojis), repeated words, ALL CAPS shouting, or meaningless character sequences. This is for testing anti-flood bots.`,
	"mixed": `Randomly pick ONE of these message styles and generate accordingly:
- Normal conversational message (40% chance)
- Spam with suspicious links or scam offers (20% chance)
- Toxic/offensive message with profanity (20% chance)
- Flood-style repetitive or spammy content (20% chance)
Commit fully to the chosen style.`,
}

// ProactiveMessagePrompt generates a prompt for spontaneous user messages.
// style selects a preset behavior; customPrompt overrides style if non-empty.
func ProactiveMessagePrompt(userName, chatTitle, recentContext, style, customPrompt string) string {
	ctx := ""
	if recentContext != "" {
		ctx = fmt.Sprintf("\nRecent chat messages for context:\n%s\n", recentContext)
	}
	chatInfo := ""
	if chatTitle != "" {
		chatInfo = fmt.Sprintf(` in the chat "%s"`, chatTitle)
	}

	// Determine behavior instruction
	behavior := stylePresets["mixed"]
	if customPrompt != "" {
		behavior = customPrompt
	} else if style != "" {
		if preset, ok := stylePresets[style]; ok {
			behavior = preset
		}
	}

	return fmt.Sprintf(`You are simulating a Telegram user named "%s"%s.
Generate a realistic, spontaneous message that this user might send.%s
%s
Keep it concise (1-3 sentences max).
Output ONLY the message text.`, userName, chatInfo, ctx, behavior)
}

