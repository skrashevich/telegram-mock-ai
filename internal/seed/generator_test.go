package seed

import (
	"encoding/json"
	"testing"

	"github.com/skrashevich/telegram-mock-ai/internal/bot"
	"github.com/skrashevich/telegram-mock-ai/internal/state"
)

func TestStripCodeFences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no fences",
			input: `{"users": []}`,
			want:  `{"users": []}`,
		},
		{
			name:  "json fences",
			input: "```json\n{\"users\": []}\n```",
			want:  `{"users": []}`,
		},
		{
			name:  "plain fences",
			input: "```\n{\"users\": []}\n```",
			want:  `{"users": []}`,
		},
		{
			name:  "with leading text",
			input: "Here is the JSON:\n```json\n{\"users\": []}\n```",
			want:  `{"users": []}`,
		},
		{
			name:  "whitespace only",
			input: "  \n{\"users\": []}  \n",
			want:  `{"users": []}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripCodeFences(tt.input)
			if got != tt.want {
				t.Errorf("StripCodeFences() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	t.Run("valid response", func(t *testing.T) {
		resp := &llmSeedResponse{
			Users: []llmUser{
				{FirstName: "Alice", LastName: "Smith", Username: "alice_s"},
				{FirstName: "Bob", LastName: "Jones", Username: "bob_j"},
			},
			Groups: []llmChat{
				{Title: "Test Group", MemberIndices: []int{0, 1}},
			},
			Channels: []llmChat{
				{Title: "News Channel", MemberIndices: []int{0}},
			},
		}
		if err := resp.validate(); err != nil {
			t.Errorf("expected valid, got error: %v", err)
		}
	})

	t.Run("empty users", func(t *testing.T) {
		resp := &llmSeedResponse{
			Users: []llmUser{},
		}
		if err := resp.validate(); err == nil {
			t.Error("expected error for empty users")
		}
	})

	t.Run("group index out of range", func(t *testing.T) {
		resp := &llmSeedResponse{
			Users: []llmUser{
				{FirstName: "Alice", Username: "alice"},
			},
			Groups: []llmChat{
				{Title: "Bad Group", MemberIndices: []int{0, 5}},
			},
		}
		if err := resp.validate(); err == nil {
			t.Error("expected error for out-of-range index")
		}
	})

	t.Run("channel negative index", func(t *testing.T) {
		resp := &llmSeedResponse{
			Users: []llmUser{
				{FirstName: "Alice", Username: "alice"},
			},
			Channels: []llmChat{
				{Title: "Bad Channel", MemberIndices: []int{-1}},
			},
		}
		if err := resp.validate(); err == nil {
			t.Error("expected error for negative index")
		}
	})
}

func TestParseAndMaterialize(t *testing.T) {
	raw := `{
		"users": [
			{"first_name": "Анна", "last_name": "Иванова", "username": "anna_iv"},
			{"first_name": "Пётр", "last_name": "Сидоров", "username": "petr_s"},
			{"first_name": "Мария", "last_name": "Козлова", "username": "masha_k"}
		],
		"groups": [
			{"title": "Друзья", "member_indices": [0, 1, 2]},
			{"title": "Работа", "member_indices": [0, 2]}
		],
		"channels": [
			{"title": "Новости", "member_indices": [0, 1]}
		]
	}`

	var parsed llmSeedResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if err := parsed.validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// Materialize
	store := state.NewStore()
	registry := bot.NewRegistry(true)
	registry.Register("test:token", "test_bot", "Test Bot")

	gen := NewGenerator(nil, store, registry, 1)
	result := gen.materialize(&parsed)

	// Verify users
	if len(result.Users) != 3 {
		t.Errorf("expected 3 users, got %d", len(result.Users))
	}
	for _, u := range result.Users {
		if u.ID == 0 {
			t.Error("expected non-zero user ID")
		}
	}

	// Verify chats (2 groups + 1 channel)
	if len(result.Chats) != 3 {
		t.Errorf("expected 3 chats, got %d", len(result.Chats))
	}

	// Verify group members
	groupMembers := store.GetChatMembers(result.Chats[0].ID)
	// 3 user members (bot user is in registry but not in store, so AddChatMember skips it)
	if len(groupMembers) != 3 {
		t.Errorf("expected 3 members in first group, got %d", len(groupMembers))
	}

	// Verify channel members
	channelMembers := store.GetChatMembers(result.Chats[2].ID)
	// 2 user members
	if len(channelMembers) != 2 {
		t.Errorf("expected 2 members in channel, got %d", len(channelMembers))
	}

	// Verify users exist in store
	allUsers := store.ListUsers()
	// 3 generated users + 1 bot user
	if len(allUsers) < 3 {
		t.Errorf("expected at least 3 users in store, got %d", len(allUsers))
	}
}
