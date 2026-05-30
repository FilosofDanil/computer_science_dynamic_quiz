// Command migrate is a one-shot seeder that imports the bundled curated topics
// from data/topics.json into PostgreSQL. Topics are grouped by their Category
// field; each category becomes one global (owner_chat IS NULL) test.
//
// The command is idempotent: a category whose test already exists is skipped,
// so it is safe to run repeatedly.
//
// Usage:
//
//	DATABASE_URL=postgres://... go run ./cmd/migrate
package main

import (
	"context"
	"log/slog"
	"os"

	"basics/internal/config"
	"basics/internal/quiz"
	"basics/internal/storage"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	_ = config.LoadDotEnv(".env")
	dbURL := config.MustDatabaseURL()

	topicsPath := "data/topics.json"
	if p := os.Getenv("TOPICS_PATH"); p != "" {
		topicsPath = p
	}

	jsonStore, err := storage.NewJSONTopicStore(topicsPath)
	if err != nil {
		slog.Error("failed to load source topics", "path", topicsPath, "err", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pg, err := storage.NewPGTestStore(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect to postgres", "err", err)
		os.Exit(1)
	}
	defer pg.Close()

	// Group topics by category, preserving first-seen order for determinism.
	grouped := map[string][]quiz.Topic{}
	var order []string
	for _, t := range jsonStore.All() {
		cat := t.Category
		if cat == "" {
			cat = "Uncategorized"
		}
		if _, ok := grouped[cat]; !ok {
			order = append(order, cat)
		}
		grouped[cat] = append(grouped[cat], t)
	}

	var created, skipped int
	for _, cat := range order {
		exists, err := pg.GlobalTitleExists(ctx, cat)
		if err != nil {
			slog.Error("failed to check existing test", "title", cat, "err", err)
			os.Exit(1)
		}
		if exists {
			slog.Info("skipping existing test", "title", cat, "questions", len(grouped[cat]))
			skipped++
			continue
		}
		id, err := pg.Create(storage.Test{
			OwnerChat: nil, // global / curated
			Title:     cat,
			Questions: grouped[cat],
		})
		if err != nil {
			slog.Error("failed to create test", "title", cat, "err", err)
			os.Exit(1)
		}
		slog.Info("created test", "id", id, "title", cat, "questions", len(grouped[cat]))
		created++
	}

	slog.Info("migration complete", "created", created, "skipped", skipped, "categories", len(order))
}
