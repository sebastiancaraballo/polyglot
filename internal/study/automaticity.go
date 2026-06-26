package study

import (
	"time"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// MasteryStreak is the run of correct answers in a row that marks a kana
// mastered. Mastery depends on accuracy only: answering correctly several times
// running shows the reading is learned. Response time is recorded as a stat (see
// BestMs) but does not gate the streak.
const MasteryStreak = 3

// GradeKana folds one answer into a kana's progress and returns the updated
// value. elapsed is the time the learner took to answer; it is recorded as the
// kana's best time but does not affect the mastery streak. It is a pure function.
func GradeKana(p model.KanaProgress, correct bool, elapsed time.Duration) model.KanaProgress {
	p.Attempts++

	if correct {
		if ms := int(elapsed / time.Millisecond); ms > 0 && (p.BestMs == 0 || ms < p.BestMs) {
			p.BestMs = ms
		}
		p.Streak++
		if p.Streak >= MasteryStreak {
			p.Mastered = true
		}
	} else {
		p.Streak = 0
	}
	return p
}
