package kana

import (
	"context"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

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

	profile, err := store.CreateProfile(context.Background(), "tester", "")
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
