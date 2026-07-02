package study

import (
	"strings"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// CardKnown is the "words before sentences" gating signal: a vocab card counts
// as known once it has survived at least one spaced-repetition review. This is
// deliberately the simplest viable signal; a heavier per-word mastery concept
// is a possible future enhancement, not Rikai's job.
func CardKnown(state model.CardState) bool { return state.Reps > 0 }

// PatternReady reports whether every slot of the pattern has at least one
// filler the learner already knows, so the pattern can be drilled without
// ever introducing new vocabulary through the grammar drill itself.
func PatternReady(p model.Pattern, known map[string]bool) bool {
	for _, slot := range p.Slots {
		if !slotReady(slot, known) {
			return false
		}
	}
	return true
}

func slotReady(slot model.Slot, known map[string]bool) bool {
	for _, id := range slot.CardIDs {
		if known[id] {
			return true
		}
	}
	return false
}

// VariableSlotIndex returns which slot index varies on drill round n
// (0-based), cycling through the pattern's slots one at a time. Cognitive Load
// Theory: change only one variable per round. This is deliberately a plain
// round-robin, not adaptive selection — prioritizing the least-mastered slot
// is left as a possible future adaptive enhancement.
func VariableSlotIndex(slotCount, round int) int {
	if slotCount <= 0 {
		return 0
	}
	return round % slotCount
}

// RenderFrame substitutes each "{name}" placeholder in frame with fill[name].
func RenderFrame(frame string, fill map[string]string) string {
	var b strings.Builder
	i := 0
	for i < len(frame) {
		if frame[i] == '{' {
			if end := strings.IndexByte(frame[i:], '}'); end >= 0 {
				name := frame[i+1 : i+end]
				if v, ok := fill[name]; ok {
					b.WriteString(v)
					i += end + 1
					continue
				}
			}
		}
		b.WriteByte(frame[i])
		i++
	}
	return b.String()
}

// GradePatternSlot folds one substitution-drill answer into a pattern slot's
// progress. Mastery is correctness-only (a streak of MasteryStreak in a row),
// matching the precedent set for kana: Processing Instruction motivates
// meaningful-input practice, not speed, so no speed requirement is introduced
// here either.
func GradePatternSlot(p model.PatternProgress, correct bool) model.PatternProgress {
	p.Attempts++
	if correct {
		p.Streak++
		if p.Streak >= MasteryStreak {
			p.Mastered = true
		}
	} else {
		p.Streak = 0
	}
	return p
}
