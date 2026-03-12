package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/skrashevich/telegram-mock-ai/internal/api"
	"github.com/skrashevich/telegram-mock-ai/internal/bot"
	"github.com/skrashevich/telegram-mock-ai/internal/config"
	"github.com/skrashevich/telegram-mock-ai/internal/models"
	"github.com/skrashevich/telegram-mock-ai/internal/state"
	"github.com/skrashevich/telegram-mock-ai/internal/updates"
	"github.com/skrashevich/telegram-mock-ai/internal/webhook"
)

func setupTestServer() (*httptest.Server, *state.Store, *bot.Registry) {
	cfg := config.DefaultConfig()
	cfg.LLM.Enabled = false
	store := state.NewStore()
	registry := bot.NewRegistry(true)
	webhookClient := webhook.NewClient(cfg.Webhook.Timeout, cfg.Webhook.MaxRetries, cfg.Webhook.RetryDelay)
	dispatcher := updates.NewDispatcher(registry, webhookClient)
	server := api.NewServer(cfg, store, registry, dispatcher, nil)
	ts := httptest.NewServer(server.Handler())
	return ts, store, registry
}

type apiResp struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	ErrorCode   int             `json:"error_code"`
	Description string          `json:"description"`
}

func doPost(t *testing.T, url string, body string) apiResp {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var result apiResp
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal failed: %v\nbody: %s", err, string(data))
	}
	return result
}

func doGet(t *testing.T, url string) apiResp {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var result apiResp
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal failed: %v\nbody: %s", err, string(data))
	}
	return result
}

func TestGetMe(t *testing.T) {
	ts, _, _ := setupTestServer()
	defer ts.Close()

	resp := doGet(t, ts.URL+"/bot123456:testtoken/getMe")
	if !resp.OK {
		t.Fatalf("expected ok=true, got %v: %s", resp.OK, resp.Description)
	}

	var user models.User
	json.Unmarshal(resp.Result, &user)
	if !user.IsBot {
		t.Error("expected is_bot=true")
	}
	if user.ID == 0 {
		t.Error("expected non-zero user ID")
	}
}

func TestUnauthorized(t *testing.T) {
	ts, _, _ := setupTestServer()
	defer ts.Close()

	// No token
	resp, err := http.Get(ts.URL + "/bot/getMe")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 for empty token")
	}
}

func TestSendMessage(t *testing.T) {
	ts, store, _ := setupTestServer()
	defer ts.Close()

	token := "123456:testtoken"

	// Create a chat and user for the test
	store.CreateUser(models.User{ID: 1001, FirstName: "Alice", Username: "alice"})
	store.CreateChat(models.Chat{ID: 1001, Type: "private"})

	resp := doPost(t, ts.URL+"/bot"+token+"/sendMessage",
		`{"chat_id": 1001, "text": "Hello, world!"}`)

	if !resp.OK {
		t.Fatalf("expected ok=true: %s", resp.Description)
	}

	var msg models.Message
	json.Unmarshal(resp.Result, &msg)

	if msg.Text != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got '%s'", msg.Text)
	}
	if msg.MessageID == 0 {
		t.Error("expected non-zero message_id")
	}
	if msg.Chat.ID != 1001 {
		t.Errorf("expected chat_id 1001, got %d", msg.Chat.ID)
	}
}

func TestSendMessageEmptyText(t *testing.T) {
	ts, _, _ := setupTestServer()
	defer ts.Close()

	resp := doPost(t, ts.URL+"/bot123456:test/sendMessage",
		`{"chat_id": 1, "text": ""}`)

	if resp.OK {
		t.Error("expected ok=false for empty text")
	}
	if resp.ErrorCode != 400 {
		t.Errorf("expected error_code 400, got %d", resp.ErrorCode)
	}
}

func TestGetUpdatesEmpty(t *testing.T) {
	ts, _, _ := setupTestServer()
	defer ts.Close()

	resp := doPost(t, ts.URL+"/bot123456:test/getUpdates",
		`{"timeout": 0}`)

	if !resp.OK {
		t.Fatalf("expected ok=true: %s", resp.Description)
	}

	var result []models.Update
	json.Unmarshal(resp.Result, &result)
	if len(result) != 0 {
		t.Errorf("expected empty updates, got %d", len(result))
	}
}

func TestGetUpdatesConflictWithWebhook(t *testing.T) {
	ts, _, registry := setupTestServer()
	defer ts.Close()

	token := "webhook-bot:token"
	registry.Register(token, "webhook_bot", "Webhook Bot")
	registry.SetWebhook(token, "http://localhost:9999/webhook", "", nil)

	resp := doPost(t, ts.URL+"/bot"+token+"/getUpdates", `{}`)
	if resp.OK {
		t.Error("expected error when calling getUpdates with active webhook")
	}
	if resp.ErrorCode != 409 {
		t.Errorf("expected 409 Conflict, got %d", resp.ErrorCode)
	}
}

