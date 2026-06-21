package menu

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

// Summary is the progress data shown in the menu header (XP, streak, and the
// number of words learned).
type Summary struct {
	Name    string
	XP      int
	Streak  int
	Learned int
	Total   int
}

type item struct {
	icon   string
	label  string
	screen nav.Screen
	quit   bool
}

// Model is the main menu screen.
type Model struct {
	theme   ui.Theme
	msgs    i18n.Messages
	summary Summary
	version string

	items  []item
	cursor int

	width  int
	height int
}

// New builds a menu model.
func New(theme ui.Theme, msgs i18n.Messages, summary Summary, version string) Model {
	return Model{
		theme:   theme,
		msgs:    msgs,
		summary: summary,
		version: version,
		cursor:  1,
		items: []item{
			{"◇", msgs.ItemKana, nav.Kana, false},
			{"▣", msgs.ItemFlashcards, nav.Flashcards, false},
			{"✓", msgs.ItemQuiz, nav.Quiz, false},
			{"▤", msgs.ItemStats, nav.Stats, false},
			{"⚙", msgs.ItemSettings, nav.Settings, false},
			{"⏻", msgs.ItemQuit, nav.Menu, true},
		},
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items) {
				m.cursor++
			}
		}
		if ui.IsConfirmKey(msg) {
			return m.choose()
		}
	}
	return m, nil
}

func (m Model) choose() (tea.Model, tea.Cmd) {
	if m.cursor == 0 {
		return m, nav.GoTo(nav.Profiles)
	}
	it := m.items[m.cursor-1]
	if it.quit {
		return m, tea.Quit
	}
	return m, nav.GoTo(it.screen)
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var b strings.Builder

	header := fmt.Sprintf("%s  %s", m.msgs.AppName, m.msgs.Tagline)
	b.WriteString(m.theme.Title.Render(header))
	b.WriteString("\n")
	b.WriteString(m.profileHeader())
	b.WriteString("\n")
	b.WriteString(m.theme.Subtle.Render(m.badge()))
	b.WriteString("\n\n")
	b.WriteString(m.msgs.MenuPrompt)
	b.WriteString("\n\n")

	for i, it := range m.items {
		line := fmt.Sprintf("%s  %s", it.icon, it.label)
		if i+1 == m.cursor {
			b.WriteString(m.theme.Selected.Render("▸ " + line))
		} else {
			b.WriteString(m.theme.Normal.Render("  " + line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.theme.Help.Render(m.msgs.MenuHelp))

	view := tea.NewView(ui.Frame(m.theme, m.width, m.height, b.String()))
	view.AltScreen = true
	return view
}

// badge renders the experience-point total and study streak.
func (m Model) badge() string {
	xp := fmt.Sprintf("★ %s: %d", m.msgs.XPLabel, m.summary.XP)
	streak := fmt.Sprintf("▲ %s: %d %s · %d %s",
		m.msgs.StreakLabel, m.summary.Streak, m.msgs.DaysSuffix,
		m.summary.Learned, m.msgs.LearnedSuffix)
	return lipgloss.JoinVertical(lipgloss.Left, xp, streak)
}

func (m Model) profileHeader() string {
	name := m.summary.Name
	if name == "" {
		name = m.msgs.ProfileNamePlaceholder
	}
	marker := "  "
	if m.cursor == 0 {
		marker = "▸ "
	}
	content := fmt.Sprintf("%s⇄ %s · %s", marker, name, m.msgs.SwitchProfile)
	if m.cursor == 0 {
		return m.theme.Selected.Render(content)
	}
	return m.theme.Normal.Render(content)
}
