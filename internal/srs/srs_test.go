package srs

import (
	"testing"
	"time"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

var now = time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)

func TestNewCardIsDue(t *testing.T) {
	card := NewCard("c1")
	if card.Ease != model.DefaultEase {
		t.Errorf("ease = %v, want %v", card.Ease, model.DefaultEase)
	}
	if !IsDue(card, now) {
		t.Error("a new card should be due immediately")
	}
}

func TestGradeValid(t *testing.T) {
	for _, g := range []Grade{Again, Hard, Good, Easy} {
		if !g.Valid() {
			t.Errorf("grade %d should be valid", g)
		}
	}
	if Grade(0).Valid() || Grade(5).Valid() {
		t.Error("out-of-range grades should be invalid")
	}
}

func TestReviewFirstSuccess(t *testing.T) {
	tests := map[Grade]int{Hard: 1, Good: 1, Easy: 4}
	for grade, wantInterval := range tests {
		got := Review(NewCard("c1"), grade, now)
		if got.Interval != wantInterval {
			t.Errorf("grade %d: interval = %d, want %d", grade, got.Interval, wantInterval)
		}
		if got.Reps != 1 {
			t.Errorf("grade %d: reps = %d, want 1", grade, got.Reps)
		}
		wantDue := now.AddDate(0, 0, wantInterval)
		if !got.DueAt.Equal(wantDue) {
			t.Errorf("grade %d: due = %v, want %v", grade, got.DueAt, wantDue)
		}
	}
}

func TestReviewAgainResets(t *testing.T) {
	// Build up some progress first.
	card := Review(NewCard("c1"), Good, now)
	card = Review(card, Good, now)
	if card.Reps == 0 {
		t.Fatal("expected reps to have grown")
	}

	got := Review(card, Again, now)
	if got.Reps != 0 {
		t.Errorf("reps = %d, want 0 after Again", got.Reps)
	}
	if got.Lapses != 1 {
		t.Errorf("lapses = %d, want 1", got.Lapses)
	}
	if got.Interval != 0 {
		t.Errorf("interval = %d, want 0", got.Interval)
	}
	if !IsDue(got, now) {
		t.Error("card should be due immediately after Again")
	}
	if got.Ease >= card.Ease {
		t.Errorf("ease should decrease after Again: got %v, was %v", got.Ease, card.Ease)
	}
}

func TestIntervalsGrow(t *testing.T) {
	card := NewCard("c1")
	prev := 0
	for i := 0; i < 5; i++ {
		card = Review(card, Good, now)
		if card.Interval <= prev {
			t.Fatalf("interval did not grow: step %d got %d (prev %d)", i, card.Interval, prev)
		}
		prev = card.Interval
	}
}

func TestEasyGrowsFasterThanGood(t *testing.T) {
	// Use an established card (interval of a few days) so that rounding at
	// single-day intervals does not mask the difference.
	base := Review(Review(NewCard("c1"), Good, now), Good, now) // interval 3, reps 2
	good := Review(base, Good, now)
	easy := Review(base, Easy, now)
	if easy.Interval <= good.Interval {
		t.Errorf("easy interval %d should exceed good interval %d", easy.Interval, good.Interval)
	}
}

func TestEaseNeverBelowMinimum(t *testing.T) {
	card := NewCard("c1")
	for i := 0; i < 20; i++ {
		card = Review(card, Again, now)
	}
	if card.Ease < minEase {
		t.Errorf("ease = %v, want >= %v", card.Ease, minEase)
	}
}

func TestPreviewMatchesReview(t *testing.T) {
	card := Review(NewCard("c1"), Good, now)
	for _, g := range []Grade{Again, Hard, Good, Easy} {
		want := Review(card, g, now).Interval
		if got := PreviewInterval(card, g, now); got != want {
			t.Errorf("grade %d: preview = %d, want %d", g, got, want)
		}
	}
}
