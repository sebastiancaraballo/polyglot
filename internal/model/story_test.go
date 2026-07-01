package model

import "testing"

func TestBeatKindValid(t *testing.T) {
	tests := map[BeatKind]bool{
		Narration: true,
		Dialogue:  true,
		Practice:  true,
		"":        false,
		"bogus":   false,
	}
	for kind, want := range tests {
		if got := kind.Valid(); got != want {
			t.Errorf("BeatKind(%q).Valid() = %v, want %v", kind, got, want)
		}
	}
}

func TestPracticeKindValid(t *testing.T) {
	tests := map[PracticeKind]bool{
		PracticeVocab: true,
		PracticeKana:  true,
		"":            false,
		"bogus":       false,
	}
	for kind, want := range tests {
		if got := kind.Valid(); got != want {
			t.Errorf("PracticeKind(%q).Valid() = %v, want %v", kind, got, want)
		}
	}
}
