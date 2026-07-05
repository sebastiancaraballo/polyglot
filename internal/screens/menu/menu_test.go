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

// openLeaf positions the menu on the leaf that navigates to screen, descending
// into the category that holds it. It fails the test if no leaf targets screen.
func openLeaf(t *testing.T, m Model, screen nav.Screen) Model {
	t.Helper()
	for ci, cat := range m.items {
		for li, child := range cat.children {
			if child.screen == screen && !child.quit {
				m.section = ci
				m.cursor = li
				return m
			}
		}
	}
	t.Fatalf("no leaf navigates to %v", screen)
	return m
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

func TestMenuDefaultsToFirstCategory(t *testing.T) {
	m := newTestMenu()
	if m.section != topLevel {
		t.Fatalf("menu should start at the top level, got section %d", m.section)
	}
	if m.cursor != 1 {
		t.Fatalf("default cursor = %d, want 1 (first category)", m.cursor)
	}
	first := m.items[m.cursor-1]
	if len(first.children) == 0 {
		t.Fatalf("first item %q should be a category with children", first.label)
	}
	if first.label != i18n.ES.CatLearn {
		t.Errorf("first category = %q, want %q", first.label, i18n.ES.CatLearn)
	}
}

func TestMenuEntersCategory(t *testing.T) {
	m := newTestMenu() // cursor 1 = first category
	next, cmd := m.choose()
	if cmd != nil {
		t.Fatalf("entering a category should not navigate, got cmd %T", cmd())
	}
	nm := next.(Model)
	if nm.section != 0 {
		t.Errorf("section = %d, want 0 (first category open)", nm.section)
	}
	if nm.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (first child selected)", nm.cursor)
	}
}

func TestMenuBackFromCategory(t *testing.T) {
	// Open the third category (Evaluar), then go back.
	m := newTestMenu()
	m.section = 2
	m.cursor = 1
	back, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	bm := back.(Model)
	if bm.section != topLevel {
		t.Errorf("section = %d, want topLevel after going back", bm.section)
	}
	if bm.cursor != 3 { // category index 2 sits at cursor 3 (profile row is 0)
		t.Errorf("cursor = %d, want 3 (restored to the category)", bm.cursor)
	}

	// Left arrow goes back too.
	m2 := newTestMenu()
	m2.section = 0
	left, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if got := left.(Model).section; got != topLevel {
		t.Errorf("left arrow should leave the category, section = %d", got)
	}
}

func TestMenuBackAtTopLevelIsNoOp(t *testing.T) {
	m := newTestMenu()
	back, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd != nil {
		t.Fatalf("esc at the top level should do nothing, got cmd %T", cmd())
	}
	if got := back.(Model).section; got != topLevel {
		t.Errorf("section = %d, want topLevel", got)
	}
}

func TestMenuCursorClampedInSubmenu(t *testing.T) {
	m := newTestMenu()
	m.section = 0 // Aprender (3 children)
	m.cursor = len(m.items[0].children) - 1
	down, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := down.(Model).cursor; got != len(m.items[0].children)-1 {
		t.Errorf("cursor = %d, should clamp at last child", got)
	}
}

func TestMenuQuitItem(t *testing.T) {
	m := newTestMenu()
	m.cursor = len(m.items) // Quit is the last top-level leaf
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
	m := openLeaf(t, newTestMenu(), nav.Kana)
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
	want := []string{catLearnGlyph, catReadGlyph, catEvalGlyph, catToolsGlyph, "⚙", "⏻"}
	if len(m.items) != len(want) {
		t.Fatalf("top-level items = %d, want %d", len(m.items), len(want))
	}
	for i, wantIcon := range want {
		if got := m.items[i].icon; got != wantIcon {
			t.Errorf("item %d icon = %q, want %q", i, got, wantIcon)
		}
	}
}

func TestMenuSpaceNavigates(t *testing.T) {
	m := openLeaf(t, newTestMenu(), nav.Kana)

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
	// With the grouped top level the block wordmark fits the frame's fixed
	// content height again, so it renders (the plain-text app name it stands in
	// for is therefore absent).
	if topLine := strings.SplitN(art.Wordmark, "\n", 2)[0]; !strings.Contains(content, topLine) {
		t.Error("view is missing the block wordmark")
	}
	for _, want := range []string{"es → ja", "Sebastián", i18n.ES.SwitchProfile, i18n.ES.XPLabel, "1240", i18n.ES.StreakLabel, i18n.ES.CatLearn, i18n.ES.CatEvaluate} {
		if !strings.Contains(content, want) {
			t.Errorf("view is missing %q", want)
		}
	}
	// Activities live one level down, so their labels are not shown at the top.
	for _, unwanted := range []string{i18n.ES.ItemKana, "8/20"} {
		if strings.Contains(content, unwanted) {
			t.Errorf("view includes %q from a submenu", unwanted)
		}
	}
}

