package profiles

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func newStore(t *testing.T) *storage.SQLiteStore {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func newTestModel(t *testing.T) (Model, int64, int64) {
	t.Helper()
	store := newStore(t)
	ctx := context.Background()
	first, err := store.CreateProfile(ctx, "Ana", "initials:0")
	if err != nil {
		t.Fatalf("CreateProfile first: %v", err)
	}
	second, err := store.CreateProfile(ctx, "Mei", "identicon:1")
	if err != nil {
		t.Fatalf("CreateProfile second: %v", err)
	}
	return New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ActiveID: first.ID}), first.ID, second.ID
}

func TestChooseProfileEmitsSwitchProfile(t *testing.T) {
	m, _, secondID := newTestModel(t)
	m.cursor = 1

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("selecting a profile should return a command")
	}
	msg, ok := cmd().(nav.SwitchProfileMsg)
	if !ok {
		t.Fatalf("expected nav.SwitchProfileMsg, got %T", cmd())
	}
	if msg.ID != secondID {
		t.Errorf("switch id = %d, want %d", msg.ID, secondID)
	}
}

func TestChooseCreateEmitsCreateProfile(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.cursor = len(m.profiles)

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("selecting create should return a command")
	}
	if _, ok := cmd().(nav.CreateProfileMsg); !ok {
		t.Fatalf("expected nav.CreateProfileMsg, got %T", cmd())
	}
}

func TestViewMarksActiveProfile(t *testing.T) {
	m, _, _ := newTestModel(t)
	content := m.View().Content
	for _, want := range []string{"Ana", "Mei", i18n.ES.ActiveProfileLabel, i18n.ES.ProfileCreateNew} {
		if !strings.Contains(content, want) {
			t.Errorf("view is missing %q", want)
		}
	}
}
