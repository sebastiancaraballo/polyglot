// Package study holds shared study-mode logic used by multiple screens:
// multiple-choice option generation and study-streak bookkeeping.
package study

import "math/rand"

// Options returns up to n answer options containing correct plus distinct
// distractors drawn from pool, along with the index of the correct option. The
// ordering is randomized using rng so the result is deterministic in tests.
func Options(rng *rand.Rand, correct string, pool []string, n int) ([]string, int) {
	seen := map[string]bool{correct: true}
	distractors := make([]string, 0, len(pool))
	for _, p := range pool {
		if !seen[p] {
			seen[p] = true
			distractors = append(distractors, p)
		}
	}
	rng.Shuffle(len(distractors), func(i, j int) {
		distractors[i], distractors[j] = distractors[j], distractors[i]
	})

	want := n - 1
	if want > len(distractors) {
		want = len(distractors)
	}
	options := make([]string, 0, want+1)
	options = append(options, correct)
	options = append(options, distractors[:want]...)
	rng.Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})

	correctIdx := 0
	for i, o := range options {
		if o == correct {
			correctIdx = i
			break
		}
	}
	return options, correctIdx
}
