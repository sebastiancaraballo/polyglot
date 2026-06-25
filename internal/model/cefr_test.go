package model

import "testing"

func TestCEFRValid(t *testing.T) {
	tests := map[CEFR]bool{
		A1:      true,
		A2:      true,
		B1:      true,
		B2:      true,
		C1:      true,
		C2:      true,
		"":      false,
		"A0":    false,
		"bogus": false,
	}
	for level, want := range tests {
		if got := level.Valid(); got != want {
			t.Errorf("CEFR(%q).Valid() = %v, want %v", level, got, want)
		}
	}
}
