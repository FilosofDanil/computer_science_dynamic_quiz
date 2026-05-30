package bot

import (
	"context"
	"fmt"
	"math/rand"

	"basics/internal/apperrors"
	"basics/internal/quiz"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (b *Bot) onSettings(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      manageMenuText(),
		ParseMode: models.ParseModeHTML,
	})
}

func (b *Bot) onText(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID

	// While awaiting test JSON, route the incoming message into the create or
	// edit flow instead of treating it as an unknown command.
	s := b.sessions.Get(chatID)
	switch s.stage {
	case stageAwaitNewTest:
		b.handleDescription(ctx, tb, update, s, 0)
		return
	case stageAwaitEditTest:
		b.handleDescription(ctx, tb, update, s, s.editTestID)
		return
	}

	_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      unknownCommandText(),
		ParseMode: models.ParseModeHTML,
	})
}

func (b *Bot) onDefault(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      unknownCommandText(),
		ParseMode: models.ParseModeHTML,
	})
}

func (b *Bot) onStart(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	s := b.sessions.Reset(chatID)
	s.stage = stageCategory
	b.sendTestMenu(ctx, tb, chatID, s)
}

func (b *Bot) onHelp(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      helpText(),
		ParseMode: models.ParseModeHTML,
	})
}

func (b *Bot) onQuit(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	b.sessions.Delete(chatID)
	_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      "👋 Session ended. Send /start whenever you want to play again!",
		ParseMode: models.ParseModeHTML,
	})
}

func (b *Bot) onCallbackTest(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	data := update.CallbackQuery.Data // "test:<id>"
	tb.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	id, err := parseIDSuffix(data, "test:")
	if err != nil {
		handleErr(ctx, tb, chatID, fmt.Errorf("%w: %q", apperrors.ErrUnknownCategory, data), fallbackMsg)
		return
	}

	test, err := b.store.Get(id)
	if err != nil {
		handleErr(ctx, tb, chatID, fmt.Errorf("load test %d: %w", id, err), fallbackMsg)
		return
	}
	// Access control: a chat may only play global tests or tests it owns.
	if !test.IsGlobal() && !test.OwnedBy(chatID) {
		handleErr(ctx, tb, chatID, fmt.Errorf("%w: chat %d not allowed test %d", apperrors.ErrTestNotFound, chatID, id), fallbackMsg)
		return
	}
	if len(test.Questions) == 0 {
		handleErr(ctx, tb, chatID, fmt.Errorf("%w: %q", apperrors.ErrNoTopics, test.Title), fallbackMsg)
		return
	}

	s := b.sessions.Get(chatID)
	s.topics = test.Questions
	s.stage = stageOrder
	sendOrEdit(ctx, tb, chatID, s, orderText(len(s.topics)), orderKeyboard())
}

func (b *Bot) onCallbackOrder(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	data := update.CallbackQuery.Data // "order:s" or "order:r"
	tb.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	s := b.sessions.Get(chatID)
	if len(data) < 7 {
		return
	}
	if data[6] == 'r' {
		rand.Shuffle(len(s.topics), func(i, j int) { s.topics[i], s.topics[j] = s.topics[j], s.topics[i] })
	}

	s.index = 0
	s.score = 0
	s.stage = stageQuiz
	s.options, s.correctIdx = quiz.BuildOptions(s.topics, s.topics[0])
	sendOrEdit(ctx, tb, chatID, s, questionText(s), answerKeyboard())
}

func (b *Bot) onCallbackAnswer(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	data := update.CallbackQuery.Data // "ans:0".."ans:3"
	tb.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	s := b.sessions.Get(chatID)
	if s.stage != stageQuiz || len(s.topics) == 0 {
		handleErr(ctx, tb, chatID, fmt.Errorf("%w: got answer in stage %d", apperrors.ErrInvalidStage, s.stage), fallbackMsg)
		return
	}
	if len(data) < 5 {
		return
	}
	chosen := int(data[4] - '0')
	if chosen < 0 || chosen > 3 {
		return
	}

	if chosen == s.correctIdx {
		s.score++
	}

	s.stage = stageReveal
	last := s.index == len(s.topics)-1
	sendOrEdit(ctx, tb, chatID, s, revealText(s, chosen), nextKeyboard(last))
}

func (b *Bot) onCallbackNext(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	tb.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	s := b.sessions.Get(chatID)
	s.index++

	if s.index >= len(s.topics) {
		s.stage = stageDone
		sendOrEdit(ctx, tb, chatID, s, finalText(s.score, len(s.topics)), againKeyboard())
		return
	}

	s.stage = stageQuiz
	s.options, s.correctIdx = quiz.BuildOptions(s.topics, s.topics[s.index])
	sendOrEdit(ctx, tb, chatID, s, questionText(s), answerKeyboard())
}

func (b *Bot) onCallbackAgain(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	tb.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	s := b.sessions.Reset(chatID)
	s.stage = stageCategory
	b.sendTestMenu(ctx, tb, chatID, s)
}
