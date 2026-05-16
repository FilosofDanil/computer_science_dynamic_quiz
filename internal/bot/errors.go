package bot

import (
	"context"
	"log/slog"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const fallbackMsg = "Something went wrong. Please send /start to begin again."

// handleErr logs the error with context and sends a user-friendly message.
// It is a no-op when err is nil.
func handleErr(ctx context.Context, b *tgbot.Bot, chatID int64, err error, userMsg string) {
	if err == nil {
		return
	}
	slog.ErrorContext(ctx, "handler error", "chat_id", chatID, "err", err)
	if userMsg == "" {
		userMsg = fallbackMsg
	}
	_, _ = b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      userMsg,
		ParseMode: models.ParseModeHTML,
	})
}
