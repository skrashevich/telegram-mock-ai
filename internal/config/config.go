package config

import (
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all server configuration.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	LLM       LLMConfig       `yaml:"llm"`
	Proactive ProactiveConfig `yaml:"proactive"`
	Webhook   WebhookConfig   `yaml:"webhook"`
	Seed      SeedConfig      `yaml:"seed"`
	Log       LogConfig       `yaml:"log"`
	Admin     AdminConfig     `yaml:"admin"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// LLMConfig holds LLM integration settings.
type LLMConfig struct {
	Enabled          bool          `yaml:"enabled"`
	APIType          string        `yaml:"api_type"` // "openai" (default) or "anthropic"
	BaseURL          string        `yaml:"base_url"`
	APIKey           string        `yaml:"api_key"`
	Model            string        `yaml:"model"`
	Temperature      float64       `yaml:"temperature"`
	MaxTokens        int           `yaml:"max_tokens"`
	Timeout          time.Duration `yaml:"timeout"`
	ResponseDelayMin time.Duration `yaml:"response_delay_min"`
	ResponseDelayMax time.Duration `yaml:"response_delay_max"`
	SystemPrompt     string        `yaml:"system_prompt"`
}

// ProactiveConfig holds proactive event generation settings.
type ProactiveConfig struct {
	Enabled      bool             `yaml:"enabled"`
	IntervalMin  time.Duration    `yaml:"interval_min"`
	IntervalMax  time.Duration    `yaml:"interval_max"`
	EventTypes   []string         `yaml:"event_types"`
	Scenarios    []ScenarioConfig `yaml:"scenarios"`
	Style        string           `yaml:"style"`         // "normal", "spam", "toxic", "flood", "mixed" or custom name
	CustomPrompt string           `yaml:"custom_prompt"` // free-form instruction overriding style presets
}

// ScenarioConfig defines a proactive event scenario.
type ScenarioConfig struct {
	Type   string  `yaml:"type"`
	Weight float64 `yaml:"weight"`
}

// WebhookConfig holds webhook delivery settings.
type WebhookConfig struct {
	MaxRetries int           `yaml:"max_retries"`
	RetryDelay time.Duration `yaml:"retry_delay"`
	Timeout    time.Duration `yaml:"timeout"`
}

// SeedConfig holds initial data for the mock server.
type SeedConfig struct {
	Users    []SeedUser    `yaml:"users"`
	Chats    []SeedChat    `yaml:"chats"`
	Bots     []SeedBot     `yaml:"bots"`
	Generate *SeedGenerate `yaml:"generate,omitempty"`
}

// SeedGenerate controls LLM-powered seed data generation.
type SeedGenerate struct {
	Enabled       bool   `yaml:"enabled"`
	UsersCount    int    `yaml:"users_count"`
	GroupsCount   int    `yaml:"groups_count"`
	ChannelsCount int    `yaml:"channels_count"`
	Locale        string `yaml:"locale"`
	MaxRetries    int    `yaml:"max_retries"`
}

// SeedUser defines a pre-created user.
type SeedUser struct {
	ID        int64  `yaml:"id"`
	FirstName string `yaml:"first_name"`
	LastName  string `yaml:"last_name"`
	Username  string `yaml:"username"`
}

// SeedChat defines a pre-created chat.
type SeedChat struct {
	ID      int64   `yaml:"id"`
	Type    string  `yaml:"type"`
	Title   string  `yaml:"title"`
	Members []int64 `yaml:"members"`
}

// SeedBot defines a pre-registered bot.
type SeedBot struct {
	Token     string `yaml:"token"`
	Username  string `yaml:"username"`
	FirstName string `yaml:"first_name"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// AdminConfig holds admin API settings.
type AdminConfig struct {
	Enabled bool   `yaml:"enabled"`
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
}

// DefaultConfig returns a Config with sane defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8081,
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 60 * time.Second,
		},
		LLM: LLMConfig{
			Enabled:         true,
			BaseURL:         "http://localhost:11434/v1",
			Model:           "gpt-4o-mini",
			Temperature:     0.8,
			MaxTokens:       512,
			Timeout:         30 * time.Second,
			ResponseDelayMin: 500 * time.Millisecond,
			ResponseDelayMax: 3 * time.Second,
		},
		Proactive: ProactiveConfig{
			Enabled:     false,
			IntervalMin: 10 * time.Second,
			IntervalMax: 60 * time.Second,
			EventTypes:  []string{"message"},
			Scenarios: []ScenarioConfig{
				{Type: "user_message", Weight: 0.6},
				{Type: "new_member", Weight: 0.1},
				{Type: "member_left", Weight: 0.05},
				{Type: "photo_message", Weight: 0.15},
				{Type: "sticker_message", Weight: 0.1},
			},
		},
		Webhook: WebhookConfig{
			MaxRetries: 3,
			RetryDelay: time.Second,
			Timeout:    10 * time.Second,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		Admin: AdminConfig{
			Enabled: true,
			Host:    "127.0.0.1",
			Port:    8082,
		},
	}
}

// Load reads config from a YAML file, merging with defaults. Env vars override.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		} else {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}
	}

	applyEnvOverrides(cfg)
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("TELEGRAM_MOCK_SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("TELEGRAM_MOCK_SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("TELEGRAM_MOCK_LLM_BASE_URL"); v != "" {
		cfg.LLM.BaseURL = v
	}
	if v := os.Getenv("TELEGRAM_MOCK_LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("TELEGRAM_MOCK_LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("TELEGRAM_MOCK_LLM_ENABLED"); v != "" {
		cfg.LLM.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("TELEGRAM_MOCK_LLM_API_TYPE"); v != "" {
		cfg.LLM.APIType = v
	}
	if v := os.Getenv("TELEGRAM_MOCK_PROACTIVE_ENABLED"); v != "" {
		cfg.Proactive.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("TELEGRAM_MOCK_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("TELEGRAM_MOCK_ADMIN_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Admin.Port = port
		}
	}
	if v := os.Getenv("TELEGRAM_MOCK_SEED_GENERATE_ENABLED"); v != "" {
		if cfg.Seed.Generate == nil {
			cfg.Seed.Generate = &SeedGenerate{}
		}
		cfg.Seed.Generate.Enabled = v == "true" || v == "1"
	}
}
