package study

import (
	"math/rand"
	"testing"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// assessmentLessons returns two N5 lessons and one N4 lesson, so tests can
// confirm the sampler filters by level.
func assessmentLessons() []model.Lesson {
	return []model.Lesson{
		{ID: "greetings", JLPT: model.N5, Cards: []model.Card{
			{ID: "greetings:1", Source: "Hola", JP: "こんにちは"},
			{ID: "greetings:2", Source: "Gracias", JP: "ありがとう"},
			{ID: "greetings:3", Source: "Adiós", JP: "さようなら"},
			{ID: "greetings:4", Source: "Sí", JP: "はい"},
		}},
		{ID: "self-intro", JLPT: model.N5, Cards: []model.Card{
			{ID: "self-intro:1", Source: "Yo", JP: "わたし"},
			{ID: "self-intro:2", Source: "Tú", JP: "あなた"},
			{ID: "self-intro:3", Source: "Estudiante", JP: "がくせい"},
		}},
		{ID: "n4-extra", JLPT: model.N4, Cards: []model.Card{
			{ID: "n4-extra:1", Source: "Economía", JP: "けいざい"},
		}},
	}
}

func assessmentKana() []model.KanaItem {
	return []model.KanaItem{
		{Char: "あ", Romaji: "a", Type: model.Hiragana, Category: model.Base},
		{Char: "い", Romaji: "i", Type: model.Hiragana, Category: model.Base},
		{Char: "う", Romaji: "u", Type: model.Hiragana, Category: model.Base},
	}
}

func assessmentPatterns() []model.Pattern {
	return []model.Pattern{
		{ID: "x-wa-n-desu", JLPT: model.N5, Frame: "{X}は{N}です", Slots: []model.Slot{
			{Name: "X", CardIDs: []string{"self-intro:1", "self-intro:2"}, Default: "self-intro:1"},
			{Name: "N", CardIDs: []string{"self-intro:3", "greetings:4"}, Default: "self-intro:3"},
		}},
		{ID: "n4-pattern", JLPT: model.N4, Frame: "{X}", Slots: []model.Slot{
			{Name: "X", CardIDs: []string{"n4-extra:1"}, Default: "n4-extra:1"},
		}},
	}
}

func assessmentCards(lessons []model.Lesson) map[string]model.Card {
	index := make(map[string]model.Card)
	for _, l := range lessons {
		for _, c := range l.Cards {
			index[c.ID] = c
		}
	}
	return index
}

func TestBuildAssessmentSamplesEveryStrand(t *testing.T) {
	lessons := assessmentLessons()
	rng := rand.New(rand.NewSource(1))
	qs := BuildAssessment(rng, model.N5, lessons, assessmentKana(), assessmentPatterns(), assessmentCards(lessons))
	if len(qs) == 0 {
		t.Fatal("expected questions")
	}
	kinds := map[AssessKind]int{}
	for _, q := range qs {
		kinds[q.Kind]++
	}
	if kinds[AssessVocab] == 0 || kinds[AssessKana] == 0 || kinds[AssessPattern] == 0 {
		t.Fatalf("every strand should contribute, got %v", kinds)
	}
}

func TestBuildAssessmentCapsAndDedupes(t *testing.T) {
	lessons := assessmentLessons()
	rng := rand.New(rand.NewSource(2))
	qs := BuildAssessment(rng, model.N5, lessons, assessmentKana(), assessmentPatterns(), assessmentCards(lessons))
	if len(qs) > AssessmentLength {
		t.Fatalf("got %d questions, want <= %d", len(qs), AssessmentLength)
	}
	seen := map[string]bool{}
	for _, q := range qs {
		if seen[q.key()] {
			t.Errorf("duplicate item %q", q.key())
		}
		seen[q.key()] = true
	}
}

func TestBuildAssessmentFiltersByLevel(t *testing.T) {
	lessons := assessmentLessons()
	rng := rand.New(rand.NewSource(3))
	qs := BuildAssessment(rng, model.N5, lessons, assessmentKana(), assessmentPatterns(), assessmentCards(lessons))
	for _, q := range qs {
		switch q.Kind {
		case AssessVocab:
			if q.Card.ID == "n4-extra:1" {
				t.Errorf("N4 card leaked into an N5 assessment: %q", q.Card.ID)
			}
		case AssessPattern:
			if q.Pattern.ID == "n4-pattern" {
				t.Errorf("N4 pattern leaked into an N5 assessment: %q", q.Pattern.ID)
			}
		}
	}
}

func TestBuildAssessmentOptionsAreValid(t *testing.T) {
	lessons := assessmentLessons()
	rng := rand.New(rand.NewSource(4))
	qs := BuildAssessment(rng, model.N5, lessons, assessmentKana(), assessmentPatterns(), assessmentCards(lessons))
	for _, q := range qs {
		if len(q.Options) == 0 {
			t.Fatalf("question %q has no options", q.key())
		}
		if q.Correct < 0 || q.Correct >= len(q.Options) {
			t.Fatalf("question %q correct index %d out of range (%d options)", q.key(), q.Correct, len(q.Options))
		}
		var want string
		switch q.Kind {
		case AssessKana:
			want = q.Kana.Romaji
		default:
			want = q.Card.JP // vocab prompt card, or pattern correct filler
		}
		if q.Options[q.Correct] != want {
			t.Errorf("question %q: Options[%d] = %q, want correct answer %q", q.key(), q.Correct, q.Options[q.Correct], want)
		}
	}
}

func TestBuildAssessmentPatternBlanksValidSlot(t *testing.T) {
	lessons := assessmentLessons()
	rng := rand.New(rand.NewSource(5))
	qs := BuildAssessment(rng, model.N5, lessons, assessmentKana(), assessmentPatterns(), assessmentCards(lessons))
	found := false
	for _, q := range qs {
		if q.Kind != AssessPattern {
			continue
		}
		found = true
		if q.SlotIdx < 0 || q.SlotIdx >= len(q.Pattern.Slots) {
			t.Fatalf("pattern %q blanked slot %d out of range", q.Pattern.ID, q.SlotIdx)
		}
		// Every non-blank slot must be pre-filled; the blanked one must not be.
		blanked := q.Pattern.Slots[q.SlotIdx].Name
		if _, ok := q.Fill[blanked]; ok {
			t.Errorf("pattern %q pre-filled the blanked slot %q", q.Pattern.ID, blanked)
		}
		for i, s := range q.Pattern.Slots {
			if i == q.SlotIdx {
				continue
			}
			if _, ok := q.Fill[s.Name]; !ok {
				t.Errorf("pattern %q missing fill for non-blank slot %q", q.Pattern.ID, s.Name)
			}
		}
	}
	if !found {
		t.Fatal("expected at least one pattern question")
	}
}

func TestBuildAssessmentEmptyCurriculum(t *testing.T) {
	rng := rand.New(rand.NewSource(6))
	if got := BuildAssessment(rng, model.N5, nil, nil, nil, nil); len(got) != 0 {
		t.Fatalf("BuildAssessment on empty curriculum = %v, want empty", got)
	}
}
