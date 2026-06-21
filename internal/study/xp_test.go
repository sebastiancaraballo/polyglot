package study

import (
	"testing"

	"github.com/sebastiancaraballo/polyglot/internal/srs"
)

func TestXPForGrade(t *testing.T) {
	tests := []struct {
		grade srs.Grade
		want  int
	}{
		{srs.Again, 2},
		{srs.Hard, 6},
		{srs.Good, 10},
		{srs.Easy, 14},
	}
	for _, tt := range tests {
		if got := XPForGrade(tt.grade); got != tt.want {
			t.Errorf("XPForGrade(%v) = %d, want %d", tt.grade, got, tt.want)
		}
	}
}

func TestXPForGradeOrdersByAccuracy(t *testing.T) {
	if XPForGrade(srs.Again) >= XPForGrade(srs.Good) {
		t.Error("a correct recall should earn more XP than forgetting")
	}
}

func TestXPForAnswer(t *testing.T) {
	if got, want := XPForAnswer(true), XPForGrade(srs.Good); got != want {
		t.Errorf("XPForAnswer(true) = %d, want %d", got, want)
	}
	if got, want := XPForAnswer(false), XPForGrade(srs.Again); got != want {
		t.Errorf("XPForAnswer(false) = %d, want %d", got, want)
	}
	if XPForAnswer(false) >= XPForAnswer(true) {
		t.Error("a correct answer should earn more XP than an incorrect one")
	}
}
