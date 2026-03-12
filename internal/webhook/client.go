package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/skrashevich/telegram-mock-ai/internal/models"
)

// Client delivers updates to webhook URLs.
type Client struct {
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration
}

// NewClient creates a new webhook client.
func NewClient(timeout time.Duration, maxRetries int, retryDelay time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}
}

// Send POSTs an update to the webhook URL.
func (c *Client) Send(url, secretToken string, update models.Update) error {
	body, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.retryDelay * time.Duration(1<<(attempt-1)) // exponential backoff
			slog.Debug("webhook retry", "attempt", attempt, "delay", delay, "url", url)
			time.Sleep(delay)
		}

		req, err := http.NewRequest("POST", url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if secretToken != "" {
			req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secretToken)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("send request: %w", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return lastErr
}
