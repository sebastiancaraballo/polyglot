package quiz

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

const (
	optionCount   = 4
	questionLimit = 10
)

// Deps are the dependencies a quiz needs.
type Deps struct {
	Theme     ui.Theme
	Msgs      i18n.Messages
	Store     storage.Storage
	ProfileID int64
	Cards     []model.Card
}

// Model is the multiple-choice vocabulary quiz screen.
type Model struct {
	deps Deps
	rng  *rand.Rand

	deck    []model.Card
	pool    []string
	index   int
	options []string
	correct int

	selected      int
	answered      bool
	score         int
	wrong         []string
	streakApplied bool
	err           error

	width, height int
}

// New builds a quiz from the given dependencies.
func New(deps Deps) Model {
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // not security-sensitive
	m := Model{deps: deps, rng: rng}

	m.pool = make([]string, 0, len(deps.Cards))
	for _, c := range deps.Cards {
		m.pool = append(m.pool, c.JP)
	}
	m.deck = append([]model.Card(nil), deps.Cards...)
	m.rng.Shuffle(len(m.deck), func(i, j int) { m.deck[i], m.deck[j] = m.deck[j], m.deck[i] })
	if len(m.deck) > questionLimit {
		m.deck = m.deck[:questionLimit]
	}
	return m.setQuestion()
}

func (m Model) setQuestion() Model {
	if m.index >= len(m.deck) {
		return m
	}
	m.options, m.correct = study.Options(m.rng, m.deck[m.index].JP, m.pool, optionCount)
	m.selected = 0
	m.answered = false
	return m
}

func (m Model) finished() bool { return m.index >= len(m.deck) }

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

	switch {
	case m.finished():
		if confirm(msg) {
			return New(m.deps), nil
		}
	case m.answered:
		if confirm(msg) {
			m.index++
			if !m.finished() {
				m = m.setQuestion()
			}
		}
	default:
		m = m.answerKey(msg)
	}
	return m, nil
}

func (m Model) answerKey(msg tea.KeyPressMsg) Model {
	switch msg.String() {
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		if m.selected < len(m.options)-1 {
			m.selected++
		}
	case "1", "2", "3", "4":
		i := int(msg.String()[0] - '1')
		if i < len(m.options) {
			m.selected = i
			m = m.reveal()
		}
	}
	if ui.IsConfirmKey(msg) {
		m = m.reveal()
	}
	return m
}

func (m Model) reveal() Model {
	m.answered = true
	card := m.deck[m.index]
	correct := m.selected == m.correct
	if correct {
		m.score++
	} else {
		m.wrong = append(m.wrong, card.Source)
	}
	if err := m.persist(card, correct); err != nil {
		m.err = err
	}
	return m
}

// persist records the review as a spaced-repetition grade and updates the streak.
func (m *Model) persist(card model.Card, correct bool) error {
	ctx := context.Background()
	state, err := m.deps.Store.GetCardState(ctx, m.deps.ProfileID, card.ID)
	if err != nil {
		state = srs.NewCard(card.ID)
	}
	grade := srs.Again
	if correct {
		grade = srs.Good
	}
	now := time.Now()
	state = srs.Review(state, grade, now)
	if err := m.deps.Store.SaveCardState(ctx, m.deps.ProfileID, state); err != nil {
		return err
	}
	if !m.streakApplied {
		stats, err := m.deps.Store.GetStats(ctx, m.deps.ProfileID)
		if err != nil {
			return err
		}
		if err := m.deps.Store.SaveStats(ctx, m.deps.ProfileID, study.UpdateStreak(stats, now)); err != nil {
			return err
		}
		m.streakApplied = true
	}
	return nil
}

func confirm(msg tea.KeyPressMsg) bool {
	return ui.IsConfirmKey(msg)
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var content string
	switch {
	case m.finished():
		content = m.summaryView()
	default:
		content = m.questionView()
	}
	view := tea.NewView(ui.Frame(m.deps.Theme, m.width, m.height, content))
	view.AltScreen = true
	return view
}

func (m Model) questionView() string {
	t := m.deps.Theme
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %d/%d\n\n", t.Title.Render(m.deps.Msgs.QuizTitle), m.index+1, len(m.deck))
	fmt.Fprintf(&b, m.deps.Msgs.QuizQuestionFmt, m.deck[m.index].Source)
	b.WriteString("\n\n")

	for i, opt := range m.options {
		line := fmt.Sprintf(" %d) %s", i+1, opt)
		switch {
		case m.answered && i == m.correct:
			b.WriteString(t.Success.Render("✓" + line))
		case m.answered && i == m.selected:
			b.WriteString(t.Error.Render("✗" + line))
		case i == m.selected:
			b.WriteString(t.Selected.Render("▸" + line))
		default:
			b.WriteString(t.Normal.Render(" " + line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	fmt.Fprintf(&b, "%s: %d/%d\n", m.deps.Msgs.ScoreLabel, m.score, m.index+boolToInt(m.answered))
	if m.answered {
		b.WriteString(t.Help.Render(m.deps.Msgs.ContinueHelp))
	} else {
		b.WriteString(t.Help.Render(m.deps.Msgs.ChoiceHelp))
	}
	return b.String()
}

func (m Model) summaryView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.SessionDone))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "%s: %d/%d\n", m.deps.Msgs.ScoreLabel, m.score, len(m.deck))
	if len(m.wrong) > 0 {
		fmt.Fprintf(&b, "%s: %s\n", m.deps.Msgs.ReviewLabel, strings.Join(m.wrong, ", "))
	}
	b.WriteString("\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.RestartHelp))
	return b.String()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
