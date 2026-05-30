package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"basics/internal/ai"
	"basics/internal/bot"
	"basics/internal/config"
	"basics/internal/storage"
)

func main() {
	// ── Logging ──────────────────────────────────────────────────────────────
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}
	var handler slog.Handler
	if os.Getenv("LOG_FORMAT") == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	}
	slog.SetDefault(slog.New(handler))

	// ── Config ───────────────────────────────────────────────────────────────
	_ = config.LoadDotEnv(".env")
	token := config.MustToken()
	dbURL := config.MustDatabaseURL()

	// ── Persistence ──────────────────────────────────────────────────────────
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	store, err := storage.NewPGTestStore(connectCtx, dbURL)
	connectCancel()
	if err != nil {
		slog.Error("failed to open test store", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	// ── AI test generation (optional) ─────────────────────────────────────────
	var gen bot.TestGenerator
	if key := config.ClaudeAPIKey(); key != "" {
		gen = ai.NewClient(key, config.ClaudeModel())
		slog.Info("AI test generation enabled")
	} else {
		slog.Warn("ANTHROPIC_API_KEY not set; /newtest AI generation disabled")
	}

	// ── Bot ──────────────────────────────────────────────────────────────────
	b := bot.New(store, gen)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := b.Run(ctx, token); err != nil {
		slog.Error("bot exited with error", "err", err)
		os.Exit(1)
	}
}
