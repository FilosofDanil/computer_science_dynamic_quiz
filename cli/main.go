// DEPRECATED: kept for reference; the Telegram bot is the active interface.
// Build and run with: go run ./cli
package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"basics/internal/quiz"
	"basics/internal/storage"
	"golang.org/x/term"
)

// ANSI colour helpers
const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	green   = "\033[32m"
	red     = "\033[31m"
	yellow  = "\033[33m"
	cyan    = "\033[36m"
	white   = "\033[97m"
	magenta = "\033[35m"
)

func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}

func readKey() byte {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		buf := make([]byte, 1)
		_, _ = os.Stdin.Read(buf)
		return buf[0]
	}
	defer term.Restore(fd, oldState)
	buf := make([]byte, 1)
	_, _ = os.Stdin.Read(buf)
	return buf[0]
}

func pressAnyKey() {
	fmt.Printf("\n%s%s  Press any key to continue...%s\n", dim, white, reset)
	readKey()
}

func wrapText(text string, width int) string {
	words := strings.Fields(text)
	var lines []string
	line := ""
	for _, w := range words {
		if len(line)+len(w)+1 > width {
			lines = append(lines, line)
			line = w
		} else {
			if line == "" {
				line = w
			} else {
				line += " " + w
			}
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n  ")
}

func printBanner() {
	fmt.Println()
	fmt.Printf("%s%s╔══════════════════════════════════════════════════════════╗%s\n", bold, cyan, reset)
	fmt.Printf("%s%s║        CS FOUNDATIONS PYRAMID  —  Quiz Game             ║%s\n", bold, cyan, reset)
	fmt.Printf("%s%s╚══════════════════════════════════════════════════════════╝%s\n", bold, cyan, reset)
	fmt.Println()
}

func printCategoryMenu(topics []quiz.Topic) {
	clearScreen()
	printBanner()
	fmt.Printf("  %s%sSelect a topic area%s\n\n", bold, yellow, reset)

	for _, e := range quiz.CategoryMenu {
		count := quiz.CountInCategory(topics, e.Cat)
		fmt.Printf("  %s[%s]%s  %-26s %s(%d topics)%s\n",
			bold+cyan, string(e.Key), reset,
			e.Label,
			dim, count, reset,
		)
	}
	fmt.Println()
	fmt.Printf("  %sPress a key (0–9, a–i):%s ", yellow, reset)
}

func selectCategory(topics []quiz.Topic) []quiz.Topic {
	validKeys := make(map[byte]quiz.CategoryEntry)
	for _, e := range quiz.CategoryMenu {
		validKeys[e.Key] = e
	}
	for {
		printCategoryMenu(topics)
		key := readKey()
		if entry, ok := validKeys[key]; ok {
			fmt.Printf("%s\n", string(key))
			return quiz.FilterTopics(topics, entry.Cat)
		}
	}
}

func printOrderMenu() {
	fmt.Println()
	fmt.Printf("  %s[S]%s  Go in order   %s[R]%s  Randomise\n\n",
		bold+cyan, reset, bold+cyan, reset)
	fmt.Printf("  %sYour choice:%s ", yellow, reset)
}

func printProgressBar(current, total int) {
	filled := 0
	if total > 0 {
		filled = (current * 30) / total
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", 30-filled)
	pct := 0
	if total > 0 {
		pct = (current * 100) / total
	}
	fmt.Printf("  %s[%s]%s %d/%d (%d%%)\n\n", dim, bar, reset, current, total, pct)
}

func runQuiz(allTopics []quiz.Topic, topics []quiz.Topic) int {
	score := 0
	total := len(topics)
	labels := []string{"A", "B", "C", "D"}

	for i, topic := range topics {
		clearScreen()
		printBanner()
		printProgressBar(i, total)

		fmt.Printf("  %s%s%s\n", dim, topic.Category, reset)
		fmt.Println()

		fmt.Printf("  %s%s%s\n", bold+white, wrapText(topic.Question, 62), reset)
		fmt.Println()

		options, correctIdx := quiz.BuildOptions(allTopics, topic)

		for j, name := range options {
			wrapped := wrapText(name, 58)
			fmt.Printf("  %s%s)%s  %s\n\n", bold+cyan, labels[j], reset, wrapped)
		}

		var chosen int
		for {
			fmt.Printf("  %sYour answer (A/B/C/D):%s ", yellow, reset)
			key := readKey()
			fmt.Printf("%s\n\n", strings.ToUpper(string(key)))

			switch key | 0x20 {
			case 'a':
				chosen = 0
			case 'b':
				chosen = 1
			case 'c':
				chosen = 2
			case 'd':
				chosen = 3
			default:
				fmt.Printf("  %sPlease press A, B, C, or D.%s\n\n", red, reset)
				continue
			}
			break
		}

		if chosen == correctIdx {
			score++
			fmt.Printf("  %s%s✓  Correct!%s\n", bold, green, reset)
		} else {
			fmt.Printf("  %s%s✗  Not quite.%s  The answer was %s%s%s.\n",
				bold, red, reset, bold+yellow, topic.Name, reset)
		}

		fmt.Printf("\n  %s────────────────────────────────────────────────────────%s\n\n", dim, reset)
		fmt.Printf("  %s%s%s\n", bold+yellow, topic.Name, reset)
		fmt.Println()
		fmt.Printf("  %s%s%s\n", white, wrapText(topic.Overview, 63), reset)
		fmt.Println()
		fmt.Printf("  %s%s%s\n", dim+white, wrapText(topic.Explanation, 63), reset)
		fmt.Printf("\n  %s────────────────────────────────────────────────────────%s\n", dim, reset)
		fmt.Printf("\n  %sScore so far: %d / %d%s\n", magenta, score, i+1, reset)

		pressAnyKey()
	}

	return score
}

func printFinalScore(score, total int) {
	clearScreen()
	printBanner()

	pct := 0
	if total > 0 {
		pct = (score * 100) / total
	}

	fmt.Printf("  %sGame complete!%s\n\n", bold+cyan, reset)
	fmt.Printf("  Your score: %s%d / %d%s  (%d%%)\n\n", bold+yellow, score, total, reset, pct)

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

	fmt.Printf("  %s%s%s\n\n", white, msg, reset)
	fmt.Printf("  Run the program again any time to practice.\n\n")
}

func runCLI(store storage.TopicStore) {
	allTopics := store.All()

	selected := selectCategory(allTopics)

	clearScreen()
	printBanner()
	fmt.Printf("  %s%d topic(s) selected.%s\n", white, len(selected), reset)
	printOrderMenu()

	for {
		key := readKey()
		switch key | 0x20 {
		case 's':
			fmt.Printf("S\n")
		case 'r':
			fmt.Printf("R\n")
			rand.Shuffle(len(selected), func(i, j int) { selected[i], selected[j] = selected[j], selected[i] })
		default:
			fmt.Printf("\n  %sPlease press S or R.%s\n", red, reset)
			printOrderMenu()
			continue
		}
		break
	}

	score := runQuiz(allTopics, selected)
	printFinalScore(score, len(selected))
}

func main() {
	store, err := storage.NewJSONTopicStore("data/topics.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load topics: %v\n", err)
		os.Exit(1)
	}
	runCLI(store)
}
