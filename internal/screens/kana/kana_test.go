package kana

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func newTestModel(t *testing.T) (Model, storage.Storage, int64) {
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

	m := Model{
		deps: Deps{
			Theme:     ui.NewTheme(true),
			Msgs:      i18n.ES,
			Store:     store,
			ProfileID: profile.ID,
		},
		deck:    []model.KanaItem{{Char: "あ", Romaji: "a", Type: model.Hiragana}},
		options: []string{"a"},
		correct: 0,
	}
	return m, store, profile.ID
}

func TestPickerStartsFilteredSession(t *testing.T) {
	items := []model.KanaItem{
		{Char: "あ", Romaji: "a", Type: model.Hiragana, Category: model.Base},
		{Char: "が", Romaji: "ga", Type: model.Hiragana, Category: model.Dakuten},
		{Char: "ぱ", Romaji: "pa", Type: model.Hiragana, Category: model.Handakuten},
		{Char: "きゃ", Romaji: "kya", Type: model.Hiragana, Category: model.Combo},
		{Char: "ア", Romaji: "a", Type: model.Katakana, Category: model.Base},
	}
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Items: items})
	if !m.picking {
		t.Fatal("trainer should open on the group picker")
	}

	// Move to "Hiragana · Dakuten / Handakuten" (index 2) and start.
	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	got := tm.(Model)
	if got.picking {
		t.Fatal("confirming a group should start the session")
	}
	if len(got.deck) != 2 {
		t.Fatalf("dakuten/handakuten deck size = %d, want 2", len(got.deck))
	}
	for _, it := range got.deck {
		if it.Category != model.Dakuten && it.Category != model.Handakuten {
			t.Errorf("deck contains out-of-group item %q (%s)", it.Char, it.Category)
		}
	}
}

func TestSpaceAnswersKanaQuestion(t *testing.T) {
	m, _, _ := newTestModel(t)

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	got := next.(Model)
	if !got.answered {
		t.Fatal("space should answer the current kana question")
	}
}

func TestAnsweringAwardsXP(t *testing.T) {
	m, store, profileID := newTestModel(t)

	// Selecting the correct option (key "1") answers and should award XP.
	next, _ := m.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	got := next.(Model)
	if got.err != nil {
		t.Fatalf("answering returned error: %v", got.err)
	}

	stats, err := store.GetStats(context.Background(), profileID)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.XP <= 0 {
		t.Errorf("XP after a correct answer = %d, want > 0", stats.XP)
	}
}

// gateItems is a minimal two-syllabary set whose base gojūon is one kana each,
// so a single mastered kana makes a syllabary fluent in tests.
var gateItems = []model.KanaItem{
	{Char: "あ", Romaji: "a", Type: model.Hiragana, Category: model.Base},
	{Char: "ア", Romaji: "a", Type: model.Katakana, Category: model.Base},
}

func katakanaGroupIndex(t *testing.T, m Model) int {
	t.Helper()
	for i, g := range m.groups {
		if strings.HasPrefix(g.label, i18n.ES.KatakanaLabel) {
			return i
		}
	}
	t.Fatal("no katakana group in picker")
	return 0
}

func TestKatakanaLockedUntilHiraganaFluent(t *testing.T) {
	store, profileID := newStore(t)
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: gateItems})

	idx := katakanaGroupIndex(t, m)
	if !m.groups[idx].locked {
		t.Fatal("katakana should be locked before hiragana fluency")
	}

	// Confirming a locked group must not start a session.
	m.groupCursor = idx
	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !tm.(Model).picking {
		t.Fatal("confirming a locked group should not start a session")
	}
}

func TestKatakanaUnlocksAfterHiraganaMastered(t *testing.T) {
	store, profileID := newStore(t)
	if err := store.SaveKanaProgress(context.Background(), profileID,
		model.KanaProgress{Char: "あ", Mastered: true}); err != nil {
		t.Fatalf("SaveKanaProgress: %v", err)
	}

	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: gateItems})
	idx := katakanaGroupIndex(t, m)
	if m.groups[idx].locked {
		t.Fatal("katakana should unlock once hiragana is fluent")
	}
}

func allGroupIndex(t *testing.T, m Model) int {
	t.Helper()
	for i, g := range m.groups {
		if g.label == i18n.ES.KanaGroupAll {
			return i
		}
	}
	t.Fatal("no \"all\" group in picker")
	return 0
}

func TestAllGroupGatedLikeKatakana(t *testing.T) {
	// The "Todo" (All) group spans both syllabaries, so it must honor the same
	// hiragana→katakana gate as the katakana groups: locked before hiragana
	// fluency, unlocked after. Otherwise it lets a learner practice katakana
	// before the gate opens.
	store, profileID := newStore(t)
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: gateItems})
	idx := allGroupIndex(t, m)
	if !m.groups[idx].locked {
		t.Fatal("\"all\" group should be locked before hiragana fluency")
	}

	// Confirming the locked group must not start a session.
	m.groupCursor = idx
	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !tm.(Model).picking {
		t.Fatal("confirming the locked \"all\" group should not start a session")
	}

	// Once hiragana is fluent, the gate opens for the "all" group too.
	if err := store.SaveKanaProgress(context.Background(), profileID,
		model.KanaProgress{Char: "あ", Mastered: true}); err != nil {
		t.Fatalf("SaveKanaProgress: %v", err)
	}
	m2 := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: gateItems})
	if m2.groups[allGroupIndex(t, m2)].locked {
		t.Fatal("\"all\" group should unlock once hiragana is fluent")
	}
}

func TestAnsweringPersistsKanaProgress(t *testing.T) {
	store, profileID := newStore(t)
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: gateItems})

	// Start the hiragana base group and answer the only card correctly.
	m.groupCursor = 1 // "Hiragana · Básico"
	m = m.startSession()
	m.selected = m.correct
	m = m.reveal()

	got, err := store.GetKanaProgress(context.Background(), profileID)
	if err != nil {
		t.Fatalf("GetKanaProgress: %v", err)
	}
	if got["あ"].Attempts == 0 {
		t.Fatalf("answering did not persist kana progress: %+v", got)
	}
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

func TestKanaTilePositionStableAfterAnswer(t *testing.T) {
	m := Model{
		deps: Deps{
			Theme: ui.PlainTheme(),
			Msgs:  i18n.ES,
		},
		deck: []model.KanaItem{{Char: "ミ", Romaji: "mi", Type: model.Katakana}},
		options: []string{
			"mu",
			"sa",
			"mi",
			"ne",
		},
		correct:  2,
		selected: 0,
		width:    80,
		height:   24,
	}

	before := kanaTileColumn(t, m.questionView())
	m.selected = 3
	m.answered = true
	after := kanaTileColumn(t, m.questionView())

	if before != after {
		t.Fatalf("tile column moved from %d to %d after answering", before, after)
	}
}

func kanaTileColumn(t *testing.T, view string) int {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if idx := strings.IndexRune(line, '╭'); idx >= 0 {
			return lipgloss.Width(line[:idx])
		}
	}
	t.Fatal("view does not contain kana tile")
	return 0
}
