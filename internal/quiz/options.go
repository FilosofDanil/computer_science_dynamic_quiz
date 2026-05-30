package quiz

import (
	"fmt"
	"math/rand"
)

// placeholderDistractors are used only when the distractor pool is too small
// to supply three distinct wrong answers (e.g. a user-created test with very
// few questions). They keep the four-option keyboard well-formed.
var placeholderDistractors = []string{
	"None of the above",
	"All of the above",
	"Not applicable",
}

// BuildOptions returns 4 shuffled topic Names (one correct, 3 distractors
// from the same category when possible) and the index of the correct Name.
// allTopics is the pool used to source distractors; when sourcing quiz answers
// per test it is the test's own questions. The function never blocks: if the
// pool cannot supply three distinct distractors it pads with placeholders.
func BuildOptions(allTopics []Topic, correct Topic) (options []string, correctIdx int) {
	wrong := make([]string, 0, 3)
	seen := map[string]bool{correct.Name: true}

	// Prefer distractors from the same category as the correct answer.
	var sameCat []string
	var others []string
	for _, t := range allTopics {
		if t.Name == correct.Name {
			continue
		}
		if t.Category == correct.Category {
			sameCat = append(sameCat, t.Name)
		} else {
			others = append(others, t.Name)
		}
	}
	rand.Shuffle(len(sameCat), func(i, j int) { sameCat[i], sameCat[j] = sameCat[j], sameCat[i] })
	rand.Shuffle(len(others), func(i, j int) { others[i], others[j] = others[j], others[i] })

	for _, name := range append(sameCat, others...) {
		if len(wrong) >= 3 {
			break
		}
		if !seen[name] {
			seen[name] = true
			wrong = append(wrong, name)
		}
	}

	// Pad with placeholders when the pool was too small.
	for i := 0; len(wrong) < 3; i++ {
		cand := placeholderDistractors[i%len(placeholderDistractors)]
		for n := 1; seen[cand]; n++ {
			cand = fmt.Sprintf("%s (%d)", placeholderDistractors[i%len(placeholderDistractors)], n)
		}
		seen[cand] = true
		wrong = append(wrong, cand)
	}

	pool := []string{correct.Name, wrong[0], wrong[1], wrong[2]}
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	correctIdx = -1
	for i, s := range pool {
		if s == correct.Name {
			correctIdx = i
			break
		}
	}
	return pool, correctIdx
}
