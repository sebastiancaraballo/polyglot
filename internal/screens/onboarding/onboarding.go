package onboarding

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

type step int

const (
	stepWelcome step = iota
	stepExercise
	stepDone
)

// Deps are the dependencies the onboarding flow needs.
type Deps struct {
	Theme     ui.Theme
	Msgs      i18n.Messages
	Store     storage.Storage
	ProfileID int64
}

// Model is the first-run onboarding flow: it teaches the controls with a guided
// sample exercise and marks the profile as onboarded on completion.
type Model struct {
	deps Deps

	step     step
	selected int
	answered bool
	correct  bool
	err      error

	width, height int
}

// New builds the onboarding flow.
func New(deps Deps) Model { return Model{deps: deps} }

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

	switch m.step {
	case stepWelcome:
		if confirm(msg) {
			m.step = stepExercise
		}
	case stepExercise:
		return m.handleExercise(msg)
	case stepDone:
		if confirm(msg) {
			return m.finish()
		}
	}
	return m, nil
}

func (m Model) handleExercise(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Once answered correctly, the next confirmation advances to the final step.
	if m.answered && m.correct {
		if confirm(msg) {
			m.step = stepDone
		}
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		if m.selected < len(m.deps.Msgs.SampleOptions)-1 {
			m.selected++
		}
	case "1", "2", "3", "4":
		i := int(msg.String()[0] - '1')
		if i < len(m.deps.Msgs.SampleOptions) {
			m.selected = i
			m = m.answer()
		}
	}
	if ui.IsConfirmKey(msg) {
		m = m.answer()
	}
	return m, nil
}

func (m Model) answer() Model {
	m.answered = true
	m.correct = m.selected == m.deps.Msgs.SampleCorrect
	return m
}

func (m Model) finish() (tea.Model, tea.Cmd) {
	if err := m.deps.Store.SetOnboarded(context.Background(), m.deps.ProfileID); err != nil {
		m.err = err
	}
	return m, nav.Back()
}

func confirm(msg tea.KeyPressMsg) bool {
	return ui.IsConfirmKey(msg)
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var content string
	switch m.step {
	case stepExercise:
		content = m.exerciseView()
	case stepDone:
		content = m.doneView()
	default:
		content = m.welcomeView()
	}
	view := tea.NewView(ui.Frame(m.deps.Theme, m.width, m.height, content))
	view.AltScreen = true
	return view
}

func (m Model) welcomeView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.WelcomeTitle))
	b.WriteString("\n\n")
	b.WriteString(m.deps.Msgs.WelcomeIntro)
	b.WriteString("\n\n")
	b.WriteString(t.Accent.Render(m.deps.Msgs.ControlsTitle))
	b.WriteString("\n")
	for _, k := range m.deps.Msgs.ControlsKeys {
		b.WriteString("  " + k + "\n")
	}
	b.WriteString("\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.WelcomeNext))
	return b.String()
}

func (m Model) exerciseView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.PracticeTitle))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "%s  (%s)\n", t.Accent.Render(m.deps.Msgs.SampleWord), m.deps.Msgs.SampleRomaji)
	b.WriteString(m.deps.Msgs.SamplePrompt)
	b.WriteString("\n\n")

	for i, opt := range m.deps.Msgs.SampleOptions {
		line := fmt.Sprintf(" %d) %s", i+1, opt)
		switch {
		case i == m.deps.Msgs.SampleCorrect && (m.answered || i == m.selected):
			b.WriteString(t.Success.Render("✓" + line + "  " + m.deps.Msgs.SampleHint))
		case i == m.selected:
			b.WriteString(t.Selected.Render("▸" + line))
		default:
			b.WriteString(t.Normal.Render(" " + line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	switch {
	case m.answered && m.correct:
		b.WriteString(t.Success.Render(m.deps.Msgs.PracticeCorrect))
		b.WriteString("\n")
		b.WriteString(t.Help.Render(m.deps.Msgs.PracticeNext))
	case m.answered:
		b.WriteString(t.Error.Render(m.deps.Msgs.PracticeRetry))
	default:
		b.WriteString(t.Help.Render(m.deps.Msgs.ChoiceHelp))
	}
	return b.String()
}

func (m Model) doneView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.DoneTitle))
	b.WriteString("\n\n")
	b.WriteString(m.deps.Msgs.DoneRecommend)
	b.WriteString("\n\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.DoneNext))
	return b.String()
}
