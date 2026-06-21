package profilesetup

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/avatar"
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

func TestSubmitInvalidNameStaysOnNameStep(t *testing.T) {
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: newStore(t), Tutorial: true})
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("invalid submit returned command %T", cmd())
	}
	got := next.(Model)
	if got.step != stepName {
		t.Fatalf("step = %v, want stepName", got.step)
	}
	if !strings.Contains(got.View().Content, i18n.ES.ProfileNameEmpty) {
		t.Error("view should show the empty-name validation message")
	}
}

func TestSubmitValidNameMovesToAvatarStep(t *testing.T) {
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: newStore(t), Tutorial: true})
	m.name = "  José Niño  "

	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("valid name returned command %T before avatar selection", cmd())
	}
	got := next.(Model)
	if got.step != stepAvatar {
		t.Fatalf("step = %v, want stepAvatar", got.step)
	}
	if got.name != "José Niño" {
		t.Errorf("normalized name = %q, want José Niño", got.name)
	}
	if len(got.choices) != len(avatar.Options("José Niño")) {
		t.Errorf("choices = %d, want generated avatar options", len(got.choices))
	}
}

func TestCreateProfileEmitsProfileCreated(t *testing.T) {
	store := newStore(t)
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, Tutorial: false})
	m.name = "Mei"
	m.choices = avatar.Options("Mei")
	m.step = stepAvatar
	m.selected = 1

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("creating a profile should return a command")
	}
	msg, ok := cmd().(nav.ProfileCreatedMsg)
	if !ok {
		t.Fatalf("expected nav.ProfileCreatedMsg, got %T", cmd())
	}
	if msg.Tutorial {
		t.Error("created profile should preserve Tutorial=false")
	}

	profile, err := store.GetProfile(context.Background(), msg.ID)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if profile.Name != "Mei" || profile.Avatar != "identicon:0" {
		t.Errorf("profile = %+v, want name Mei avatar identicon:0", profile)
	}
}
