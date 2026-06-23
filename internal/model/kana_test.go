package model

import "testing"

func TestKanaCategoryValid(t *testing.T) {
	tests := map[KanaCategory]bool{
		Base:       true,
		Dakuten:    true,
		Handakuten: true,
		Combo:      true,
		"":         false,
		"bogus":    false,
	}
	for cat, want := range tests {
		if got := cat.Valid(); got != want {
			t.Errorf("KanaCategory(%q).Valid() = %v, want %v", cat, got, want)
		}
	}
}
