package study

import "github.com/sebastiancaraballo/polyglot/internal/model"

// The Foundations decoding gate. Polyglot follows the Simple View of Reading:
// comprehension can only ride on top of decoding, so the learner must be able to
// read kana fluently before meeting Japanese words and sentences. For a learner
// whose native script is the Latin alphabet (Spanish L1), kana is entirely new
// decoding, so this gate stands at the very start of the course.
//
// The gate is measured over the *base gojūon* of each syllabary — the core set
// you must decode automatically. Dakuten, handakuten, and combination kana are
// still practiced and reviewed, but they are derived forms and do not hold the
// gate.

// Fluency summarizes how much of one syllabary's base set a learner has driven
// to automaticity.
type Fluency struct {
	Mastered int
	Total    int
}

// Fluent reports whether every base kana in the syllabary is mastered.
func (f Fluency) Fluent() bool { return f.Total > 0 && f.Mastered >= f.Total }

// kanaFluency computes fluency over the base gojūon of one syllabary.
func kanaFluency(items []model.KanaItem, progress map[string]model.KanaProgress, typ model.KanaType) Fluency {
	var f Fluency
	for _, it := range items {
		if it.Type != typ || it.Category != model.Base {
			continue
		}
		f.Total++
		if progress[it.Char].Mastered {
			f.Mastered++
		}
	}
	return f
}

// Gate is the Foundations decoding gate, derived from a learner's kana progress.
// Katakana practice unlocks once hiragana is fluent; reading Japanese words and
// sentences unlocks once both syllabaries are fluent.
type Gate struct {
	Hiragana Fluency
	Katakana Fluency
}

// NewGate computes the gate from the full kana set and a learner's progress.
func NewGate(items []model.KanaItem, progress map[string]model.KanaProgress) Gate {
	return Gate{
		Hiragana: kanaFluency(items, progress, model.Hiragana),
		Katakana: kanaFluency(items, progress, model.Katakana),
	}
}

// KatakanaUnlocked reports whether katakana practice is available yet.
func (g Gate) KatakanaUnlocked() bool { return g.Hiragana.Fluent() }

// ReadingUnlocked reports whether reading Japanese words and sentences is
// available yet: both syllabaries must be fluent.
func (g Gate) ReadingUnlocked() bool { return g.Hiragana.Fluent() && g.Katakana.Fluent() }
