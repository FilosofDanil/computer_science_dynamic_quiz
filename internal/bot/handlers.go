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

func (b *Bot) onDefault(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	s := b.sessions.Get(chatID)
	sendOrEdit(ctx, tb, chatID, s, categoryText(), b.categoryKeyboard())
}

func (b *Bot) onStart(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	s := b.sessions.Reset(chatID)
	s.stage = stageCategory
	sendOrEdit(ctx, tb, chatID, s, categoryText(), b.categoryKeyboard())
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

func (b *Bot) onCallbackCategory(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	data := update.CallbackQuery.Data // "cat:<key>"
	tb.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	if len(data) < 5 {
		return
	}
	key := data[4]

	var matched *quiz.CategoryEntry
	for i := range quiz.CategoryMenu {
		if quiz.CategoryMenu[i].Key == key {
			matched = &quiz.CategoryMenu[i]
			break
		}
	}
	if matched == nil {
		handleErr(ctx, tb, chatID, fmt.Errorf("%w: %c", apperrors.ErrUnknownCategory, key), fallbackMsg)
		return
	}

	topics := b.store.ByCategory(matched.Cat)
	if len(topics) == 0 {
		handleErr(ctx, tb, chatID, fmt.Errorf("%w: %q", apperrors.ErrNoTopics, matched.Cat), fallbackMsg)
		return
	}

	s := b.sessions.Get(chatID)
	s.topics = topics
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
	s.options, s.correctIdx = quiz.BuildOptions(b.store.All(), s.topics[0])
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
	s.options, s.correctIdx = quiz.BuildOptions(b.store.All(), s.topics[s.index])
	sendOrEdit(ctx, tb, chatID, s, questionText(s), answerKeyboard())
}

func (b *Bot) onCallbackAgain(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	tb.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	s := b.sessions.Reset(chatID)
	s.stage = stageCategory
	sendOrEdit(ctx, tb, chatID, s, categoryText(), b.categoryKeyboard())
}
