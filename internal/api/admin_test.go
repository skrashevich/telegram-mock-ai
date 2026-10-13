package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/skrashevich/telegram-mock-ai/internal/api"
	"github.com/skrashevich/telegram-mock-ai/internal/models"
	"github.com/skrashevich/telegram-mock-ai/internal/state"
)

func TestListChatMessagesFiltersBeforeLimit(t *testing.T) {
	for _, from := range []string{"bot", "user"} {
		t.Run(from, func(t *testing.T) {
			store := state.NewStore()
			chat := models.Chat{ID: 900001, Type: "private"}
			target := models.User{ID: 1, IsBot: from == "bot"}
			other := models.User{ID: 2, IsBot: !target.IsBot}
			// Explicit, out-of-order IDs ensure results use message order.
			for _, id := range []int{3, 1, 2} {
				store.StoreMessage(models.Message{MessageID: id, Chat: chat, From: &target})
			}
			// All matching messages are outside the old 200-message window.
			for id := range 200 {
				store.StoreMessage(models.Message{MessageID: id + 4, Chat: chat, From: &other})
			}
			handler := api.NewAdminServer(store, nil, nil, nil, nil).Handler()
			for _, tc := range []struct {
				limit int
				want  []int
			}{
				{1, []int{3}},
				{2, []int{2, 3}},
				{5, []int{1, 2, 3}},
			} {
				t.Run(fmt.Sprintf("limit=%d", tc.limit), func(t *testing.T) {
					assertHistoryIDs(t, handler, fmt.Sprintf("from=%s&limit=%d", from, tc.limit), tc.want)
				})
			}
			assertHistoryIDs(t, handler, "limit=2", []int{202, 203})
			assertHistoryIDs(t, handler, "from=unknown&limit=2", []int{202, 203})
		})
	}
}

func TestListChatMessagesFilteredEmpty(t *testing.T) {
	for _, from := range []string{"bot", "user"} {
		t.Run(from, func(t *testing.T) {
			store := state.NewStore()
			handler := api.NewAdminServer(store, nil, nil, nil, nil).Handler()
			assertHistoryIDs(t, handler, "from="+from, []int{})
			other := models.User{ID: 1, IsBot: from != "bot"}
			store.StoreMessage(models.Message{Chat: models.Chat{ID: 900001}, From: &other})
			assertHistoryIDs(t, handler, "from="+from, []int{})
		})
	}
}

func assertHistoryIDs(t *testing.T, handler http.Handler, query string, want []int) {
	t.Helper()
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/chats/900001/messages?"+query, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var messages []models.Message
	if err := json.Unmarshal(rr.Body.Bytes(), &messages); err != nil {
		t.Fatal(err)
	}
	if len(want) == 0 && messages == nil {
		t.Fatal("expected an empty JSON array, got null")
	}
	ids := make([]int, len(messages))
	for i, message := range messages {
		ids[i] = message.MessageID
	}
	if !slices.Equal(ids, want) {
		t.Fatalf("query %q: message IDs = %v, want %v", query, ids, want)
	}
}
