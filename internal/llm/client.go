package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Client communicates with an OpenAI-compatible chat completion endpoint.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
	temp       float64
	maxTokens  int
}

// ChatMessage represents a message in the chat completion API.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type completionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type completionResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

// NewClient creates a new LLM client.
func NewClient(baseURL, apiKey, model string, temperature float64, maxTokens int, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		temp:       temperature,
		maxTokens:  maxTokens,
	}
}

// Complete sends a chat completion request and returns the generated text.
func (c *Client) Complete(ctx context.Context, messages []ChatMessage) (string, error) {
	reqBody := completionRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: c.temp,
		MaxTokens:   c.maxTokens,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result completionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	text := strings.TrimSpace(result.Choices[0].Message.Content)
	slog.Debug("LLM completion", "model", c.model, "response_len", len(text))
	return text, nil
}

// GenerateReply generates a reply to a bot message using chat context.
func (c *Client) GenerateReply(ctx context.Context, systemPrompt string, chatHistory []ChatMessage) (string, error) {
	messages := make([]ChatMessage, 0, len(chatHistory)+1)
	messages = append(messages, ChatMessage{Role: "system", Content: systemPrompt})
	messages = append(messages, chatHistory...)
	return c.Complete(ctx, messages)
}
