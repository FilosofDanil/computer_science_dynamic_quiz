package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strings"

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

// readKey blocks until the user presses a single key and returns it.
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

// wrapText wraps a string at the given column width without breaking words.
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

// categoryEntry defines one entry in the topic selection menu.
type categoryEntry struct {
	key   byte
	label string
	cat   string // empty string = all topics
}

var categoryMenu = []categoryEntry{
	{'0', "All Topics", ""},
	{'1', "CPU & Hardware", "CPU & Hardware"},
	{'2', "Memory & Concurrency", "Memory & Concurrency"},
	{'3', "Systems Programming", "Systems Programming"},
	{'4', "Runtimes & Compilers", "Runtimes & Compilers"},
	{'5', "Operating Systems", "Operating Systems"},
	{'6', "Containers", "Containers"},
	{'7', "Networking", "Networking"},
	{'8', "Security & Crypto", "Security & Crypto"},
	{'9', "Observability", "Observability"},
	{'a', "Linux", "Linux"},
	{'b', "Virtualization", "Virtualization"},
	{'c', "Algorithms & DS", "Algorithms & DS"},
	{'d', "LLMs & AI", "LLMs & AI"},
	{'e', "ML Fundamentals", "ML Fundamentals"},
	{'f', "System Design", "System Design"},
	{'g', "Java + Spring", "Java + Spring"},
	{'h', "JS / TS / React", "JS / TS / React"},
	{'i', "Golang", "Golang"},
}

func countInCategory(cat string) int {
	if cat == "" {
		return len(AllTopics)
	}
	n := 0
	for _, t := range AllTopics {
		if t.Category == cat {
			n++
		}
	}
	return n
}

func filterTopics(cat string) []Topic {
	if cat == "" {
		out := make([]Topic, len(AllTopics))
		copy(out, AllTopics)
		return out
	}
	var out []Topic
	for _, t := range AllTopics {
		if t.Category == cat {
			out = append(out, t)
		}
	}
	return out
}

func printCategoryMenu() {
	clearScreen()
	printBanner()
	fmt.Printf("  %s%sSelect a topic area%s\n\n", bold, yellow, reset)

	for _, e := range categoryMenu {
		count := countInCategory(e.cat)
		fmt.Printf("  %s[%s]%s  %-26s %s(%d topics)%s\n",
			bold+cyan, string(e.key), reset,
			e.label,
			dim, count, reset,
		)
	}
	fmt.Println()
	fmt.Printf("  %sPress a key (0–9):%s ", yellow, reset)
}

func selectCategory() []Topic {
	validKeys := make(map[byte]categoryEntry)
	for _, e := range categoryMenu {
		validKeys[e.key] = e
	}
	for {
		printCategoryMenu()
		key := readKey()
		if entry, ok := validKeys[key]; ok {
			fmt.Printf("%s\n", string(key))
			return filterTopics(entry.cat)
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

// buildOptions returns 4 shuffled topic Names (one correct, 3 distractors from
// the same category when possible) and the index of the correct Name.
func buildOptions(correct Topic) ([]string, int) {
	// Collect same-category candidates (excluding the correct topic)
	var sameCat []string
	for _, t := range AllTopics {
		if t.Category == correct.Category && t.Name != correct.Name {
			sameCat = append(sameCat, t.Name)
		}
	}
	rand.Shuffle(len(sameCat), func(i, j int) { sameCat[i], sameCat[j] = sameCat[j], sameCat[i] })

	wrong := make([]string, 0, 3)
	seen := map[string]bool{correct.Name: true}

	// Prefer same-category distractors
	for _, name := range sameCat {
		if len(wrong) >= 3 {
			break
		}
		if !seen[name] {
			seen[name] = true
			wrong = append(wrong, name)
		}
	}

	// Fall back to any topic if the category is too small
	for len(wrong) < 3 {
		idx := rand.Intn(len(AllTopics))
		name := AllTopics[idx].Name
		if !seen[name] {
			seen[name] = true
			wrong = append(wrong, name)
		}
	}

	pool := []string{correct.Name, wrong[0], wrong[1], wrong[2]}
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	correctIdx := -1
	for i, s := range pool {
		if s == correct.Name {
			correctIdx = i
			break
		}
	}
	return pool, correctIdx
}

func runQuiz(topics []Topic) int {
	score := 0
	total := len(topics)
	labels := []string{"A", "B", "C", "D"}

	for i, topic := range topics {
		clearScreen()
		printBanner()
		printProgressBar(i, total)

		// Category label (no topic name — that's the answer!)
		fmt.Printf("  %s%s%s\n", dim, topic.Category, reset)
		fmt.Println()

		// Scenario question
		fmt.Printf("  %s%s%s\n", bold+white, wrapText(topic.Question, 62), reset)
		fmt.Println()

		options, correctIdx := buildOptions(topic)

		// Display 4 topic-name options
		for j, name := range options {
			wrapped := wrapText(name, 58)
			fmt.Printf("  %s%s)%s  %s\n\n", bold+cyan, labels[j], reset, wrapped)
		}

		// Read answer (A/B/C/D)
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

		// ── Reveal ────────────────────────────────────────────────────────────

		if chosen == correctIdx {
			score++
			fmt.Printf("  %s%s✓  Correct!%s\n", bold, green, reset)
		} else {
			fmt.Printf("  %s%s✗  Not quite.%s  The answer was %s%s%s.\n",
				bold, red, reset, bold+yellow, topic.Name, reset)
		}

		// Divider
		fmt.Printf("\n  %s────────────────────────────────────────────────────────%s\n\n", dim, reset)

		// Topic name + one-sentence overview
		fmt.Printf("  %s%s%s\n", bold+yellow, topic.Name, reset)
		fmt.Println()
		fmt.Printf("  %s%s%s\n", white, wrapText(topic.Overview, 63), reset)

		// Beginner-friendly explanation
		fmt.Println()
		fmt.Printf("  %s%s%s\n", dim+white, wrapText(topic.Explanation, 63), reset)

		// Divider + running score
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

func main() {
	// Step 1: category selection
	topics := selectCategory()

	// Step 2: order / randomise
	clearScreen()
	printBanner()
	fmt.Printf("  %s%d topic(s) selected.%s\n", white, len(topics), reset)
	printOrderMenu()

	for {
		key := readKey()
		switch key | 0x20 {
		case 's':
			fmt.Printf("S\n")
		case 'r':
			fmt.Printf("R\n")
			rand.Shuffle(len(topics), func(i, j int) { topics[i], topics[j] = topics[j], topics[i] })
		default:
			fmt.Printf("\n  %sPlease press S or R.%s\n", red, reset)
			printOrderMenu()
			continue
		}
		break
	}

	// Step 3: quiz
	score := runQuiz(topics)

	// Step 4: final score
	printFinalScore(score, len(topics))
}
