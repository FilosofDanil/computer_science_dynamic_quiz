package bot

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"basics/internal/storage"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// TestGenerator produces a JSON test document from a natural-language
// description. It is implemented by the Claude client in internal/ai.
type TestGenerator interface {
	GenerateTestJSON(ctx context.Context, description string) ([]byte, error)
}

// Bot wires together the Telegram bot library, the test store, the session
// store, and (optionally) an AI test generator. All handler methods live on
// this struct so they can reach dependencies via the receiver instead of
// package-level globals.
type Bot struct {
	store    storage.TestStore
	sessions SessionStore
	gen      TestGenerator // nil = AI generation disabled
}

// New creates a Bot with the provided store and an optional AI test generator
// (pass nil to disable AI generation). Sessions are managed with an in-memory
// store by default.
func New(store storage.TestStore, gen TestGenerator) *Bot {
	return &Bot{
		store:    store,
		sessions: NewInMemorySessionStore(),
		gen:      gen,
	}
}

// Run registers all handlers and starts the long-polling loop.
// It blocks until ctx is cancelled (e.g. on SIGINT).
func (b *Bot) Run(ctx context.Context, token string) error {
	tb, err := tgbot.New(token,
		tgbot.WithDefaultHandler(b.onDefault),
		tgbot.WithCallbackQueryDataHandler("test:", tgbot.MatchTypePrefix, b.onCallbackTest),
		tgbot.WithCallbackQueryDataHandler("order:", tgbot.MatchTypePrefix, b.onCallbackOrder),
		tgbot.WithCallbackQueryDataHandler("ans:", tgbot.MatchTypePrefix, b.onCallbackAnswer),
		tgbot.WithCallbackQueryDataHandler("next", tgbot.MatchTypeExact, b.onCallbackNext),
		tgbot.WithCallbackQueryDataHandler("again", tgbot.MatchTypeExact, b.onCallbackAgain),
		tgbot.WithCallbackQueryDataHandler("edit:", tgbot.MatchTypePrefix, b.onCallbackEdit),
		tgbot.WithCallbackQueryDataHandler("del:", tgbot.MatchTypePrefix, b.onCallbackDelete),
		tgbot.WithCallbackQueryDataHandler("delyes:", tgbot.MatchTypePrefix, b.onCallbackDeleteConfirm),
		tgbot.WithCallbackQueryDataHandler("delno", tgbot.MatchTypeExact, b.onCallbackDeleteCancel),
	)
	if err != nil {
		return fmt.Errorf("bot: create: %w", err)
	}

	tb.RegisterHandler(tgbot.HandlerTypeMessageText, "start", tgbot.MatchTypeCommand, b.onStart)
	tb.RegisterHandler(tgbot.HandlerTypeMessageText, "help", tgbot.MatchTypeCommand, b.onHelp)
	tb.RegisterHandler(tgbot.HandlerTypeMessageText, "quit", tgbot.MatchTypeCommand, b.onQuit)
	tb.RegisterHandler(tgbot.HandlerTypeMessageText, "settings", tgbot.MatchTypeCommand, b.onSettings)
	tb.RegisterHandler(tgbot.HandlerTypeMessageText, "mytests", tgbot.MatchTypeCommand, b.onMyTests)
	tb.RegisterHandler(tgbot.HandlerTypeMessageText, "newtest", tgbot.MatchTypeCommand, b.onNewTest)
	tb.RegisterHandler(tgbot.HandlerTypeMessageText, "", tgbot.MatchTypeContains, b.onText)

	slog.Info("bot started")
	fmt.Fprintln(os.Stdout, "Bot is running. Press Ctrl+C to stop.")
	tb.Start(ctx)
	return nil
}

// sendTestMenu fetches the tests available to chatID and renders the selection
// keyboard. It is shared by /start and the "play again" callback.
func (b *Bot) sendTestMenu(ctx context.Context, tb *tgbot.Bot, chatID int64, s *Session) {
	tests, err := b.store.ListAvailable(chatID)
	if err != nil {
		handleErr(ctx, tb, chatID, fmt.Errorf("list available tests: %w", err), fallbackMsg)
		return
	}
	if len(tests) == 0 {
		sendOrEdit(ctx, tb, chatID, s, noTestsText(), nil)
		return
	}
	sendOrEdit(ctx, tb, chatID, s, categoryText(), testKeyboard(tests))
}

// sendOrEdit tries to edit the last message in the session; on failure it
// sends a new message. Edit failures are logged at Debug level rather than
// silently swallowed.
func sendOrEdit(ctx context.Context, tb *tgbot.Bot, chatID int64, s *Session, text string, kb *models.InlineKeyboardMarkup) {
	if s.lastMsgID != 0 {
		editParams := &tgbot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: s.lastMsgID,
			Text:      text,
			ParseMode: models.ParseModeHTML,
		}
		// Only set ReplyMarkup when non-nil: passing a typed-nil pointer into
		// the `any` field marshals to JSON null, which Telegram rejects with
		// "object expected as reply markup".
		if kb != nil {
			editParams.ReplyMarkup = kb
		}
		if _, err := tb.EditMessageText(ctx, editParams); err == nil {
			return
		} else {
			slog.DebugContext(ctx, "edit message failed, sending new", "chat_id", chatID, "err", err)
		}
	}
	sendParams := &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	}
	if kb != nil {
		sendParams.ReplyMarkup = kb
	}
	msg, err := tb.SendMessage(ctx, sendParams)
	if err != nil {
		slog.ErrorContext(ctx, "send message failed", "chat_id", chatID, "err", err)
		return
	}
	if msg != nil {
		s.lastMsgID = msg.ID
	}
}
