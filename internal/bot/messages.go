package bot

import (
	"fmt"
	"strings"
)

func unknownCommandText() string {
	return "I didn't understand that. Use /start to begin a quiz or /help for more info."
}

func helpText() string {
	return "🎓 <b>CS Foundations Pyramid — Quiz</b>\n\n" +
		"A flashcard quiz that drills you on core computer science concepts. " +
		"Curated tests ship with the bot, and you can create your own.\n\n" +
		"<b>Commands</b>\n" +
		"/start — start (or restart) a quiz session\n" +
		"/mytests — manage the tests you created (edit / delete)\n" +
		"/newtest — generate a new test with AI from a short description\n" +
		"/settings — show test-management help\n" +
		"/help  — show this message\n" +
		"/quit  — end your current session\n\n" +
		"<b>How it works</b>\n" +
		"1. Pick a test from the inline keyboard.\n" +
		"2. Choose <i>In order</i> or <i>Shuffle</i>.\n" +
		"3. Tap <b>A / B / C / D</b> to answer each question.\n" +
		"4. After each answer the correct answer, overview, and explanation are shown.\n" +
		"5. At the end your score is displayed with a <i>Play again</i> button.\n\n" +
		"Each chat has its own independent session — multiple users can play simultaneously."
}

func categoryText() string {
	return "🎓 <b>CS Foundations Pyramid — Quiz</b>\n\nChoose a test to practice (👤 marks tests you created):"
}

func noTestsText() string {
	return "There are no tests available yet.\n\nUse /newtest to create your first test."
}

func noOwnedTestsText() string {
	return "You haven't created any tests yet.\n\nUse /newtest to add one."
}

func myTestsText() string {
	return "📚 <b>Your tests</b>\n\nTap ✏️ to replace a test with new JSON, or 🗑 to delete it."
}

func manageMenuText() string {
	return "⚙️ <b>Manage your tests</b>\n\n" +
		"/mytests — list your tests with edit and delete buttons\n" +
		"/newtest — generate a new test with AI from a short description\n\n" +
		"Tests you create are private to this chat. Curated tests are shared and read-only."
}

// testPromptText asks the user to describe the test they want generated.
func testPromptText(editing bool) string {
	action := "Describe the test you'd like me to create"
	if editing {
		action = "Describe the new test to replace this one"
	}
	return fmt.Sprintf(
		"🤖 %s and I'll generate it with AI.\n\n"+
			"<b>Examples</b>\n"+
			"• <i>Create a test with 10 questions about Africa</i>\n"+
			"• <i>5 questions on the basics of Go concurrency</i>\n"+
			"• <i>A quiz about the water cycle for beginners</i>\n\n"+
			"Just send your description as a message. Send /quit to cancel.",
		action,
	)
}

func aiUnavailableText() string {
	return "🤖 AI test generation isn't configured on this bot.\n\n" +
		"An <code>ANTHROPIC_API_KEY</code> needs to be set. See INSTRUCTION.md for setup steps."
}

func generatingText() string {
	return "⏳ Generating your test with AI… this can take a few seconds."
}

func testSavedText(title string, n int, id int64, edited bool) string {
	verb := "created"
	if edited {
		verb = "updated"
	}
	return fmt.Sprintf(
		"✅ Test %s!\n\n<b>%s</b> — %d question(s) (id %d).\n\nSend /start to play it.",
		verb, escapeHTML(title), n, id,
	)
}

func deleteConfirmText(title string) string {
	return fmt.Sprintf("🗑 Delete <b>%s</b>? This cannot be undone.", escapeHTML(title))
}

func deletedText() string {
	return "🗑 Test deleted."
}

func deleteCancelledText() string {
	return "↩️ Deletion cancelled."
}

func orderText(count int) string {
	return fmt.Sprintf(
		"🎓 <b>CS Foundations Pyramid — Quiz</b>\n\n<b>%d topic(s)</b> selected.\n\nHow would you like to go through them?",
		count,
	)
}

func questionText(s *Session) string {
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

func revealText(s *Session, chosen int) string {
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

// progressBar builds a 10-character Unicode progress bar.
func progressBar(current, total int) string {
	const barLen = 10
	filled := 0
	if total > 0 {
		filled = (current * barLen) / total
	}
	return strings.Repeat("▰", filled) + strings.Repeat("▱", barLen-filled)
}

// escapeHTML escapes the minimal HTML entities needed for Telegram HTML mode.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
