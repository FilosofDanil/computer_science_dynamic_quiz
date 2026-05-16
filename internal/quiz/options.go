package quiz

import "math/rand"

// BuildOptions returns 4 shuffled topic Names (one correct, 3 distractors
// from the same category when possible) and the index of the correct Name.
// allTopics should be the full topic list used to source distractors.
func BuildOptions(allTopics []Topic, correct Topic) (options []string, correctIdx int) {
	var sameCat []string
	for _, t := range allTopics {
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
		idx := rand.Intn(len(allTopics))
		name := allTopics[idx].Name
		if !seen[name] {
			seen[name] = true
			wrong = append(wrong, name)
		}
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
