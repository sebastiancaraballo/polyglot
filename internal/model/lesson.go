package model

// Card is a single vocabulary or phrase item: a term in the learner's source
// language paired with its Japanese form and romaji reading.
type Card struct {
	ID     string // stable identifier, "<lessonID>:<index>"
	Source string // term in the learner's source language (Spanish in v1)
	JP     string // Japanese form
	Romaji string // romanized reading
	Notes  string // optional usage notes
	JLPT   JLPT
}

// Lesson is an ordered collection of cards sharing a theme and JLPT level.
type Lesson struct {
	ID    string
	Title string
	JLPT  JLPT
	Cards []Card
}
