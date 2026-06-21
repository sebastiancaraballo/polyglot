package menu

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func newTestMenu() Model {
	summary := Summary{Name: "Sebastián", XP: 1240, Streak: 5, Learned: 8, Total: 20}
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
	if got := down.(Model).cursor; got != 2 {
		t.Fatalf("cursor after down = %d, want 2", got)
	}

	up, _ := down.(Model).Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if got := up.(Model).cursor; got != 1 {
		t.Fatalf("cursor after up = %d, want 1", got)
	}
}

func TestMenuCursorClamped(t *testing.T) {
	m := newTestMenu()
	m.cursor = 0
	// Pressing up at the top should not move past 0.
	up, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if got := up.(Model).cursor; got != 0 {
		t.Fatalf("cursor = %d, want 0", got)
	}
}

func TestMenuDefaultsToKana(t *testing.T) {
	m := newTestMenu()
	if m.cursor != 1 {
		t.Fatalf("default cursor = %d, want 1 for Kana", m.cursor)
	}
}

func TestMenuQuitItem(t *testing.T) {
	m := newTestMenu()
	m.cursor = len(m.items) // Quit
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

func TestMenuNavigates(t *testing.T) {
	m := newTestMenu()
	m.cursor = 1 // kana
	_, cmd := m.choose()
	if cmd == nil {
		t.Fatal("selecting a study screen should return a navigation command")
	}
	msg, ok := cmd().(nav.GoToMsg)
	if !ok {
		t.Fatalf("expected nav.GoToMsg, got %T", cmd())
	}
	if msg.Screen != nav.Kana {
		t.Errorf("navigated to %v, want Kana", msg.Screen)
	}
}

func TestMenuUsesTextSymbols(t *testing.T) {
	m := newTestMenu()
	want := []string{"◇", "▣", "✓", "▤", "⚙", "⏻"}
	if len(m.items) != len(want) {
		t.Fatalf("items = %d, want %d", len(m.items), len(want))
	}
	for i, wantIcon := range want {
		if got := m.items[i].icon; got != wantIcon {
			t.Errorf("item %d icon = %q, want %q", i, got, wantIcon)
		}
	}
}

func TestMenuSpaceNavigates(t *testing.T) {
	m := newTestMenu()
	m.cursor = 1 // kana

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if cmd == nil {
		t.Fatal("pressing space should return a navigation command")
	}
	msg, ok := cmd().(nav.GoToMsg)
	if !ok {
		t.Fatalf("expected nav.GoToMsg, got %T", cmd())
	}
	if msg.Screen != nav.Kana {
		t.Errorf("navigated to %v, want Kana", msg.Screen)
	}
}

func TestMenuViewShowsProgress(t *testing.T) {
	m := newTestMenu()
	content := m.View().Content
	for _, want := range []string{"Polyglot", "es → ja", "Sebastián", i18n.ES.SwitchProfile, i18n.ES.XPLabel, "1240", i18n.ES.StreakLabel, i18n.ES.ItemKana} {
		if !strings.Contains(content, want) {
			t.Errorf("view is missing %q", want)
		}
	}
}

func TestMenuProfileHeaderNavigatesToProfiles(t *testing.T) {
	m := newTestMenu()
	m.cursor = 0

	_, cmd := m.choose()
	if cmd == nil {
		t.Fatal("selecting the profile header should return a navigation command")
	}
	msg, ok := cmd().(nav.GoToMsg)
	if !ok {
		t.Fatalf("expected nav.GoToMsg, got %T", cmd())
	}
	if msg.Screen != nav.Profiles {
		t.Errorf("navigated to %v, want Profiles", msg.Screen)
	}
}
