package model

// PatternProgress tracks a learner's progress drilling one slot of one
// grammar pattern to a correctness-based mastery streak — the same
// automaticity idiom used for KanaProgress. Slots are tracked independently
// so words-before-sentences sequencing can tell which slot still needs
// practice.
type PatternProgress struct {
	PatternID string
	Slot      string
	Streak    int
	Attempts  int
	Mastered  bool
}
