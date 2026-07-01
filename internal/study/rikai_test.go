package study

import (
	"testing"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

func TestCardKnown(t *testing.T) {
	tests := []struct {
		name  string
		state model.CardState
		want  bool
	}{
		{"never reviewed", model.CardState{Reps: 0}, false},
		{"reviewed once", model.CardState{Reps: 1}, true},
		{"reviewed many times", model.CardState{Reps: 5}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CardKnown(tt.state); got != tt.want {
				t.Errorf("CardKnown(%+v) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestPatternReady(t *testing.T) {
	pattern := model.Pattern{
		Slots: []model.Slot{
			{Name: "X", CardIDs: []string{"a:1", "a:2"}, Default: "a:1"},
			{Name: "N", CardIDs: []string{"b:1", "b:2"}, Default: "b:1"},
		},
	}

	tests := []struct {
		name  string
		known map[string]bool
		want  bool
	}{
		{"nothing known", map[string]bool{}, false},
		{"only one slot has a known filler", map[string]bool{"a:1": true}, false},
		{"every slot has a known filler", map[string]bool{"a:1": true, "b:2": true}, true},
		{"extra known cards not referenced by any slot are irrelevant", map[string]bool{"a:1": true, "b:2": true, "c:1": true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PatternReady(pattern, tt.known); got != tt.want {
				t.Errorf("PatternReady(...) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVariableSlotIndex(t *testing.T) {
	tests := []struct {
		name      string
		slotCount int
		round     int
		want      int
	}{
		{"round 0 of 2 slots", 2, 0, 0},
		{"round 1 of 2 slots", 2, 1, 1},
		{"round 2 wraps back to slot 0", 2, 2, 0},
		{"round 5 of 3 slots", 3, 5, 2},
		{"single slot always varies slot 0", 1, 7, 0},
		{"zero slots does not panic", 0, 3, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VariableSlotIndex(tt.slotCount, tt.round); got != tt.want {
				t.Errorf("VariableSlotIndex(%d, %d) = %d, want %d", tt.slotCount, tt.round, got, tt.want)
			}
		})
	}
}

func TestRenderFrame(t *testing.T) {
	tests := []struct {
		name  string
		frame string
		fill  map[string]string
		want  string
	}{
		{
			name:  "two placeholders",
			frame: "{X}は{N}です",
			fill:  map[string]string{"X": "わたし", "N": "がくせい"},
			want:  "わたしはがくせいです",
		},
		{
			name:  "no placeholders",
			frame: "こんにちは",
			fill:  map[string]string{},
			want:  "こんにちは",
		},
		{
			name:  "unfilled placeholder is left as-is",
			frame: "{X}は{N}です",
			fill:  map[string]string{"X": "わたし"},
			want:  "わたしは{N}です",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RenderFrame(tt.frame, tt.fill); got != tt.want {
				t.Errorf("RenderFrame(%q, %v) = %q, want %q", tt.frame, tt.fill, got, tt.want)
			}
		})
	}
}

func TestGradePatternSlotMasteryRequiresAccurateRun(t *testing.T) {
	var p model.PatternProgress
	for i := 1; i <= MasteryStreak; i++ {
		p = GradePatternSlot(p, true)
		wantMastered := i >= MasteryStreak
		if p.Mastered != wantMastered {
			t.Fatalf("after %d correct answers: Mastered = %v, want %v", i, p.Mastered, wantMastered)
		}
	}
	if p.Streak != MasteryStreak {
		t.Errorf("Streak = %d, want %d", p.Streak, MasteryStreak)
	}
	if p.Attempts != MasteryStreak {
		t.Errorf("Attempts = %d, want %d", p.Attempts, MasteryStreak)
	}
}

func TestGradePatternSlotWrongAnswerResetsStreakButKeepsMastery(t *testing.T) {
	var p model.PatternProgress
	for range MasteryStreak {
		p = GradePatternSlot(p, true)
	}

	p = GradePatternSlot(p, false)
	if p.Streak != 0 {
		t.Errorf("wrong answer left streak at %d, want 0", p.Streak)
	}
	if !p.Mastered {
		t.Error("mastery should remain sticky after a later lapse")
	}
}
