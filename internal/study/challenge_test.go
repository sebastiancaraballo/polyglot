package study

import (
	"math/rand"
	"testing"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

func TestChallengePassed(t *testing.T) {
	tests := []struct {
		name           string
		correct, total int
		want           bool
	}{
		{"zero of zero", 0, 0, false},
		{"three of five fails", 3, 5, false},
		{"four of five passes", 4, 5, true},
		{"five of five passes", 5, 5, true},
		{"two of three fails (short pools need all)", 2, 3, false},
		{"three of three passes", 3, 3, true},
		{"three of four fails", 3, 4, false},
		{"four of four passes", 4, 4, true},
		{"eight of ten passes", 8, 10, true},
		{"seven of ten fails", 7, 10, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ChallengePassed(tt.correct, tt.total); got != tt.want {
				t.Errorf("ChallengePassed(%d, %d) = %v, want %v", tt.correct, tt.total, got, tt.want)
			}
		})
	}
}

func TestChallengeNeededMatchesChallengePassed(t *testing.T) {
	for total := 1; total <= 12; total++ {
		need := ChallengeNeeded(total)
		if !ChallengePassed(need, total) {
			t.Errorf("ChallengeNeeded(%d) = %d, but ChallengePassed(%d, %d) is false", total, need, need, total)
		}
		if need > 0 && ChallengePassed(need-1, total) {
			t.Errorf("ChallengeNeeded(%d) = %d is not minimal: %d also passes", total, need, need-1)
		}
	}
}

func challengeLessons() []model.Lesson {
	return []model.Lesson{
		{ID: "greetings", Cards: []model.Card{
			{ID: "greetings:1", Source: "Hola", JP: "こんにちは", Romaji: "konnichiwa"},
			{ID: "greetings:2", Source: "Gracias", JP: "ありがとう", Romaji: "arigatou"},
			{ID: "greetings:3", Source: "Adiós", JP: "さようなら", Romaji: "sayounara"},
			{ID: "greetings:4", Source: "Sí", JP: "はい", Romaji: "hai"},
			{ID: "greetings:5", Source: "No", JP: "いいえ", Romaji: "iie"},
			{ID: "greetings:6", Source: "Por favor", JP: "おねがいします", Romaji: "onegaishimasu"},
		}},
	}
}

func challengeKana() []model.KanaItem {
	return []model.KanaItem{
		{Char: "あ", Romaji: "a", Type: model.Hiragana, Category: model.Base},
		{Char: "い", Romaji: "i", Type: model.Hiragana, Category: model.Base},
		{Char: "う", Romaji: "u", Type: model.Hiragana, Category: model.Base},
	}
}

func practiceBeat(kind model.PracticeKind, ref string) model.Beat {
	return model.Beat{Kind: model.Practice, Practice: kind, RefID: ref}
}

func TestBuildChallengeNoPracticeBeatsReturnsNil(t *testing.T) {
	chapter := model.Chapter{ID: "c", Beats: []model.Beat{
		{Kind: model.Narration, Source: "x", JP: "y"},
	}}
	rng := rand.New(rand.NewSource(1))
	if got := BuildChallenge(rng, chapter, challengeLessons(), challengeKana()); got != nil {
		t.Fatalf("BuildChallenge = %v, want nil for a chapter with no practice beats", got)
	}
}

func TestBuildChallengeDrawsDistinctCards(t *testing.T) {
	chapter := model.Chapter{ID: "c", Beats: []model.Beat{practiceBeat(model.PracticeVocab, "greetings")}}
	rng := rand.New(rand.NewSource(1))
	qs := BuildChallenge(rng, chapter, challengeLessons(), challengeKana())
	if len(qs) != ChallengeLength {
		t.Fatalf("got %d questions, want %d", len(qs), ChallengeLength)
	}
	seen := make(map[string]bool)
	for _, q := range qs {
		if q.Practice != model.PracticeVocab || q.RefID != "greetings" {
			t.Errorf("unexpected question source: %+v", q)
		}
		if seen[q.Card.ID] {
			t.Errorf("duplicate card %q in challenge", q.Card.ID)
		}
		seen[q.Card.ID] = true
	}
}

func TestBuildChallengeRepresentsEveryPool(t *testing.T) {
	chapter := model.Chapter{ID: "c", Beats: []model.Beat{
		practiceBeat(model.PracticeVocab, "greetings"),
		practiceBeat(model.PracticeKana, "hiragana"),
	}}
	rng := rand.New(rand.NewSource(1))
	qs := BuildChallenge(rng, chapter, challengeLessons(), challengeKana())
	if len(qs) != ChallengeLength {
		t.Fatalf("got %d questions, want %d", len(qs), ChallengeLength)
	}
	kinds := make(map[model.PracticeKind]int)
	for _, q := range qs {
		kinds[q.Practice]++
	}
	if kinds[model.PracticeVocab] == 0 || kinds[model.PracticeKana] == 0 {
		t.Fatalf("both pools should contribute, got %v", kinds)
	}
}

func TestBuildChallengeCapsAtPoolSize(t *testing.T) {
	// Only the 3-kana pool is referenced: the challenge is shorter than
	// ChallengeLength rather than repeating items.
	chapter := model.Chapter{ID: "c", Beats: []model.Beat{practiceBeat(model.PracticeKana, "hiragana")}}
	rng := rand.New(rand.NewSource(1))
	qs := BuildChallenge(rng, chapter, challengeLessons(), challengeKana())
	if len(qs) != 3 {
		t.Fatalf("got %d questions, want 3 (pool size)", len(qs))
	}
	seen := make(map[string]bool)
	for _, q := range qs {
		if seen[q.Kana.Char] {
			t.Errorf("duplicate kana %q in challenge", q.Kana.Char)
		}
		seen[q.Kana.Char] = true
	}
}

func TestBuildChallengeDedupesRepeatedRefs(t *testing.T) {
	// Two practice beats referencing the same lesson must not double the pool.
	chapter := model.Chapter{ID: "c", Beats: []model.Beat{
		practiceBeat(model.PracticeVocab, "greetings"),
		practiceBeat(model.PracticeVocab, "greetings"),
	}}
	rng := rand.New(rand.NewSource(1))
	qs := BuildChallenge(rng, chapter, challengeLessons(), challengeKana())
	seen := make(map[string]bool)
	for _, q := range qs {
		if seen[q.Card.ID] {
			t.Errorf("duplicate card %q despite repeated refs", q.Card.ID)
		}
		seen[q.Card.ID] = true
	}
}

func TestBuildChallengeUnknownRefYieldsNoQuestions(t *testing.T) {
	// Content validation prevents this, but the pure function must not panic.
	chapter := model.Chapter{ID: "c", Beats: []model.Beat{practiceBeat(model.PracticeVocab, "nope")}}
	rng := rand.New(rand.NewSource(1))
	if got := BuildChallenge(rng, chapter, challengeLessons(), challengeKana()); got != nil {
		t.Fatalf("BuildChallenge = %v, want nil for an unresolvable ref", got)
	}
}
