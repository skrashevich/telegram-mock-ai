package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/skrashevich/telegram-mock-ai/internal/api"
	"github.com/skrashevich/telegram-mock-ai/internal/bot"
	"github.com/skrashevich/telegram-mock-ai/internal/config"
	"github.com/skrashevich/telegram-mock-ai/internal/llm"
	"github.com/skrashevich/telegram-mock-ai/internal/models"
	"github.com/skrashevich/telegram-mock-ai/internal/proactive"
	"github.com/skrashevich/telegram-mock-ai/internal/seed"
	"github.com/skrashevich/telegram-mock-ai/internal/state"
	"github.com/skrashevich/telegram-mock-ai/internal/updates"
	"github.com/skrashevich/telegram-mock-ai/internal/webhook"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	setupLogging(cfg.Log)

	slog.Info("starting telegram-mock-ai",
		"port", cfg.Server.Port,
		"llm_enabled", cfg.LLM.Enabled,
		"proactive_enabled", cfg.Proactive.Enabled,
	)

	// Initialize core components
	store := state.NewStore()
	registry := bot.NewRegistry(true) // auto-register bots
	webhookClient := webhook.NewClient(cfg.Webhook.Timeout, cfg.Webhook.MaxRetries, cfg.Webhook.RetryDelay)
	dispatcher := updates.NewDispatcher(registry, webhookClient)

	// Initialize LLM client
	var llmClient *llm.Client
	if cfg.LLM.Enabled {
		llmClient = llm.NewClient(
			cfg.LLM.BaseURL,
			cfg.LLM.APIKey,
			cfg.LLM.Model,
			cfg.LLM.APIType,
			cfg.LLM.Temperature,
			cfg.LLM.MaxTokens,
			cfg.LLM.Timeout,
		)
		slog.Info("LLM client initialized", "base_url", cfg.LLM.BaseURL, "model", cfg.LLM.Model)
	}

	// Seed initial data
	seedData(cfg, store, registry, llmClient)

	// Create Bot API server
	botServer := api.NewServer(cfg, store, registry, dispatcher, llmClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start proactive engine
	if cfg.Proactive.Enabled && llmClient != nil {
		engine := proactive.NewEngine(cfg.Proactive, store, llmClient, dispatcher, registry, cfg.LLM.Timeout)
		go engine.Run(ctx)
	}

	// Start Bot API HTTP server
	botHTTP := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      botServer.Handler(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		slog.Info("Bot API server listening", "addr", botHTTP.Addr)
		if err := botHTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Bot API server error", "err", err)
			os.Exit(1)
		}
	}()

	// Start Admin API server
	var adminHTTP *http.Server
	if cfg.Admin.Enabled {
		adminServer := api.NewAdminServer(store, registry, dispatcher, llmClient, cfg.Seed.Generate)
		adminHTTP = &http.Server{
			Addr:    fmt.Sprintf("%s:%d", cfg.Admin.Host, cfg.Admin.Port),
			Handler: adminServer.Handler(),
		}
		go func() {
			slog.Info("Admin API server listening", "addr", adminHTTP.Addr)
			if err := adminHTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("Admin API server error", "err", err)
			}
		}()
	}

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	botHTTP.Shutdown(shutdownCtx)
	if adminHTTP != nil {
		adminHTTP.Shutdown(shutdownCtx)
	}

	slog.Info("shutdown complete")
}

func setupLogging(cfg config.LogConfig) {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}

func seedData(cfg *config.Config, store *state.Store, registry *bot.Registry, llmClient *llm.Client) {
	// Seed users
	for _, su := range cfg.Seed.Users {
		store.CreateUser(models.User{
			ID:        su.ID,
			FirstName: su.FirstName,
			LastName:  su.LastName,
			Username:  su.Username,
		})
		slog.Debug("seeded user", "id", su.ID, "name", su.FirstName)
	}

	// Seed bots
	for _, sb := range cfg.Seed.Bots {
		b := registry.Register(sb.Token, sb.Username, sb.FirstName)
		// Also add bot as a user in the store so AddChatMember can find it
		store.CreateUser(b.User)
		slog.Debug("seeded bot", "token", sb.Token[:min(10, len(sb.Token))]+"...", "username", sb.Username, "id", b.User.ID)
	}

	// Seed chats
	for _, sc := range cfg.Seed.Chats {
		store.CreateChat(models.Chat{
			ID:    sc.ID,
			Type:  sc.Type,
			Title: sc.Title,
		})
		for _, uid := range sc.Members {
			store.AddChatMember(sc.ID, uid, "member")
		}
		// Also add all bots to each seeded chat
		for _, b := range registry.List() {
			store.AddChatMember(sc.ID, b.User.ID, "member")
		}
		slog.Debug("seeded chat", "id", sc.ID, "title", sc.Title, "members", len(sc.Members))
	}

	// LLM-powered seed generation
	if cfg.Seed.Generate != nil && cfg.Seed.Generate.Enabled && llmClient != nil {
		gen := seed.NewGenerator(llmClient, store, registry, cfg.Seed.Generate.MaxRetries)
		timeout := cfg.LLM.Timeout * 3
		if timeout == 0 {
			timeout = 90 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		result, err := gen.Generate(ctx, seed.GenerateRequest{
			UsersCount:    cfg.Seed.Generate.UsersCount,
			GroupsCount:   cfg.Seed.Generate.GroupsCount,
			ChannelsCount: cfg.Seed.Generate.ChannelsCount,
			Locale:        cfg.Seed.Generate.Locale,
		})
		if err != nil {
			slog.Error("LLM seed generation failed, continuing with manual seed data", "err", err)
		} else {
			slog.Info("LLM seed generation complete",
				"users", len(result.Users),
				"chats", len(result.Chats),
			)
		}
	}
}
