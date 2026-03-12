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

// APIType defines the LLM API protocol.
const (
	APITypeOpenAI    = "openai"
	APITypeAnthropic = "anthropic"
)

// Client communicates with an OpenAI-compatible or Anthropic chat completion endpoint.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
	temp       float64
	maxTokens  int
	apiType    string
}

// ChatMessage represents a message in the chat completion API.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewClient creates a new LLM client.
// apiType should be "openai" (default) or "anthropic".
func NewClient(baseURL, apiKey, model, apiType string, temperature float64, maxTokens int, timeout time.Duration) *Client {
	if apiType == "" {
		apiType = APITypeOpenAI
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		temp:       temperature,
		maxTokens:  maxTokens,
		apiType:    apiType,
	}
}

// Complete sends a chat completion request and returns the generated text.
func (c *Client) Complete(ctx context.Context, messages []ChatMessage) (string, error) {
	if c.apiType == APITypeAnthropic {
		return c.completeAnthropic(ctx, messages)
	}
	return c.completeOpenAI(ctx, messages)
}

// GenerateReply generates a reply to a bot message using chat context.
func (c *Client) GenerateReply(ctx context.Context, systemPrompt string, chatHistory []ChatMessage) (string, error) {
	messages := make([]ChatMessage, 0, len(chatHistory)+1)
	messages = append(messages, ChatMessage{Role: "system", Content: systemPrompt})
	messages = append(messages, chatHistory...)
	return c.Complete(ctx, messages)
}

// --- OpenAI protocol ---

type openAIRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type openAIResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

func (c *Client) completeOpenAI(ctx context.Context, messages []ChatMessage) (string, error) {
	reqBody := openAIRequest{
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

	text, err := c.doRequest(req)
	if err != nil {
		return "", err
	}

	var result openAIResponse
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	content := strings.TrimSpace(result.Choices[0].Message.Content)
	slog.Debug("LLM completion", "api", "openai", "model", c.model, "response_len", len(content))
	return content, nil
}

// --- Anthropic protocol ---

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	Temperature float64            `json:"temperature,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Client) completeAnthropic(ctx context.Context, messages []ChatMessage) (string, error) {
	// Anthropic separates system prompt from messages.
	var systemPrompt string
	var anthropicMsgs []anthropicMessage

	for _, m := range messages {
		if m.Role == "system" {
			systemPrompt = m.Content
			continue
		}
		anthropicMsgs = append(anthropicMsgs, anthropicMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	// Anthropic requires at least one user message.
	if len(anthropicMsgs) == 0 {
		anthropicMsgs = append(anthropicMsgs, anthropicMessage{
			Role:    "user",
			Content: "Hello",
		})
	}

	reqBody := anthropicRequest{
		Model:       c.model,
		MaxTokens:   c.maxTokens,
		System:      systemPrompt,
		Messages:    anthropicMsgs,
		Temperature: c.temp,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/messages"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}

	text, err := c.doRequest(req)
	if err != nil {
		return "", err
	}

	var result anthropicResponse
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("Anthropic API error: %s: %s", result.Error.Type, result.Error.Message)
	}

	// Extract text from content blocks.
	var sb strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}

	content := strings.TrimSpace(sb.String())
	if content == "" {
		return "", fmt.Errorf("Anthropic returned no text content")
	}

	slog.Debug("LLM completion", "api", "anthropic", "model", c.model, "response_len", len(content))
	return content, nil
}

// --- shared helpers ---

func (c *Client) doRequest(req *http.Request) (string, error) {
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

	return string(respBody), nil
}
