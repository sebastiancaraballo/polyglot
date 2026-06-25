package study

import (
	"time"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// Automaticity thresholds. A kana counts as mastered once the learner answers it
// correctly and quickly several times in a row. "Quickly" matters: the goal is
// effortless decoding (automaticity), so an answer that is accurate but slow is
// not yet automatic and does not advance the run.
const (
	// FluentResponse is the per-kana speed threshold for automaticity: a correct
	// answer slower than this is accurate but still effortful.
	FluentResponse = 4 * time.Second
	// MasteryStreak is the run of correct, fast answers that marks a kana mastered.
	MasteryStreak = 3
)

// GradeKana folds one answer into a kana's progress and returns the updated
// value. elapsed is the time the learner took to answer; a non-positive elapsed
// (e.g. an untimed answer) is treated as not fast. It is a pure function.
func GradeKana(p model.KanaProgress, correct bool, elapsed time.Duration) model.KanaProgress {
	p.Attempts++

	fast := elapsed > 0 && elapsed <= FluentResponse
	if correct {
		if ms := int(elapsed / time.Millisecond); ms > 0 && (p.BestMs == 0 || ms < p.BestMs) {
			p.BestMs = ms
		}
	}

	if correct && fast {
		p.Streak++
		if p.Streak >= MasteryStreak {
			p.Mastered = true
		}
	} else {
		p.Streak = 0
	}
	return p
}
