package model

// Slot is a named blank in a grammar pattern's frame, filled by choosing among
// candidate vocabulary cards the learner already knows. A slot never
// introduces new vocabulary through the grammar drill itself ("words before
// sentences") — every candidate must be an existing Card.ID.
type Slot struct {
	Name    string   // placeholder name referenced in the frame, e.g. "X", "N"
	CardIDs []string // candidate vocab Card.ID values this slot may be filled with
	Default string   // Card.ID held fixed when this slot is not the round's
	// variable slot (Cognitive Load Theory: only one slot varies per round).
}

// Pattern is a fixed sentence frame with one or more slots, used for
// structured-input / minimal-substitution practice (Processing Instruction,
// VanPatten). Frame holds each slot's name in "{Name}" placeholders, e.g.
// "{X}は{N}です".
type Pattern struct {
	ID    string
	Title string // short label, e.g. "X wa N desu"
	JLPT  JLPT
	Frame string
	Slots []Slot
	Notes string
}
