package model

// JLPT is a Japanese-Language Proficiency Test level, from N5 (easiest) to N1
// (hardest). Levels are used as a motivational progress indicator.
type JLPT string

const (
	N5 JLPT = "N5"
	N4 JLPT = "N4"
	N3 JLPT = "N3"
	N2 JLPT = "N2"
	N1 JLPT = "N1"
)

// JLPTLevels lists every level ordered from easiest to hardest.
var JLPTLevels = []JLPT{N5, N4, N3, N2, N1}

// Valid reports whether j is a recognized JLPT level.
func (j JLPT) Valid() bool {
	switch j {
	case N5, N4, N3, N2, N1:
		return true
	default:
		return false
	}
}
