package model

// CEFR is a Common European Framework of Reference for Languages level, from A1
// (beginner) to C2 (mastery). It grades communicative functions in the
// language-agnostic curriculum spine, independent of any single language's own
// proficiency scale (e.g. JLPT for Japanese).
type CEFR string

const (
	A1 CEFR = "A1"
	A2 CEFR = "A2"
	B1 CEFR = "B1"
	B2 CEFR = "B2"
	C1 CEFR = "C1"
	C2 CEFR = "C2"
)

// CEFRLevels lists every level ordered from beginner to mastery.
var CEFRLevels = []CEFR{A1, A2, B1, B2, C1, C2}

// Valid reports whether c is a recognized CEFR level.
func (c CEFR) Valid() bool {
	switch c {
	case A1, A2, B1, B2, C1, C2:
		return true
	default:
		return false
	}
}
