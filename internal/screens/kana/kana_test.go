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
