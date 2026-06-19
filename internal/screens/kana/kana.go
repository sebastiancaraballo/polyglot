package kana

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/study"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

const optionCount = 4

// Model is the kana trainer screen: it shows a kana character and asks the
// learner to choose its romaji reading from four options.
type Model struct {
	theme ui.Theme
	msgs  i18n.Messages
	rng   *rand.Rand

	items []model.KanaItem // original set, for restarting
	deck  []model.KanaItem // shuffled session order
	pool  []string         // all romaji, for distractors

	index        int
	options      []string
	correct      int
	selected     int
	answered     bool
	correctCount int

	width, height int
}

// New builds a kana trainer for the given items.
func New(theme ui.Theme, msgs i18n.Messages, items []model.KanaItem) Model {
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // not security-sensitive
	m := Model{theme: theme, msgs: msgs, rng: rng, items: items}
	m.pool = make([]string, 0, len(items))
	for _, it := range items {
		m.pool = append(m.pool, it.Romaji)
	}
	m.deck = append([]model.KanaItem(nil), items...)
	m.rng.Shuffle(len(m.deck), func(i, j int) { m.deck[i], m.deck[j] = m.deck[j], m.deck[i] })
	return m.setQuestion()
}

func (m Model) setQuestion() Model {
	if m.index >= len(m.deck) {
		return m
	}
	m.options, m.correct = study.Options(m.rng, m.deck[m.index].Romaji, m.pool, optionCount)
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
		if isConfirm(msg) {
			return New(m.theme, m.msgs, m.items), nil // restart
		}
	case m.answered:
		if isConfirm(msg) {
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
	case "enter", " ":
		m = m.reveal()
	}
	return m
}

func (m Model) reveal() Model {
	m.answered = true
	if m.selected == m.correct {
		m.correctCount++
	}
	return m
}

func isConfirm(msg tea.KeyPressMsg) bool {
	return msg.String() == "enter" || msg.String() == " "
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var content string
	if m.finished() {
		content = m.summaryView()
	} else {
		content = m.questionView()
	}
	view := tea.NewView(ui.Center(m.width, m.height, m.theme.Box.Render(content)))
	view.AltScreen = true
	return view
}

func (m Model) questionView() string {
	header := fmt.Sprintf("%s  %d/%d", m.theme.Title.Render(m.msgs.KanaTitle), m.index+1, len(m.deck))

	var lower strings.Builder
	lower.WriteString(m.msgs.KanaPrompt)
	lower.WriteString("\n\n")
	for i, opt := range m.options {
		line := fmt.Sprintf(" %d) %s", i+1, opt)
		switch {
		case m.answered && i == m.correct:
			lower.WriteString(m.theme.Success.Render("✓" + line))
		case m.answered && i == m.selected:
			lower.WriteString(m.theme.Error.Render("✗" + line))
		case i == m.selected:
			lower.WriteString(m.theme.Selected.Render("▸" + line))
		default:
			lower.WriteString(m.theme.Normal.Render(" " + line))
		}
		lower.WriteString("\n")
	}
	lower.WriteString("\n")
	if m.answered {
		lower.WriteString(m.theme.Help.Render(m.msgs.ContinueHelp))
	} else {
		lower.WriteString(m.theme.Help.Render(m.msgs.ChoiceHelp))
	}
	lowerStr := lower.String()

	// Render the kana as a large, prominent tile centered over the content.
	tile := m.bigKana(m.deck[m.index].Char)
	width := lipgloss.Width(lowerStr)
	if w := lipgloss.Width(tile); w > width {
		width = w
	}
	tile = lipgloss.PlaceHorizontal(width, lipgloss.Center, tile)

	return lipgloss.JoinVertical(lipgloss.Left, header, "", tile, "", lowerStr)
}

// bigKana renders a kana character as a large, bordered focal tile. A terminal
// app cannot change the font size (that is the terminal's job), so prominence is
// achieved with a wide border and generous padding around the glyph.
func (m Model) bigKana(char string) string {
	return m.theme.Accent.
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		Padding(2, 6).
		Align(lipgloss.Center).
		Render(char)
}

func (m Model) summaryView() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render(m.msgs.SessionDone))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "%s: %d/%d\n\n", m.msgs.ScoreLabel, m.correctCount, len(m.deck))
	b.WriteString(m.theme.Help.Render(m.msgs.RestartHelp))
	return b.String()
}
