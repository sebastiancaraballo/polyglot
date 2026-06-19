package menu

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func newTestMenu() Model {
	summary := Summary{Level: "N5", NextLevel: "N4", Percent: 40, Streak: 5, Learned: 8, Total: 20}
	return New(ui.NewTheme(true), i18n.ES, summary, "test")
}

func key(s string) tea.KeyPressMsg {
	if len(s) == 1 {
		return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
	}
	return tea.KeyPressMsg{}
}

func TestMenuNavigationMovesCursor(t *testing.T) {
	m := newTestMenu()

	down, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := down.(Model).cursor; got != 1 {
		t.Fatalf("cursor after down = %d, want 1", got)
	}

	up, _ := down.(Model).Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if got := up.(Model).cursor; got != 0 {
		t.Fatalf("cursor after up = %d, want 0", got)
	}
}

func TestMenuCursorClamped(t *testing.T) {
	m := newTestMenu()
	// Pressing up at the top should not move past 0.
	up, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if got := up.(Model).cursor; got != 0 {
		t.Fatalf("cursor = %d, want 0", got)
	}
}

func TestMenuQuitItem(t *testing.T) {
	m := newTestMenu()
	m.cursor = len(m.items) - 1 // Quit
	_, cmd := m.choose()
	if cmd == nil {
		t.Fatal("selecting Quit should return a command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("selecting Quit should return tea.QuitMsg")
	}
}

func TestMenuQuitKey(t *testing.T) {
	m := newTestMenu()
	_, cmd := m.Update(key("q"))
	if cmd == nil {
		t.Fatal("pressing q should return a quit command")
	}
}

func TestMenuComingSoon(t *testing.T) {
	m := newTestMenu()
	m.cursor = 0 // a study screen, not yet wired
	updated, cmd := m.choose()
	if cmd != nil {
		t.Fatal("selecting an unimplemented screen should not issue a command")
	}
	if updated.(Model).status != i18n.ES.ComingSoon {
		t.Fatalf("status = %q, want %q", updated.(Model).status, i18n.ES.ComingSoon)
	}
}

func TestMenuViewShowsProgress(t *testing.T) {
	m := newTestMenu()
	content := m.View().Content
	for _, want := range []string{"Polyglot", "N5", "40%", i18n.ES.StreakLabel, i18n.ES.ItemKana} {
		if !strings.Contains(content, want) {
			t.Errorf("view is missing %q", want)
		}
	}
}
