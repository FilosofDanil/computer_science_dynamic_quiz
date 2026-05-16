package bot

import (
	"fmt"
	"strings"
)

func categoryText() string {
	return "🎓 <b>CS Foundations Pyramid — Quiz</b>\n\nChoose a topic category to practice:"
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
