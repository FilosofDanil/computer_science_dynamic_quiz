package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"basics/internal/apperrors"
	"basics/internal/quiz"
	"basics/internal/storage"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// ownedTest builds a user-owned storage.Test, taking the address of a local
// copy of owner so the pointer is stable.
func ownedTest(owner int64, title string, questions []quiz.Topic) storage.Test {
	o := owner
	return storage.Test{OwnerChat: &o, Title: title, Questions: questions}
}

// onMyTests lists the tests owned by the chat with edit/delete buttons.
func (b *Bot) onMyTests(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	tests, err := b.store.ListOwned(chatID)
	if err != nil {
		handleErr(ctx, tb, chatID, fmt.Errorf("list owned tests: %w", err), fallbackMsg)
		return
	}
	if len(tests) == 0 {
		_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:    chatID,
			Text:      noOwnedTestsText(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}
	_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        myTestsText(),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: ownedTestsKeyboard(tests),
	})
}

// onNewTest puts the session into "awaiting description" mode and prompts the
// user to describe the test they want Claude to generate.
func (b *Bot) onNewTest(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	if b.gen == nil {
		_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:    chatID,
			Text:      aiUnavailableText(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}
	s := b.sessions.Get(chatID)
	s.stage = stageAwaitNewTest
	s.editTestID = 0
	_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      testPromptText(false),
		ParseMode: models.ParseModeHTML,
	})
}

// onCallbackEdit starts an edit flow for a test the user owns.
func (b *Bot) onCallbackEdit(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	tb.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	if b.gen == nil {
		_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:    chatID,
			Text:      aiUnavailableText(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	id, err := parseIDSuffix(update.CallbackQuery.Data, "edit:")
	if err != nil {
		handleErr(ctx, tb, chatID, fmt.Errorf("parse edit id: %w", err), fallbackMsg)
		return
	}
	test, err := b.store.Get(id)
	if err != nil || !test.OwnedBy(chatID) {
		handleErr(ctx, tb, chatID, fmt.Errorf("%w: edit id %d by chat %d", apperrors.ErrTestNotFound, id, chatID), "You can only edit your own tests.")
		return
	}

	s := b.sessions.Get(chatID)
	s.stage = stageAwaitEditTest
	s.editTestID = id
	_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      testPromptText(true),
		ParseMode: models.ParseModeHTML,
	})
}

// onCallbackDelete asks the user to confirm deletion.
func (b *Bot) onCallbackDelete(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	tb.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	id, err := parseIDSuffix(update.CallbackQuery.Data, "del:")
	if err != nil {
		handleErr(ctx, tb, chatID, fmt.Errorf("parse delete id: %w", err), fallbackMsg)
		return
	}
	test, err := b.store.Get(id)
	if err != nil || !test.OwnedBy(chatID) {
		handleErr(ctx, tb, chatID, fmt.Errorf("%w: delete id %d by chat %d", apperrors.ErrTestNotFound, id, chatID), "You can only delete your own tests.")
		return
	}
	_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        deleteConfirmText(test.Title),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: deleteConfirmKeyboard(id),
	})
}

// onCallbackDeleteConfirm performs the (owner-scoped) deletion.
func (b *Bot) onCallbackDeleteConfirm(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	tb.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	id, err := parseIDSuffix(update.CallbackQuery.Data, "delyes:")
	if err != nil {
		handleErr(ctx, tb, chatID, fmt.Errorf("parse delete id: %w", err), fallbackMsg)
		return
	}
	if err := b.store.Delete(id, chatID); err != nil {
		handleErr(ctx, tb, chatID, fmt.Errorf("delete test %d: %w", id, err), "Could not delete that test. It may already be gone.")
		return
	}
	s := b.sessions.Get(chatID)
	sendOrEdit(ctx, tb, chatID, s, deletedText(), nil)
}

// onCallbackDeleteCancel dismisses the delete confirmation.
func (b *Bot) onCallbackDeleteCancel(ctx context.Context, tb *tgbot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	tb.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	s := b.sessions.Get(chatID)
	sendOrEdit(ctx, tb, chatID, s, deleteCancelledText(), nil)
}

