package assessment

import (
	"context"
	"math/rand"
	"path/filepath"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/study"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func testLessons() []model.Lesson {
	return []model.Lesson{
		{ID: "greetings", JLPT: model.N5, Cards: []model.Card{
			{ID: "greetings:1", Source: "Hola", JP: "こんにちは", Romaji: "konnichiwa"},
			{ID: "greetings:2", Source: "Gracias", JP: "ありがとう", Romaji: "arigatou"},
			{ID: "greetings:3", Source: "Adiós", JP: "さようなら", Romaji: "sayounara"},
			{ID: "greetings:4", Source: "Sí", JP: "はい", Romaji: "hai"},
		}},
		{ID: "question-words", JLPT: model.N5, Cards: []model.Card{
			// A deliberately long gloss to stress the prompt's wrapping.
			{ID: "question-words:1", Source: "Cuál (de dos) / Dónde (cortés)", JP: "どちら", Romaji: "dochira"},
			{ID: "question-words:2", Source: "Dónde", JP: "どこ", Romaji: "doko"},
			{ID: "question-words:3", Source: "Cómo", JP: "どう", Romaji: "dou"},
		}},
	}
}

func testKana() []model.KanaItem {
	return []model.KanaItem{
		{Char: "あ", Romaji: "a", Type: model.Hiragana, Category: model.Base},
		{Char: "い", Romaji: "i", Type: model.Hiragana, Category: model.Base},
		{Char: "う", Romaji: "u", Type: model.Hiragana, Category: model.Base},
		{Char: "え", Romaji: "e", Type: model.Hiragana, Category: model.Base},
	}
}

func testPatterns() []model.Pattern {
	return []model.Pattern{
		{ID: "x-wa-n-desu", JLPT: model.N5, Frame: "{X}は{N}です", Slots: []model.Slot{
			{Name: "X", CardIDs: []string{"greetings:1", "greetings:2"}, Default: "greetings:1"},
			{Name: "N", CardIDs: []string{"question-words:2", "question-words:3"}, Default: "question-words:2"},
		}},
	}
}

func testCards(lessons []model.Lesson) map[string]model.Card {
	index := map[string]model.Card{}
	for _, l := range lessons {
		for _, c := range l.Cards {
			index[c.ID] = c
		}
	}
	return index
}

func testDeps(store storage.Storage, profileID int64, showRomaji bool) Deps {
	lessons := testLessons()
	return Deps{
		Theme: ui.PlainTheme(), Msgs: i18n.ES, Store: store, ProfileID: profileID,
		Level: model.N5, Lessons: lessons, Kana: testKana(), Patterns: testPatterns(),
		Cards: testCards(lessons), ShowRomaji: showRomaji,
	}
}

func newTestStore(t *testing.T) (storage.Storage, int64) {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	profile, err := store.CreateProfile(context.Background(), "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	return store, profile.ID
}

func TestAssessmentPassPersistsResult(t *testing.T) {
	store, pid := newTestStore(t)
	m := New(testDeps(store, pid, true))
	m = m.start()
	for m.phase == phaseQuestion {
		m.selected = m.deck[m.index].Correct // answer every question correctly
		m = m.reveal()
		m = m.advance()
	}
	if m.phase != phaseResult {
		t.Fatalf("phase = %v, want result", m.phase)
	}
	if !m.attemptPassed() {
		t.Fatalf("all-correct attempt should pass (%d/%d)", m.correctCount, len(m.deck))
	}
	res, err := store.GetAssessmentResult(context.Background(), pid, model.N5)
	if err != nil {
		t.Fatalf("GetAssessmentResult: %v", err)
	}
	if !res.Passed {
		t.Error("passing attempt should persist Passed=true")
	}
	if res.BestCorrect != len(m.deck) || res.Total != len(m.deck) {
		t.Errorf("best score = %d/%d, want %d/%d", res.BestCorrect, res.Total, len(m.deck), len(m.deck))
	}
}

func TestAssessmentFailAllowsRetryAndKeepsStickyPass(t *testing.T) {
	store, pid := newTestStore(t)
	m := New(testDeps(store, pid, false))
	m = m.start()
	for m.phase == phaseQuestion {
		q := m.deck[m.index]
		m.selected = (q.Correct + 1) % len(q.Options) // always wrong
		m = m.reveal()
		m = m.advance()
	}
	if m.attemptPassed() {
		t.Fatal("all-wrong attempt should not pass")
	}
	res, err := store.GetAssessmentResult(context.Background(), pid, model.N5)
	if err != nil {
		t.Fatalf("GetAssessmentResult: %v", err)
	}
	if res.Passed {
		t.Error("a failed first attempt should not be marked passed")
	}
	// Retry re-samples a fresh deck and returns to the question phase.
	m = m.restart()
	if m.phase != phaseQuestion {
		t.Fatalf("after restart phase = %v, want question", m.phase)
	}
}

// TestAssessmentFitsFrame renders the intro, every question (answered and not),
// and both result screens (a large all-missed failure and a pass) at the real
// frame size, asserting nothing overflows — width (long, space-less Japanese
// must wrap) or height (the review list must cap).
func TestAssessmentFitsFrame(t *testing.T) {
	th := ui.PlainTheme()
	const termW, termH = 80, 30
	maxW := ui.FrameContentWidth(th, termW)
	maxH := ui.FrameContentHeight(th, termH)
	lessons := testLessons()

	for _, showRomaji := range []bool{true, false} {
		deps := Deps{
			Theme: th, Msgs: i18n.ES, Level: model.N5, Lessons: lessons,
			Kana: testKana(), Patterns: testPatterns(), Cards: testCards(lessons), ShowRomaji: showRomaji,
		}
		m := Model{deps: deps, rng: rand.New(rand.NewSource(1)), width: termW, height: termH, romaji: map[string]string{}}
		for _, c := range deps.Cards {
			m.romaji[c.JP] = c.Romaji
		}
		m.deck = study.BuildAssessment(m.rng, model.N5, deps.Lessons, deps.Kana, deps.Patterns, deps.Cards)

		check := func(label, content string) {
			if w := lipgloss.Width(content); w > maxW {
				t.Errorf("[WIDTH] romaji=%v %s: %d > %d", showRomaji, label, w, maxW)
			}
			if h := lipgloss.Height(content); h > maxH {
				t.Errorf("[HEIGHT] romaji=%v %s: %d > %d", showRomaji, label, h, maxH)
			}
		}

		m.phase = phaseIntro
		check("intro", m.introView())

		m.phase = phaseQuestion
		for i := range m.deck {
			m.index = i
			m.answered = false
			check("question", m.questionView())
			m.selected = 0
			m.answered = true
			check("question-answered", m.questionView())
		}

		// A worst-case failure: every question missed, forcing the review list
		// to cap rather than overflow.
		m.phase = phaseResult
		m.correctCount = 0
		m.missed = append([]study.AssessQuestion(nil), m.deck...)
		check("result-fail", m.resultView())

		m.correctCount = len(m.deck)
		m.missed = nil
		check("result-pass", m.resultView())
	}
}
