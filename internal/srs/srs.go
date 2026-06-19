package srs

import (
	"math"
	"time"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// Grade is the learner's self-assessment of a review, from Again (forgot) to Easy.
type Grade int

const (
	Again Grade = iota + 1 // forgot the card
	Hard                   // recalled with difficulty
	Good                   // recalled correctly
	Easy                   // recalled effortlessly
)

// Valid reports whether g is a recognized grade.
func (g Grade) Valid() bool {
	return g >= Again && g <= Easy
}

const (
	minEase   = 1.3
	easyBonus = 1.3
)

// NewCard returns the initial scheduling state for a card that has never been
// reviewed. Its zero DueAt makes it immediately due.
func NewCard(cardID string) model.CardState {
	return model.CardState{CardID: cardID, Ease: model.DefaultEase}
}

// IsDue reports whether the card is due for review at now.
func IsDue(state model.CardState, now time.Time) bool {
	return !state.DueAt.After(now)
}

// Review applies a grade to a card's state and returns the updated state,
// including the next interval (in days) and due date. It is a pure function:
// the result depends only on the inputs.
func Review(state model.CardState, grade Grade, now time.Time) model.CardState {
	s := state
	if s.Ease == 0 {
		s.Ease = model.DefaultEase
	}
	s.LastReviewedAt = now

	if grade == Again {
		s.Reps = 0
		s.Lapses++
		s.Ease = clampEase(s.Ease - 0.20)
		s.Interval = 0
		s.DueAt = now // review again in the same session
		return s
	}

	switch grade {
	case Hard:
		s.Ease = clampEase(s.Ease - 0.15)
	case Easy:
		s.Ease = clampEase(s.Ease + 0.15)
	}

	if s.Reps == 0 {
		s.Interval = firstInterval(grade)
	} else {
		s.Interval = grownInterval(s.Interval, s.Ease, grade)
	}

	s.Reps++
	s.DueAt = now.AddDate(0, 0, s.Interval)
	return s
}

// PreviewInterval returns the interval (in days) that Review would assign for the
// given grade, without mutating state. It is used to show the learner when each
// answer choice will schedule the next review.
func PreviewInterval(state model.CardState, grade Grade, now time.Time) int {
	return Review(state, grade, now).Interval
}

func firstInterval(grade Grade) int {
	switch grade {
	case Easy:
		return 4
	default: // Hard, Good
		return 1
	}
}

func grownInterval(prev int, ease float64, grade Grade) int {
	if prev < 1 {
		prev = 1
	}
	var factor float64
	switch grade {
	case Hard:
		factor = 1.2
	case Easy:
		factor = ease * easyBonus
	default: // Good
		factor = ease
	}
	next := int(math.Round(float64(prev) * factor))
	if next <= prev {
		next = prev + 1
	}
	return next
}

func clampEase(ease float64) float64 {
	if ease < minEase {
		return minEase
	}
	return ease
}
