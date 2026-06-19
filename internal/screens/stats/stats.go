package stats

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/content"
	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

// Deps are the dependencies the stats screen needs.
type Deps struct {
	Theme     ui.Theme
	Msgs      i18n.Messages
	Store     storage.Storage
	ProfileID int64
	Course    *content.Course
}

// Model is the statistics screen: JLPT progress, streak, and kana totals.
type Model struct {
	deps Deps

	streak  int
	best    int
	learned int
	total   int
	percent int
	hira    int
	kata    int
	err     error

	width, height int
}

// New builds the stats screen, reading aggregate progress from storage.
func New(deps Deps) Model {
	m := Model{deps: deps}
	ctx := context.Background()

	stats, err := deps.Store.GetStats(ctx, deps.ProfileID)
	if err != nil {
		m.err = err
		return m
	}
	learned, err := deps.Store.CountLearnedCards(ctx, deps.ProfileID)
	if err != nil {
		m.err = err
		return m
	}

	m.streak = stats.Streak
	m.best = stats.BestStreak
	m.learned = learned
	for _, lesson := range deps.Course.Lessons {
		m.total += len(lesson.Cards)
	}
	if m.total > 0 {
		m.percent = m.learned * 100 / m.total
	}
	for _, k := range deps.Course.Kana {
		switch k.Type {
		case model.Hiragana:
			m.hira++
		case model.Katakana:
			m.kata++
		}
	}
	return m
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
		case "ctrl+c":
			return m, tea.Quit
		case "esc", "q", "enter":
			return m, nav.Back()
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.StatsTitle))
	b.WriteString("\n\n")

	bar := ui.ProgressBar(m.percent, 10)
	fmt.Fprintf(&b, "%s %s  %s %d%%  (%d/%d)\n",
		m.deps.Msgs.LevelLabel, string(model.N5), bar, m.percent, m.learned, m.total)
	fmt.Fprintf(&b, "🔥 %s: %d %s  (%s: %d)\n",
		m.deps.Msgs.StreakLabel, m.streak, m.deps.Msgs.DaysSuffix, m.deps.Msgs.BestLabel, m.best)
	fmt.Fprintf(&b, "%s: %d   %s: %d\n",
		m.deps.Msgs.HiraganaLabel, m.hira, m.deps.Msgs.KatakanaLabel, m.kata)

	b.WriteString("\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.BackHelp))

	view := tea.NewView(ui.Center(m.width, m.height, t.Box.Render(b.String())))
	view.AltScreen = true
	return view
}
