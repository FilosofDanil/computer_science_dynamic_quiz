package bot

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"basics/internal/quiz"
	"basics/internal/storage"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Bot wires together the Telegram bot library, the topic store, and the
// session store. All handler methods live on this struct so they can reach
// dependencies via the receiver instead of package-level globals.
type Bot struct {
	store    storage.TopicStore
	sessions SessionStore
}

// New creates a Bot with the provided store. Sessions are managed with an
// in-memory store by default.
func New(store storage.TopicStore) *Bot {
	return &Bot{
		store:    store,
		sessions: NewInMemorySessionStore(),
	}
}

// Run registers all handlers and starts the long-polling loop.
// It blocks until ctx is cancelled (e.g. on SIGINT).
func (b *Bot) Run(ctx context.Context, token string) error {
	tb, err := tgbot.New(token,
		tgbot.WithDefaultHandler(b.onDefault),
		tgbot.WithCallbackQueryDataHandler("cat:", tgbot.MatchTypePrefix, b.onCallbackCategory),
		tgbot.WithCallbackQueryDataHandler("order:", tgbot.MatchTypePrefix, b.onCallbackOrder),
		tgbot.WithCallbackQueryDataHandler("ans:", tgbot.MatchTypePrefix, b.onCallbackAnswer),
		tgbot.WithCallbackQueryDataHandler("next", tgbot.MatchTypeExact, b.onCallbackNext),
		tgbot.WithCallbackQueryDataHandler("again", tgbot.MatchTypeExact, b.onCallbackAgain),
	)
	if err != nil {
		return fmt.Errorf("bot: create: %w", err)
	}

	tb.RegisterHandler(tgbot.HandlerTypeMessageText, "/start", tgbot.MatchTypeCommand, b.onStart)
	tb.RegisterHandler(tgbot.HandlerTypeMessageText, "/quit", tgbot.MatchTypeCommand, b.onQuit)

	slog.Info("bot started", "topics", len(b.store.All()))
	fmt.Fprintln(os.Stdout, "Bot is running. Press Ctrl+C to stop.")
	tb.Start(ctx)
	return nil
}

// categoryKeyboard is a Bot method so it can reach the store for topic counts.
func (b *Bot) categoryKeyboard() *models.InlineKeyboardMarkup {
	return categoryKeyboard(b.store)
}

// sendOrEdit tries to edit the last message in the session; on failure it
// sends a new message. Edit failures are logged at Debug level rather than
// silently swallowed.
func sendOrEdit(ctx context.Context, tb *tgbot.Bot, chatID int64, s *Session, text string, kb *models.InlineKeyboardMarkup) {
	if s.lastMsgID != 0 {
		_, err := tb.EditMessageText(ctx, &tgbot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   s.lastMsgID,
			Text:        text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: kb,
		})
		if err == nil {
			return
		}
		slog.DebugContext(ctx, "edit message failed, sending new", "chat_id", chatID, "err", err)
	}
	msg, err := tb.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: kb,
	})
	if err != nil {
		slog.ErrorContext(ctx, "send message failed", "chat_id", chatID, "err", err)
		return
	}
	if msg != nil {
		s.lastMsgID = msg.ID
	}
}

// ensure JSONTopicStore satisfies the interface the keyboards helper expects.
var _ interface{ All() []quiz.Topic } = (*storage.JSONTopicStore)(nil)