func TestMenuSubmenuShowsChildren(t *testing.T) {
	m := newTestMenu()
	m.section = 0 // Aprender
	content := m.View().Content
	for _, want := range []string{i18n.ES.ItemKana, i18n.ES.ItemFlashcards, i18n.ES.ItemRikai} {
		if !strings.Contains(content, want) {
			t.Errorf("Aprender submenu is missing %q", want)
		}
	}
	// The back hint replaces the top-level help, and the profile switcher is gone.
	if !strings.Contains(content, i18n.ES.MenuHelpSub) {
		t.Error("submenu should show the back hint in the footer")
	}
	if strings.Contains(content, i18n.ES.SwitchProfile) {
		t.Error("submenu should not show the profile switcher")
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

// TestMenuDropsWordmarkWhenLevelOverflows is a regression test for the wordmark
// fallback: if a level's list is long enough that the header block wordmark plus
// the globe/info columns no longer fit the frame's fixed content height (an
// upper bound that does not grow with a taller terminal — see ui.Frame), the
// wordmark is dropped rather than pushing the frame's own bottom border out of
// view. A generously tall window confirms this isn't a small-terminal edge case.
func TestMenuDropsWordmarkWhenLevelOverflows(t *testing.T) {
	// Eleven rows is the count that first pushed the wordmark past the fixed
	// budget (the flat menu this change replaced): the fallback still fits, but
	// the wordmark on top does not.
	many := make([]item, 0, 11)
	for range 11 {
		many = append(many, item{icon: "▤", label: i18n.ES.ItemStats, screen: nav.Stats})
	}
	m := Model{
		theme:   ui.NewTheme(true),
		msgs:    i18n.ES,
		summary: Summary{Name: "Sebastián"},
		version: "test",
		section: topLevel,
		cursor:  1,
		items:   many,
	}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 100})
	content := updated.(Model).View().Content
	if strings.Contains(content, art.Wordmark) {
		t.Error("view should drop the block wordmark once it no longer fits the fixed content height")
	}
	if !strings.Contains(content, "╰") {
		t.Error("frame's bottom border should still be visible, not clipped off-screen")
	}
}

// TestMenuKeepsWordmarkWhenItFits is the acceptance criterion: the real grouped
// menu has few enough top-level rows that the block wordmark fits and renders.
func TestMenuKeepsWordmarkWhenItFits(t *testing.T) {
	m := newTestMenu()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 100})
	content := updated.(Model).View().Content
	if topLine := strings.SplitN(art.Wordmark, "\n", 2)[0]; !strings.Contains(content, topLine) {
		t.Error("the grouped top-level menu should keep the block wordmark")
	}
}

func lockedMenu() Model {
	summary := Summary{Name: "Sebastián", XP: 0, Streak: 0, Learned: 0, Total: 20, ReadingLocked: true}
	return New(ui.NewTheme(true), i18n.ES, summary, "test")
}

func TestLockedReadingItemDoesNotNavigate(t *testing.T) {
	m := openLeaf(t, lockedMenu(), nav.Flashcards)

	next, cmd := m.choose()
	if cmd != nil {
		t.Fatalf("opening a locked item should not navigate, got cmd %T", cmd())
	}
	if got := next.(Model).notice; got != i18n.ES.ReadingLocked {
		t.Errorf("notice = %q, want the reading-locked hint", got)
	}
}

func TestLockedMenuViewMarksAndExplains(t *testing.T) {
	m := openLeaf(t, lockedMenu(), nav.Quiz)
	m, _ = func() (Model, tea.Cmd) { tm, cmd := m.choose(); return tm.(Model), cmd }()

	content := m.View().Content
	if !strings.Contains(content, lockGlyph) {
		t.Errorf("locked submenu view missing the lock glyph %q", lockGlyph)
	}
	if !strings.Contains(content, i18n.ES.ReadingLocked) {
		t.Errorf("locked submenu view missing the reading-locked notice")
	}
}

func TestLockedItemReplacesIconWithLock(t *testing.T) {
	m := openLeaf(t, lockedMenu(), nav.Flashcards) // Flashcards lives under Aprender
	rendered := m.menuItems()
	if got := strings.Count(rendered, lockGlyph); got != 1 {
		t.Errorf("lock glyphs in the Aprender submenu = %d, want 1 (Flashcards)", got)
	}
	if strings.Contains(rendered, "▣") { // the Flashcards icon is replaced by the lock
		t.Error("locked Flashcards should show the lock glyph, not its own icon")
	}
}

func TestUnlockedMenuHasNoLocks(t *testing.T) {
	m := newTestMenu() // ReadingLocked defaults to false
	for _, cat := range m.items {
		for _, child := range cat.children {
			if child.locked {
				t.Fatalf("item %q is locked when reading is unlocked", child.label)
			}
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
