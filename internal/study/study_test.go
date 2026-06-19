package study

import (
	"math/rand"
	"testing"
	"time"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

func TestOptionsContainsCorrect(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	pool := []string{"a", "b", "c", "d", "e", "f"}
	opts, idx := Options(rng, "a", pool, 4)

	if len(opts) != 4 {
		t.Fatalf("len(opts) = %d, want 4", len(opts))
	}
	if opts[idx] != "a" {
		t.Errorf("opts[%d] = %q, want correct %q", idx, opts[idx], "a")
	}
	seen := map[string]bool{}
	for _, o := range opts {
		if seen[o] {
			t.Errorf("duplicate option %q", o)
		}
		seen[o] = true
	}
}

func TestOptionsSmallPool(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	opts, idx := Options(rng, "x", []string{"x", "y"}, 4)
	if len(opts) != 2 {
		t.Fatalf("len(opts) = %d, want 2 (limited by pool)", len(opts))
	}
	if opts[idx] != "x" {
		t.Errorf("correct option missing")
	}
}

func TestUpdateStreak(t *testing.T) {
	today := time.Date(2026, 6, 19, 20, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)
	lastWeek := today.AddDate(0, 0, -7)

	tests := map[string]struct {
		in         model.Stats
		wantStreak int
		wantBest   int
	}{
		"first ever":      {model.Stats{}, 1, 1},
		"continued":       {model.Stats{Streak: 3, BestStreak: 3, LastStudiedAt: yesterday}, 4, 4},
		"gap resets":      {model.Stats{Streak: 9, BestStreak: 9, LastStudiedAt: lastWeek}, 1, 9},
		"best preserved":  {model.Stats{Streak: 2, BestStreak: 5, LastStudiedAt: yesterday}, 3, 5},
		"already studied": {model.Stats{Streak: 4, BestStreak: 6, LastStudiedAt: today}, 4, 6},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := UpdateStreak(tc.in, today)
			if got.Streak != tc.wantStreak {
				t.Errorf("streak = %d, want %d", got.Streak, tc.wantStreak)
			}
			if got.BestStreak != tc.wantBest {
				t.Errorf("best = %d, want %d", got.BestStreak, tc.wantBest)
			}
		})
	}
}
