package flashcards

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/srs"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/study"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

const sessionLimit = 20

// Deps are the dependencies a flashcards session needs.
type Deps struct {
	Theme     ui.Theme
	Msgs      i18n.Messages
	Store     storage.Storage
	ProfileID int64
	Cards     []model.Card
}

// gradeKeys maps number keys to spaced-repetition grades.
var gradeKeys = map[string]srs.Grade{
	"1": srs.Again,
	"2": srs.Hard,
	"3": srs.Good,
	"4": srs.Easy,
}

// Model is the spaced-repetition flashcard study screen.
type Model struct {
	deps   Deps
	queue  []model.Card
	states map[string]model.CardState

	index         int
	revealed      bool
	reviewed      int
	streakApplied bool
	err           error

	width, height int
}

// New builds a flashcards session containing the cards currently due.
func New(deps Deps) Model {
	m := Model{deps: deps, states: make(map[string]model.CardState)}
	now := time.Now()
	ctx := context.Background()

	for _, card := range deps.Cards {
		state, err := deps.Store.GetCardState(ctx, deps.ProfileID, card.ID)
		if err != nil {
			state = srs.NewCard(card.ID)
		}
		if srs.IsDue(state, now) {
			m.queue = append(m.queue, card)
			m.states[card.ID] = state
		}
	}

	rng := rand.New(rand.NewSource(now.UnixNano())) //nolint:gosec // not security-sensitive
	rng.Shuffle(len(m.queue), func(i, j int) { m.queue[i], m.queue[j] = m.queue[j], m.queue[i] })
	if len(m.queue) > sessionLimit {
		m.queue = m.queue[:sessionLimit]
	}
	return m
}

func (m Model) finished() bool { return m.index >= len(m.queue) }

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		return m, nav.Back()
	}
	if m.finished() {
		return m, nil
	}
	if !m.revealed {
		if ui.IsConfirmKey(msg) {
			m.revealed = true
		}
		return m, nil
	}
	if grade, ok := gradeKeys[msg.String()]; ok {
		m = m.grade(grade)
	}
	return m, nil
}

func (m Model) grade(grade srs.Grade) Model {
	card := m.queue[m.index]
	state := srs.Review(m.states[card.ID], grade, time.Now())
	if err := m.persist(state); err != nil {
		m.err = err
	}
	m.reviewed++
	m.index++
	m.revealed = false
	return m
}

func (m *Model) persist(state model.CardState) error {
	ctx := context.Background()
	if err := m.deps.Store.SaveCardState(ctx, m.deps.ProfileID, state); err != nil {
		return err
	}
	if !m.streakApplied {
		stats, err := m.deps.Store.GetStats(ctx, m.deps.ProfileID)
		if err != nil {
			return err
		}
		if err := m.deps.Store.SaveStats(ctx, m.deps.ProfileID, study.UpdateStreak(stats, time.Now())); err != nil {
			return err
		}
		m.streakApplied = true
	}
	return nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var content string
	switch {
	case len(m.queue) == 0:
		content = m.deps.Theme.Title.Render(m.deps.Msgs.NothingDue)
	case m.finished():
		content = m.summaryView()
	default:
		content = m.cardView()
	}
	view := tea.NewView(ui.Center(m.width, m.height, m.deps.Theme.Box.Render(content)))
	view.AltScreen = true
	return view
}

func (m Model) cardView() string {
	t := m.deps.Theme
	card := m.queue[m.index]
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %d/%d\n\n", t.Title.Render(m.deps.Msgs.FlashTitle), m.index+1, len(m.queue))
	b.WriteString(t.Accent.Render(card.Source))
	b.WriteString("\n\n")

	if !m.revealed {
		b.WriteString(t.Help.Render(m.deps.Msgs.RevealHelp))
		return b.String()
	}

	b.WriteString(t.Title.Render(card.JP))
	b.WriteString("\n")
	b.WriteString(t.Subtle.Render(card.Romaji))
	b.WriteString("\n")
	if card.Notes != "" {
		b.WriteString(t.Subtle.Render(card.Notes))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.deps.Msgs.GradePrompt)
	b.WriteString("\n")
	b.WriteString(m.gradeOptions(card))
	b.WriteString("\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.BackHelp))
	return b.String()
}

func (m Model) gradeOptions(card model.Card) string {
	now := time.Now()
	state := m.states[card.ID]
	labels := []struct {
		key   string
		label string
		grade srs.Grade
	}{
		{"1", m.deps.Msgs.GradeAgain, srs.Again},
		{"2", m.deps.Msgs.GradeHard, srs.Hard},
		{"3", m.deps.Msgs.GradeGood, srs.Good},
		{"4", m.deps.Msgs.GradeEasy, srs.Easy},
	}
	parts := make([]string, 0, len(labels))
	for _, l := range labels {
		days := srs.PreviewInterval(state, l.grade, now)
		parts = append(parts, fmt.Sprintf("[%s] %s (%s)", l.key, l.label, m.formatInterval(days)))
	}
	return strings.Join(parts, "  ")
}

func (m Model) formatInterval(days int) string {
	if days <= 0 {
		return m.deps.Msgs.Today
	}
	return fmt.Sprintf("%d%s", days, m.deps.Msgs.DayShort)
}

func (m Model) summaryView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.SessionDone))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "%s: %d\n\n", m.deps.Msgs.ReviewedLabel, m.reviewed)
	b.WriteString(t.Help.Render(m.deps.Msgs.BackHelp))
	return b.String()
}
