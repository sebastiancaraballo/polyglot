package flashcards

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/review"
	"github.com/sebastiancaraballo/polyglot/internal/srs"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/study"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

const sessionLimit = 20

// Deps are the dependencies a review session needs.
type Deps struct {
	Theme      ui.Theme
	Msgs       i18n.Messages
	Store      storage.Storage
	ProfileID  int64
	Items      []review.Item
	ShowRomaji bool
	Title      string // screen title (vocabulary-only vs. cross-curriculum review)
}

// gradeKeys maps number keys to spaced-repetition grades.
var gradeKeys = map[string]srs.Grade{
	"1": srs.Again,
	"2": srs.Hard,
	"3": srs.Good,
	"4": srs.Easy,
}

// Model is the flashcard-style review screen. It presents the items currently due
// — built by the cross-curriculum review queue — one at a time: prompt, then the
// revealed answer, then a spaced-repetition grade.
type Model struct {
	deps  Deps
	queue []review.Scheduled

	index         int
	revealed      bool
	reviewed      int
	streakApplied bool
	err           error

	width, height int
}

// New builds a review session containing the items currently due.
func New(deps Deps) Model {
	m := Model{deps: deps}
	q, err := review.BuildQueue(context.Background(), deps.Store, deps.ProfileID, deps.Items, time.Now(), sessionLimit)
	if err != nil {
		m.err = err
	}
	m.queue = q
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
	sched := m.queue[m.index]
	state := srs.Review(sched.State, grade, time.Now())
	if err := m.persist(state, grade); err != nil {
		m.err = err
	}
	m.reviewed++
	m.index++
	m.revealed = false
	return m
}

func (m *Model) persist(state model.CardState, grade srs.Grade) error {
	ctx := context.Background()
	if err := m.deps.Store.SaveCardState(ctx, m.deps.ProfileID, state); err != nil {
		return err
	}
	if err := m.deps.Store.AddXP(ctx, m.deps.ProfileID, study.XPForGrade(grade)); err != nil {
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
	view := tea.NewView(ui.Frame(m.deps.Theme, m.width, m.height, content))
	view.AltScreen = true
	return view
}

func (m Model) title() string {
	if m.deps.Title != "" {
		return m.deps.Title
	}
	return m.deps.Msgs.FlashTitle
}

func (m Model) cardView() string {
	t := m.deps.Theme
	item := m.queue[m.index].Item
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %d/%d\n\n", t.Title.Render(m.title()), m.index+1, len(m.queue))
	b.WriteString(t.Accent.Render(item.Prompt))
	b.WriteString("\n\n")

	if !m.revealed {
		b.WriteString(t.Help.Render(m.deps.Msgs.RevealHelp))
		return b.String()
	}

	b.WriteString(t.Title.Render(item.Answer))
	b.WriteString("\n")
	if m.deps.ShowRomaji && item.Secondary != "" {
		b.WriteString(t.Subtle.Render(item.Secondary))
		b.WriteString("\n")
	}
	if item.Notes != "" {
		b.WriteString(t.Subtle.Render(ui.WrapText(item.Notes, m.noteWidth())))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.deps.Msgs.GradePrompt)
	b.WriteString("\n")
	b.WriteString(m.gradeOptions(m.queue[m.index].State))
	b.WriteString("\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.BackHelp))
	return b.String()
}

func (m Model) noteWidth() int {
	if m.width <= 0 {
		return 64
	}
	width := m.width - 12
	if width < 24 {
		return 24
	}
	if width > 64 {
		return 64
	}
	return width
}

func (m Model) gradeOptions(state model.CardState) string {
	now := time.Now()
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
	return strings.Join(parts, "\n")
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
