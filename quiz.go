package main

import "math/rand"

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

// buildOptions returns 4 shuffled topic Names (one correct, 3 distractors from
// the same category when possible) and the index of the correct Name.
func buildOptions(correct Topic) ([]string, int) {
	var sameCat []string
	for _, t := range AllTopics {
		if t.Category == correct.Category && t.Name != correct.Name {
			sameCat = append(sameCat, t.Name)
		}
	}
	rand.Shuffle(len(sameCat), func(i, j int) { sameCat[i], sameCat[j] = sameCat[j], sameCat[i] })

	wrong := make([]string, 0, 3)
	seen := map[string]bool{correct.Name: true}

	for _, name := range sameCat {
		if len(wrong) >= 3 {
			break
		}
		if !seen[name] {
			seen[name] = true
			wrong = append(wrong, name)
		}
	}

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