func TestSetWebhook(t *testing.T) {
	ts, _, _ := setupTestServer()
	defer ts.Close()

	token := "123456:webhook"

	resp := doPost(t, ts.URL+"/bot"+token+"/setWebhook",
		`{"url": "http://localhost:9999/webhook"}`)

	if !resp.OK {
		t.Fatalf("expected ok=true: %s", resp.Description)
	}

	// Verify via getWebhookInfo
	info := doGet(t, ts.URL+"/bot"+token+"/getWebhookInfo")
	if !info.OK {
		t.Fatalf("expected ok=true: %s", info.Description)
	}

	var webhookInfo models.WebhookInfo
	json.Unmarshal(info.Result, &webhookInfo)
	if webhookInfo.URL != "http://localhost:9999/webhook" {
		t.Errorf("expected webhook URL, got '%s'", webhookInfo.URL)
	}
}

func TestDeleteWebhook(t *testing.T) {
	ts, _, registry := setupTestServer()
	defer ts.Close()

	token := "123456:delwebhook"
	registry.Register(token, "bot", "Bot")
	registry.SetWebhook(token, "http://example.com/hook", "", nil)

	resp := doPost(t, ts.URL+"/bot"+token+"/deleteWebhook", `{}`)
	if !resp.OK {
		t.Fatalf("expected ok=true: %s", resp.Description)
	}

	b, _ := registry.Get(token)
	if b.WebhookURL != "" {
		t.Error("expected webhook to be cleared")
	}
}

func TestEditMessageText(t *testing.T) {
	ts, store, registry := setupTestServer()
	defer ts.Close()

	token := "123456:edit"
	b := registry.Register(token, "edit_bot", "Edit Bot")
	store.CreateUser(b.User)
	chat := store.CreateChat(models.Chat{ID: 999, Type: "private"})

	// Create a message "from the bot"
	msg := store.StoreMessage(models.Message{
		From: &b.User,
		Chat: *chat,
		Date: 1000,
		Text: "original",
	})

	resp := doPost(t, ts.URL+"/bot"+token+"/editMessageText",
		`{"chat_id": 999, "message_id": `+itoa(msg.MessageID)+`, "text": "edited"}`)

	if !resp.OK {
		t.Fatalf("expected ok=true: %s", resp.Description)
	}

	var edited models.Message
	json.Unmarshal(resp.Result, &edited)
	if edited.Text != "edited" {
		t.Errorf("expected text 'edited', got '%s'", edited.Text)
	}
}

func TestDeleteMessage(t *testing.T) {
	ts, store, registry := setupTestServer()
	defer ts.Close()

	token := "123456:del"
	b := registry.Register(token, "del_bot", "Del Bot")
	store.CreateUser(b.User)
	chat := store.CreateChat(models.Chat{ID: 888, Type: "private"})

	msg := store.StoreMessage(models.Message{
		From: &b.User,
		Chat: *chat,
		Date: 1000,
		Text: "to delete",
	})

	resp := doPost(t, ts.URL+"/bot"+token+"/deleteMessage",
		`{"chat_id": 888, "message_id": `+itoa(msg.MessageID)+`}`)

	if !resp.OK {
		t.Fatalf("expected ok=true: %s", resp.Description)
	}

	// Verify message is deleted
	_, exists := store.GetMessage(888, msg.MessageID)
	if exists {
		t.Error("expected message to be deleted")
	}
}

func TestGetChat(t *testing.T) {
	ts, store, _ := setupTestServer()
	defer ts.Close()

	store.CreateChat(models.Chat{ID: -500, Type: "group", Title: "My Group"})

	resp := doPost(t, ts.URL+"/bot123456:test/getChat",
		`{"chat_id": -500}`)

	if !resp.OK {
		t.Fatalf("expected ok=true: %s", resp.Description)
	}

	var chat models.Chat
	json.Unmarshal(resp.Result, &chat)
	if chat.Title != "My Group" {
		t.Errorf("expected title 'My Group', got '%s'", chat.Title)
	}
}

func TestGetChatNotFound(t *testing.T) {
	ts, _, _ := setupTestServer()
	defer ts.Close()

	resp := doPost(t, ts.URL+"/bot123456:test/getChat",
		`{"chat_id": 99999}`)

	if resp.OK {
		t.Error("expected ok=false for non-existent chat")
	}
}

func TestMethodNotFound(t *testing.T) {
	ts, _, _ := setupTestServer()
	defer ts.Close()

	resp := doGet(t, ts.URL+"/bot123456:test/nonExistentMethod")
	if resp.OK {
		t.Error("expected ok=false for unknown method")
	}
	if resp.ErrorCode != 400 {
		t.Errorf("expected 400, got %d", resp.ErrorCode)
	}
}

func TestAnswerCallbackQuery(t *testing.T) {
	ts, _, _ := setupTestServer()
	defer ts.Close()

	resp := doPost(t, ts.URL+"/bot123456:test/answerCallbackQuery",
		`{"callback_query_id": "abc123"}`)

	if !resp.OK {
		t.Fatalf("expected ok=true: %s", resp.Description)
	}
}

func TestSendPhoto(t *testing.T) {
	ts, store, _ := setupTestServer()
	defer ts.Close()

	store.CreateChat(models.Chat{ID: 100, Type: "private"})

	resp := doPost(t, ts.URL+"/bot123456:test/sendPhoto",
		`{"chat_id": 100, "photo": "file_id_xxx", "caption": "Nice photo"}`)

	if !resp.OK {
		t.Fatalf("expected ok=true: %s", resp.Description)
	}

	var msg models.Message
	json.Unmarshal(resp.Result, &msg)
	if len(msg.Photo) == 0 {
		t.Error("expected photo in message")
	}
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
