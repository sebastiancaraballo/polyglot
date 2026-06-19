package model

import "time"

// DefaultEase is the starting ease factor for a new card (SM-2 style).
const DefaultEase = 2.5

// CardState captures the spaced-repetition scheduling state of a single card,
// scoped to a profile. It is updated after every review.
type CardState struct {
	CardID         string
	Interval       int       // days until the next review
	Ease           float64   // ease factor; higher means longer intervals
	Reps           int       // consecutive successful reviews
	Lapses         int       // number of times the card was forgotten
	DueAt          time.Time // when the card is next due for review
	LastReviewedAt time.Time
}