// handleDescription takes the user's natural-language description, asks Claude
// to generate a test, validates the result, and either creates a new test
// (editID == 0) or replaces an existing owned test.
func (b *Bot) handleDescription(ctx context.Context, tb *tgbot.Bot, update *models.Update, s *Session, editID int64) {
	chatID := update.Message.Chat.ID

	if b.gen == nil {
		s.stage = stageCategory
		s.editTestID = 0
		_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:    chatID,
			Text:      aiUnavailableText(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	description := strings.TrimSpace(update.Message.Text)
	if description == "" {
		_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:    chatID,
			Text:      "Please describe the test in a text message, e.g. \"Create a test with 10 questions about Africa\". Or /quit to cancel.",
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	// Generation can take several seconds; let the user know.
	_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      generatingText(),
		ParseMode: models.ParseModeHTML,
	})

	raw, err := b.gen.GenerateTestJSON(ctx, description)
	if err != nil {
		// Stay in the await stage so the user can rephrase and try again.
		handleErr(ctx, tb, chatID, fmt.Errorf("generate test: %w", err),
			"AI generation failed. Please try again with a clearer description, or /quit to cancel.")
		return
	}

	title, questions, err := parseTestInput(raw)
	if err != nil {
		handleErr(ctx, tb, chatID, fmt.Errorf("validate generated test: %w", err),
			"The AI returned an unexpected result. Please try rephrasing your description, or /quit to cancel.")
		return
	}

	if editID == 0 {
		newID, err := b.store.Create(ownedTest(chatID, title, questions))
		if err != nil {
			handleErr(ctx, tb, chatID, fmt.Errorf("create test: %w", err), fallbackMsg)
			return
		}
		s.stage = stageCategory
		s.editTestID = 0
		_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:    chatID,
			Text:      testSavedText(title, len(questions), newID, false),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	t := ownedTest(chatID, title, questions)
	t.ID = editID
	if err := b.store.Update(t); err != nil {
		handleErr(ctx, tb, chatID, fmt.Errorf("update test %d: %w", editID, err),
			"Could not update that test. You can only edit your own tests.")
		return
	}
	s.stage = stageCategory
	s.editTestID = 0
	_, _ = tb.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      testSavedText(title, len(questions), editID, true),
		ParseMode: models.ParseModeHTML,
	})
}

// testInput is the JSON shape users submit to create or edit a test.
type testInput struct {
	Title     string       `json:"title"`
	Questions []quiz.Topic `json:"questions"`
}

// parseTestInput unmarshals and validates a user-submitted test document.
func parseTestInput(raw []byte) (title string, questions []quiz.Topic, err error) {
	var in testInput
	if e := json.Unmarshal(raw, &in); e != nil {
		return "", nil, fmt.Errorf("%w: not valid JSON (%v)", apperrors.ErrInvalidTest, e)
	}

	title = strings.TrimSpace(in.Title)
	if title == "" {
		return "", nil, fmt.Errorf("%w: \"title\" is required", apperrors.ErrInvalidTest)
	}
	if len(in.Questions) == 0 {
		return "", nil, fmt.Errorf("%w: \"questions\" must contain at least one question", apperrors.ErrInvalidTest)
	}

	out := make([]quiz.Topic, 0, len(in.Questions))
	for i, q := range in.Questions {
		q.Name = strings.TrimSpace(q.Name)
		q.Question = strings.TrimSpace(q.Question)
		q.Overview = strings.TrimSpace(q.Overview)
		q.Explanation = strings.TrimSpace(q.Explanation)
		if q.Name == "" || q.Question == "" || q.Overview == "" || q.Explanation == "" {
			return "", nil, fmt.Errorf("%w: question %d must have non-empty Name, Question, Overview, and Explanation", apperrors.ErrInvalidTest, i+1)
		}
		// Category drives in-quiz labelling and distractor grouping; default it
		// to the test title so every question shares one group.
		if strings.TrimSpace(q.Category) == "" {
			q.Category = title
		}
		out = append(out, q)
	}
	return title, out, nil
}

// parseIDSuffix extracts the numeric id from callback data of the form
// "<prefix><id>", e.g. "test:42".
func parseIDSuffix(data, prefix string) (int64, error) {
	if !strings.HasPrefix(data, prefix) {
		return 0, fmt.Errorf("data %q missing prefix %q", data, prefix)
	}
	return strconv.ParseInt(strings.TrimPrefix(data, prefix), 10, 64)
}
