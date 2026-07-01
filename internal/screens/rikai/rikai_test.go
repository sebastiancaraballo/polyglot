package rikai

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

// testPattern is a small two-slot pattern for tests, mirroring the shape of
// the real es-ja "X wa N desu" content without depending on it.
var testPattern = model.Pattern{
	ID:    "test-pattern",
	Title: "X wa N desu",
	JLPT:  model.N5,
	Frame: "{X}は{N}です",
	Slots: []model.Slot{
		{Name: "X", CardIDs: []string{"a:1", "a:2"}, Default: "a:1"},
		{Name: "N", CardIDs: []string{"b:1", "b:2", "b:3"}, Default: "b:1"},
	},
}

var testCards = map[string]model.Card{
	"a:1": {ID: "a:1", Source: "Yo", JP: "わたし", Romaji: "watashi"},
	"a:2": {ID: "a:2", Source: "Tú", JP: "あなた", Romaji: "anata"},
	"b:1": {ID: "b:1", Source: "Estudiante", JP: "がくせい", Romaji: "gakusei"},
	"b:2": {ID: "b:2", Source: "Profesor", JP: "せんせい", Romaji: "sensei"},
	"b:3": {ID: "b:3", Source: "Japonés", JP: "にほんじん", Romaji: "nihonjin"},
}

func newStore(t *testing.T) (storage.Storage, int64) {
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

// markKnown records a card as known (survived at least one review).
func markKnown(t *testing.T, store storage.Storage, profileID int64, cardID string) {
	t.Helper()
	if err := store.SaveCardState(context.Background(), profileID,
		model.CardState{CardID: cardID, Ease: model.DefaultEase, Reps: 1}); err != nil {
		t.Fatalf("SaveCardState: %v", err)
	}
}

func TestEmptyViewWhenNoPatterns(t *testing.T) {
	store, profileID := newStore(t)
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID})
	if len(m.patterns) != 0 {
		t.Fatalf("patterns = %d, want 0", len(m.patterns))
	}
	if !strings.Contains(m.View().Content, i18n.ES.RikaiLocked) {
		t.Error("empty state should show the RikaiLocked message")
	}
}

func TestPatternLockedWhenNoWordsKnown(t *testing.T) {
	store, profileID := newStore(t)
	m := New(Deps{
		Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID,
		Patterns: []model.Pattern{testPattern}, Cards: testCards,
	})
	if !m.patterns[0].locked {
		t.Fatal("pattern should be locked when no slot filler is known")
	}
}

func TestPatternUnlocksOnceEachSlotHasAKnownWord(t *testing.T) {
	store, profileID := newStore(t)
	markKnown(t, store, profileID, "a:1")
	markKnown(t, store, profileID, "b:2")

	m := New(Deps{
		Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID,
		Patterns: []model.Pattern{testPattern}, Cards: testCards,
	})
	if m.patterns[0].locked {
		t.Fatal("pattern should unlock once every slot has a known filler")
	}
}

func TestPickingLockedPatternDoesNotStartSession(t *testing.T) {
	store, profileID := newStore(t)
	m := New(Deps{
		Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID,
		Patterns: []model.Pattern{testPattern}, Cards: testCards,
	})

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !tm.(Model).picking {
		t.Fatal("confirming a locked pattern should not start a session")
	}
}

func TestStartSessionCyclesVariableSlot(t *testing.T) {
	store, profileID := newStore(t)
	markKnown(t, store, profileID, "a:1")
	markKnown(t, store, profileID, "a:2")
	markKnown(t, store, profileID, "b:1")
	markKnown(t, store, profileID, "b:2")

	m := New(Deps{
		Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID,
		Patterns: []model.Pattern{testPattern}, Cards: testCards,
	})
	m = m.startSession()
	if m.picking {
		t.Fatal("confirming an unlocked pattern should start a session")
	}
	if len(m.deck) != roundLimit {
		t.Fatalf("deck size = %d, want %d", len(m.deck), roundLimit)
	}
	for i, r := range m.deck {
		want := i % len(testPattern.Slots)
		if r.slotIdx != want {
			t.Errorf("round %d slotIdx = %d, want %d", i, r.slotIdx, want)
		}
	}
}

func TestAnsweringPersistsPatternProgressAndXP(t *testing.T) {
	store, profileID := newStore(t)
	markKnown(t, store, profileID, "a:1")
	markKnown(t, store, profileID, "b:1")

	m := New(Deps{
		Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID,
		Patterns: []model.Pattern{testPattern}, Cards: testCards,
	})
	m = m.startSession()
	m.selected = m.correct
	m = m.reveal()
	if m.err != nil {
		t.Fatalf("reveal returned error: %v", m.err)
	}

	stats, err := store.GetStats(context.Background(), profileID)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.XP <= 0 {
		t.Errorf("XP after a correct answer = %d, want > 0", stats.XP)
	}

	progress, err := store.GetPatternProgress(context.Background(), profileID)
	if err != nil {
		t.Fatalf("GetPatternProgress: %v", err)
	}
	slot := testPattern.Slots[m.deck[0].slotIdx]
	key := testPattern.ID + ":" + slot.Name
	if progress[key].Attempts == 0 {
		t.Fatalf("answering did not persist pattern progress: %+v", progress)
	}
}

func TestQuestionViewShowsBlankForVaryingSlotAndDefaultForFixedSlot(t *testing.T) {
	store, profileID := newStore(t)
	markKnown(t, store, profileID, "a:1")
	markKnown(t, store, profileID, "b:1")

	m := New(Deps{
		Theme: ui.PlainTheme(), Msgs: i18n.ES, Store: store, ProfileID: profileID,
		Patterns: []model.Pattern{testPattern}, Cards: testCards,
	})
	m = m.startSession()

	view := m.questionView()
	if !strings.Contains(view, blank) {
		t.Errorf("question view should show a blank marker; view:\n%s", view)
	}
	fixedSlot := testPattern.Slots[1-m.deck[0].slotIdx]
	fixedJP := testCards[fixedSlot.Default].JP
	if !strings.Contains(view, fixedJP) {
		t.Errorf("question view should show the fixed slot's default filler %q; view:\n%s", fixedJP, view)
	}
}
