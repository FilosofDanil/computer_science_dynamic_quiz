package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

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

	// ── Persistence ──────────────────────────────────────────────────────────
	topicsPath := "data/topics.json"
	if p := os.Getenv("TOPICS_PATH"); p != "" {
		topicsPath = p
	}
	store, err := storage.NewJSONTopicStore(topicsPath)
	if err != nil {
		slog.Error("failed to load topic store", "path", topicsPath, "err", err)
		os.Exit(1)
	}

	// ── Bot ──────────────────────────────────────────────────────────────────
	b := bot.New(store)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := b.Run(ctx, token); err != nil {
		slog.Error("bot exited with error", "err", err)
		os.Exit(1)
	}
}
