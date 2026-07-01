package model

// BeatKind distinguishes what kind of moment a Beat is. Narration sets a scene
// or advances the plot with no specific speaker; dialogue is a line spoken by
// a character; practice pauses the story for one diegetic check that reuses
// an existing trainer's grading logic.
type BeatKind string

const (
	Narration BeatKind = "narration"
	Dialogue  BeatKind = "dialogue"
	Practice  BeatKind = "practice"
)

// Valid reports whether k is a recognized beat kind.
func (k BeatKind) Valid() bool {
	switch k {
	case Narration, Dialogue, Practice:
		return true
	default:
		return false
	}
}

// PracticeKind identifies which existing trainer's pool a practice beat draws
// its question from.
type PracticeKind string

const (
	PracticeVocab PracticeKind = "vocab"
	PracticeKana  PracticeKind = "kana"
)

// Valid reports whether k is a recognized practice kind.
func (k PracticeKind) Valid() bool {
	return k == PracticeVocab || k == PracticeKana
}

// Beat is a single moment in a chapter. Only the fields relevant to Kind are
// populated; the rest stay at their zero value, the same "optional, zero
// means unset" idiom already used by Card.Freq and Pattern.Notes.
type Beat struct {
	Kind BeatKind

	Speaker string // dialogue only: the character's name
	Place   string // optional: a real-world place this beat evokes

	Source string // narration/dialogue: the line in the learner's source language (Spanish)
	JP     string // narration/dialogue: the line in Japanese
	Romaji string // narration/dialogue: optional romanized reading

	Practice PracticeKind // practice only
	// RefID resolves the practice question pool: a Lesson.ID when Practice is
	// PracticeVocab, or "hiragana"/"katakana" (== string(KanaType)) when
	// Practice is PracticeKana.
	RefID string
}

// Chapter is an ordered sequence of beats: Katsudoo's communicative-activity
// unit.
type Chapter struct {
	ID    string
	Title string
	Beats []Beat
}
