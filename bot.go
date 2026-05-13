package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// ── Session ──────────────────────────────────────────────────────────────────

type stage int

const (
	stageCategory stage = iota
	stageOrder
	stageQuiz
	stageReveal
	stageDone
)

type session struct {
	stage      stage
	topics     []Topic
	index      int
	score      int
	options    []string
	correctIdx int
	lastMsgID  int
}

var (
	sessionsMu sync.Mutex
	sessions   = map[int64]*session{}
)

func getSession(chatID int64) *session {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	s, ok := sessions[chatID]
	if !ok {
		s = &session{}
		sessions[chatID] = s
	}
	return s
}

func resetSession(chatID int64) *session {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	s := &session{}
	sessions[chatID] = s
	return s
}

func deleteSession(chatID int64) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	delete(sessions, chatID)
}

// ── Bot entry point ───────────────────────────────────────────────────────────

func runBot() {
	_ = loadDotEnv(".env")
	token := mustToken()

	b, err := bot.New(token,
		bot.WithDefaultHandler(onDefault),
		bot.WithCallbackQueryDataHandler("cat:", bot.MatchTypePrefix, onCallbackCategory),
		bot.WithCallbackQueryDataHandler("order:", bot.MatchTypePrefix, onCallbackOrder),
		bot.WithCallbackQueryDataHandler("ans:", bot.MatchTypePrefix, onCallbackAnswer),
		bot.WithCallbackQueryDataHandler("next", bot.MatchTypeExact, onCallbackNext),
		bot.WithCallbackQueryDataHandler("again", bot.MatchTypeExact, onCallbackAgain),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create bot: %v\n", err)
		os.Exit(1)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeCommand, onStart)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/quit", bot.MatchTypeCommand, onQuit)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	fmt.Println("Bot is running. Press Ctrl+C to stop.")
	b.Start(ctx)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func chatIDFromUpdate(update *models.Update) int64 {
	if update.Message != nil {
		return update.Message.Chat.ID
	}
	if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.Message.Chat.ID
	}
	return 0
}

func progressBar(current, total int) string {
	const barLen = 10
	filled := 0
	if total > 0 {
		filled = (current * barLen) / total
	}
	return strings.Repeat("▰", filled) + strings.Repeat("▱", barLen-filled)
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func sendOrEdit(ctx context.Context, b *bot.Bot, chatID int64, s *session, text string, kb *models.InlineKeyboardMarkup) {
	if s.lastMsgID != 0 {
		_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   s.lastMsgID,
			Text:        text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: kb,
		})
		if err == nil {
			return
		}
		// If edit fails (message too old etc.), fall through to send a new one
	}
	msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: kb,
	})
	if err == nil && msg != nil {
		s.lastMsgID = msg.ID
	}
}

// ── Keyboards ─────────────────────────────────────────────────────────────────

func categoryKeyboard() *models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton
	for i := 0; i < len(categoryMenu); i += 2 {
		e1 := categoryMenu[i]
		btn1 := models.InlineKeyboardButton{
			Text:         fmt.Sprintf("%s (%d)", e1.label, countInCategory(e1.cat)),
			CallbackData: fmt.Sprintf("cat:%c", e1.key),
		}
		row := []models.InlineKeyboardButton{btn1}
		if i+1 < len(categoryMenu) {
			e2 := categoryMenu[i+1]
			btn2 := models.InlineKeyboardButton{
				Text:         fmt.Sprintf("%s (%d)", e2.label, countInCategory(e2.cat)),
				CallbackData: fmt.Sprintf("cat:%c", e2.key),
			}
			row = append(row, btn2)
		}
		rows = append(rows, row)
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func orderKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "In order", CallbackData: "order:s"},
				{Text: "Shuffle 🔀", CallbackData: "order:r"},
			},
		},
	}
}

func answerKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "A", CallbackData: "ans:0"},
				{Text: "B", CallbackData: "ans:1"},
			},
			{
				{Text: "C", CallbackData: "ans:2"},
				{Text: "D", CallbackData: "ans:3"},
			},
		},
	}
}

func nextKeyboard(last bool) *models.InlineKeyboardMarkup {
	label := "Next ▶"
	data := "next"
	if last {
		label = "See results 🏁"
		data = "next"
	}
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: label, CallbackData: data}},
		},
	}
}

func againKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "Play again 🔄", CallbackData: "again"}},
		},
	}
}

// ── Message builders ──────────────────────────────────────────────────────────

func categoryText() string {
	return "🎓 <b>CS Foundations Pyramid — Quiz</b>\n\nChoose a topic category to practice:"
}

func orderText(count int) string {
	return fmt.Sprintf("🎓 <b>CS Foundations Pyramid — Quiz</b>\n\n<b>%d topic(s)</b> selected.\n\nHow would you like to go through them?", count)
}

func questionText(s *session) string {
	topic := s.topics[s.index]
	total := len(s.topics)
	bar := progressBar(s.index, total)

	var sb strings.Builder
	fmt.Fprintf(&sb, "🎓 <b>CS Foundations Pyramid</b>  |  Question %d/%d\n", s.index+1, total)
	fmt.Fprintf(&sb, "%s\n\n", bar)
	fmt.Fprintf(&sb, "<i>%s</i>\n\n", escapeHTML(topic.Category))
	fmt.Fprintf(&sb, "<b>%s</b>\n\n", escapeHTML(topic.Question))

	labels := []string{"A", "B", "C", "D"}
	for i, name := range s.options {
		fmt.Fprintf(&sb, "<b>%s)</b>  %s\n\n", labels[i], escapeHTML(name))
	}
	return sb.String()
}

