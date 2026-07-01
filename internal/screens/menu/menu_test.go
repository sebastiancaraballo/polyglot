package menu

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/art"
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
	want := []string{"あ", "▦", "▣", "♻", "✓", "◧", "▧", "▤", "⚙", "⏻"}
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
	// The app name is rendered as the block wordmark in the header, not plain text.
	// Styling wraps each line separately, so match a single (top) wordmark line.
	if topLine := strings.SplitN(art.Wordmark, "\n", 2)[0]; !strings.Contains(content, topLine) {
		t.Error("view is missing the app wordmark")
	}
	for _, want := range []string{"es → ja", "Sebastián", i18n.ES.SwitchProfile, i18n.ES.XPLabel, "1240", i18n.ES.StreakLabel, i18n.ES.ItemKana} {
		if !strings.Contains(content, want) {
			t.Errorf("view is missing %q", want)
		}
	}
	for _, unwanted := range []string{"¿Qué quieres estudiar hoy?", "8/20"} {
		if strings.Contains(content, unwanted) {
			t.Errorf("view includes %q", unwanted)
		}
	}
}

func TestMenuNarrowWidthFallsBackToTextTitle(t *testing.T) {
	m := newTestMenu()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 24})
	content := updated.(Model).View().Content
	if strings.Contains(content, art.Wordmark) {
		t.Error("narrow view should drop the block wordmark")
	}
	if !strings.Contains(content, i18n.ES.AppName) {
		t.Errorf("narrow view should keep the plain text title %q", i18n.ES.AppName)
	}
}

func lockedMenu() Model {
	summary := Summary{Name: "Sebastián", XP: 0, Streak: 0, Learned: 0, Total: 20, ReadingLocked: true}
	return New(ui.NewTheme(true), i18n.ES, summary, "test")
}

func itemIndex(m Model, screen nav.Screen) int {
	for i, it := range m.items {
		if it.screen == screen {
			return i
		}
	}
	return -1
}

func TestLockedReadingItemDoesNotNavigate(t *testing.T) {
	m := lockedMenu()
	m.cursor = itemIndex(m, nav.Flashcards) + 1 // cursor is 1-based (0 is the profile header)

	next, cmd := m.choose()
	if cmd != nil {
		t.Fatalf("opening a locked item should not navigate, got cmd %T", cmd())
	}
	if got := next.(Model).notice; got != i18n.ES.ReadingLocked {
		t.Errorf("notice = %q, want the reading-locked hint", got)
	}
}

func TestLockedMenuViewMarksAndExplains(t *testing.T) {
	m := lockedMenu()
	m.cursor = itemIndex(m, nav.Quiz) + 1
	m, _ = func() (Model, tea.Cmd) { tm, cmd := m.choose(); return tm.(Model), cmd }()

	content := m.View().Content
	if !strings.Contains(content, lockGlyph) {
		t.Errorf("locked menu view missing the lock glyph %q", lockGlyph)
	}
	if !strings.Contains(content, i18n.ES.ReadingLocked) {
		t.Errorf("locked menu view missing the reading-locked notice")
	}
}

func TestLockedItemReplacesIconWithLock(t *testing.T) {
	m := lockedMenu()
	for _, it := range m.items {
		if it.screen == nav.Quiz && !it.locked {
			t.Fatal("Quiz should be locked in a locked menu")
		}
	}
	// The lock glyph must appear once per locked reading item (Flashcards, Quiz).
	if got := strings.Count(m.menuItems(), lockGlyph); got != 2 {
		t.Errorf("lock glyphs in menu = %d, want 2", got)
	}
}

func TestUnlockedMenuHasNoLocks(t *testing.T) {
	m := newTestMenu() // ReadingLocked defaults to false
	for _, it := range m.items {
		if it.locked {
			t.Fatalf("item %q is locked when reading is unlocked", it.label)
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
