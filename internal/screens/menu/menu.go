package menu

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

// Summary is the progress data shown in the menu header (JLPT badge and streak).
type Summary struct {
	Level     string
	NextLevel string
	Percent   int
	Streak    int
	Learned   int
	Total     int
}

type item struct {
	icon  string
	label string
}

// Model is the main menu screen.
type Model struct {
	theme   ui.Theme
	msgs    i18n.Messages
	summary Summary
	version string

	items  []item
	cursor int
	status string

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
		items: []item{
			{"かな", msgs.ItemKana},
			{"🎴", msgs.ItemFlashcards},
			{"✓", msgs.ItemQuiz},
			{"📊", msgs.ItemStats},
			{"⏻", msgs.ItemQuit},
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
			m.status = ""
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			m.status = ""
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", " ":
			return m.choose()
		}
	}
	return m, nil
}

func (m Model) choose() (tea.Model, tea.Cmd) {
	// The Quit item is always last.
	if m.cursor == len(m.items)-1 {
		return m, tea.Quit
	}
	// Study screens are wired in a later step.
	m.status = m.msgs.ComingSoon
	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var b strings.Builder

	header := fmt.Sprintf("%s  %s", m.msgs.AppName, m.msgs.Tagline)
	b.WriteString(m.theme.Title.Render(header))
	b.WriteString("\n")
	b.WriteString(m.theme.Subtle.Render(m.badge()))
	b.WriteString("\n\n")
	b.WriteString(m.msgs.MenuPrompt)
	b.WriteString("\n\n")

	for i, it := range m.items {
		line := fmt.Sprintf("%s  %s", it.icon, it.label)
		if i == m.cursor {
			b.WriteString(m.theme.Selected.Render("▸ " + line))
		} else {
			b.WriteString(m.theme.Normal.Render("  " + line))
		}
		b.WriteString("\n")
	}

	if m.status != "" {
		b.WriteString("\n")
		b.WriteString(m.theme.Accent.Render(m.status))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.theme.Help.Render(m.msgs.MenuHelp))

	content := m.theme.Box.Render(b.String())
	view := tea.NewView(ui.Center(m.width, m.height, content))
	view.AltScreen = true
	return view
}

// badge renders the JLPT progress line and study streak.
func (m Model) badge() string {
	bar := ui.ProgressBar(m.summary.Percent, 10)
	level := fmt.Sprintf("%s: %s  %s %d%% %s %s",
		m.msgs.LevelLabel, m.summary.Level, bar, m.summary.Percent,
		m.msgs.TowardLabel, m.summary.NextLevel)
	streak := fmt.Sprintf("🔥 %s: %d %s · %d %s",
		m.msgs.StreakLabel, m.summary.Streak, m.msgs.DaysSuffix,
		m.summary.Learned, m.msgs.LearnedSuffix)
	return lipgloss.JoinVertical(lipgloss.Left, level, streak)
}