func revealText(s *session, chosen int) string {
	topic := s.topics[s.index]
	total := len(s.topics)
	bar := progressBar(s.index+1, total)

	var sb strings.Builder
	fmt.Fprintf(&sb, "🎓 <b>CS Foundations Pyramid</b>  |  Question %d/%d\n", s.index+1, total)
	fmt.Fprintf(&sb, "%s\n\n", bar)

	if chosen == s.correctIdx {
		sb.WriteString("✅ <b>Correct!</b>\n\n")
	} else {
		labels := []string{"A", "B", "C", "D"}
		fmt.Fprintf(&sb, "❌ <b>Not quite.</b>  The answer was <b>%s) %s</b>\n\n",
			labels[s.correctIdx], escapeHTML(topic.Name))
	}

	fmt.Fprintf(&sb, "──────────────────────────\n")
	fmt.Fprintf(&sb, "<b>%s</b>\n\n", escapeHTML(topic.Name))
	fmt.Fprintf(&sb, "%s\n\n", escapeHTML(topic.Overview))
	fmt.Fprintf(&sb, "<i>%s</i>\n\n", escapeHTML(topic.Explanation))
	fmt.Fprintf(&sb, "──────────────────────────\n")
	fmt.Fprintf(&sb, "Score so far: <b>%d / %d</b>", s.score, s.index+1)
	return sb.String()
}

func finalText(score, total int) string {
	pct := 0
	if total > 0 {
		pct = (score * 100) / total
	}

	var msg string
	switch {
	case pct == 100:
		msg = "Perfect score! You've fully internalised the pyramid."
	case pct >= 80:
		msg = "Excellent! The fundamentals are solid."
	case pct >= 60:
		msg = "Good progress. A second pass will lock it in."
	case pct >= 40:
		msg = "Keep going — the lower layers are worth revisiting."
	default:
		msg = "The journey starts here. Run it again and watch the score climb!"
	}

	return fmt.Sprintf(
		"🏁 <b>Quiz complete!</b>\n\nYour score: <b>%d / %d</b> (%d%%)\n\n%s",
		score, total, pct, escapeHTML(msg),
	)
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func onDefault(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	s := getSession(chatID)
	text := categoryText()
	kb := categoryKeyboard()
	sendOrEdit(ctx, b, chatID, s, text, kb)
}

func onStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	s := resetSession(chatID)
	s.stage = stageCategory
	sendOrEdit(ctx, b, chatID, s, categoryText(), categoryKeyboard())
}

func onQuit(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	deleteSession(chatID)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      "👋 Session ended. Send /start whenever you want to play again!",
		ParseMode: models.ParseModeHTML,
	})
}

func onCallbackCategory(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	data := update.CallbackQuery.Data // "cat:<key>"
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	if len(data) < 5 {
		return
	}
	key := data[4] // byte after "cat:"

	var matched *categoryEntry
	for i := range categoryMenu {
		if categoryMenu[i].key == key {
			matched = &categoryMenu[i]
			break
		}
	}
	if matched == nil {
		return
	}

	s := getSession(chatID)
	s.topics = filterTopics(matched.cat)
	s.stage = stageOrder

	sendOrEdit(ctx, b, chatID, s, orderText(len(s.topics)), orderKeyboard())
}

func onCallbackOrder(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	data := update.CallbackQuery.Data // "order:s" or "order:r"
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	s := getSession(chatID)
	if len(data) < 7 {
		return
	}
	if data[6] == 'r' {
		rand.Shuffle(len(s.topics), func(i, j int) { s.topics[i], s.topics[j] = s.topics[j], s.topics[i] })
	}

	s.index = 0
	s.score = 0
	s.stage = stageQuiz
	s.options, s.correctIdx = buildOptions(s.topics[0])

	sendOrEdit(ctx, b, chatID, s, questionText(s), answerKeyboard())
}

func onCallbackAnswer(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	data := update.CallbackQuery.Data // "ans:0".."ans:3"
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	s := getSession(chatID)
	if s.stage != stageQuiz || len(s.topics) == 0 {
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
	sendOrEdit(ctx, b, chatID, s, revealText(s, chosen), nextKeyboard(last))
}

func onCallbackNext(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	s := getSession(chatID)
	s.index++

	if s.index >= len(s.topics) {
		s.stage = stageDone
		sendOrEdit(ctx, b, chatID, s, finalText(s.score, len(s.topics)), againKeyboard())
		return
	}

	s.stage = stageQuiz
	s.options, s.correctIdx = buildOptions(s.topics[s.index])
	sendOrEdit(ctx, b, chatID, s, questionText(s), answerKeyboard())
}

func onCallbackAgain(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})

	s := resetSession(chatID)
	s.stage = stageCategory
	sendOrEdit(ctx, b, chatID, s, categoryText(), categoryKeyboard())
}
