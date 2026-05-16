package quiz

// CategoryEntry defines one entry in the topic selection menu.
type CategoryEntry struct {
	Key   byte
	Label string
	Cat   string // empty string = all topics
}

// CategoryMenu is the ordered list of selectable topic categories.
var CategoryMenu = []CategoryEntry{
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

// CountInCategory returns how many topics belong to the given category.
// An empty cat string counts all topics.
func CountInCategory(topics []Topic, cat string) int {
	if cat == "" {
		return len(topics)
	}
	n := 0
	for _, t := range topics {
		if t.Category == cat {
			n++
		}
	}
	return n
}

// FilterTopics returns a new slice containing only the topics matching cat.
// An empty cat string returns a copy of all topics.
func FilterTopics(topics []Topic, cat string) []Topic {
	if cat == "" {
		out := make([]Topic, len(topics))
		copy(out, topics)
		return out
	}
	var out []Topic
	for _, t := range topics {
		if t.Category == cat {
			out = append(out, t)
		}
	}
	return out
}
