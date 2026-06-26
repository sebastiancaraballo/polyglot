package study

import (
	"testing"
	"time"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

func TestGradeKanaMasteryRequiresAccurateRun(t *testing.T) {
	const elapsed = time.Second
	var p model.KanaProgress

	// A run of correct answers reaches mastery exactly at MasteryStreak.
	for i := 1; i <= MasteryStreak; i++ {
		p = GradeKana(p, true, elapsed)
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

func TestGradeKanaSlowAnswerStillAdvancesStreak(t *testing.T) {
	// Response time does not gate the streak: a slow but correct answer counts.
	slow := 30 * time.Second
	p := GradeKana(model.KanaProgress{}, true, slow)
	if p.Streak != 1 {
		t.Errorf("slow correct answer advanced streak to %d, want 1", p.Streak)
	}
	if p.BestMs == 0 {
		t.Error("a correct answer should still record a best time")
	}
}

func TestGradeKanaWrongAnswerResetsStreakButKeepsMastery(t *testing.T) {
	const elapsed = time.Second
	var p model.KanaProgress
	for range MasteryStreak {
		p = GradeKana(p, true, elapsed)
	}

	p = GradeKana(p, false, elapsed)
	if p.Streak != 0 {
		t.Errorf("wrong answer left streak at %d, want 0", p.Streak)
	}
	if !p.Mastered {
		t.Error("mastery should remain sticky after a later lapse")
	}
}

func TestGradeKanaUntimedCorrectAnswerAdvancesStreak(t *testing.T) {
	// An untimed answer (elapsed == 0) still counts toward mastery; it just does
	// not record a best time.
	p := GradeKana(model.KanaProgress{}, true, 0)
	if p.Streak != 1 {
		t.Errorf("untimed correct answer advanced streak to %d, want 1", p.Streak)
	}
	if p.BestMs != 0 {
		t.Errorf("untimed answer recorded BestMs = %d, want 0", p.BestMs)
	}
}

func mastered(chars ...string) map[string]model.KanaProgress {
	m := make(map[string]model.KanaProgress, len(chars))
	for _, c := range chars {
		m[c] = model.KanaProgress{Char: c, Mastered: true}
	}
	return m
}

func TestGateUnlocksInOrder(t *testing.T) {
	items := []model.KanaItem{
		{Char: "あ", Type: model.Hiragana, Category: model.Base},
		{Char: "い", Type: model.Hiragana, Category: model.Base},
		{Char: "が", Type: model.Hiragana, Category: model.Dakuten}, // derived: not part of the gate
		{Char: "ア", Type: model.Katakana, Category: model.Base},
		{Char: "イ", Type: model.Katakana, Category: model.Base},
	}

	tests := []struct {
		name             string
		progress         map[string]model.KanaProgress
		wantKatakana     bool
		wantReading      bool
		wantHiraFluent   bool
		wantKataFluent   bool
		wantHiraMastered int
	}{
		{
			name:     "nothing mastered locks everything",
			progress: mastered(),
		},
		{
			name:             "partial hiragana still locked",
			progress:         mastered("あ"),
			wantHiraMastered: 1,
		},
		{
			name:           "all hiragana base unlocks katakana only",
			progress:       mastered("あ", "い"),
			wantKatakana:   true,
			wantHiraFluent: true,
		},
		{
			name:           "both syllabaries fluent unlocks reading",
			progress:       mastered("あ", "い", "ア", "イ"),
			wantKatakana:   true,
			wantReading:    true,
			wantHiraFluent: true,
			wantKataFluent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGate(items, tt.progress)
			if g.KatakanaUnlocked() != tt.wantKatakana {
				t.Errorf("KatakanaUnlocked() = %v, want %v", g.KatakanaUnlocked(), tt.wantKatakana)
			}
			if g.ReadingUnlocked() != tt.wantReading {
				t.Errorf("ReadingUnlocked() = %v, want %v", g.ReadingUnlocked(), tt.wantReading)
			}
			if g.Hiragana.Fluent() != tt.wantHiraFluent {
				t.Errorf("Hiragana.Fluent() = %v, want %v", g.Hiragana.Fluent(), tt.wantHiraFluent)
			}
			if g.Katakana.Fluent() != tt.wantKataFluent {
				t.Errorf("Katakana.Fluent() = %v, want %v", g.Katakana.Fluent(), tt.wantKataFluent)
			}
			if tt.wantHiraMastered != 0 && g.Hiragana.Mastered != tt.wantHiraMastered {
				t.Errorf("Hiragana.Mastered = %d, want %d", g.Hiragana.Mastered, tt.wantHiraMastered)
			}
			// The base gate must ignore derived (dakuten) kana: only 2 base hiragana.
			if g.Hiragana.Total != 2 {
				t.Errorf("Hiragana.Total = %d, want 2 (base only)", g.Hiragana.Total)
			}
		})
	}
}
