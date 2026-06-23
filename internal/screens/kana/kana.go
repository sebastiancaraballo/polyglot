package kana

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/study"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

const optionCount = 4

// Deps are the dependencies the kana trainer needs.
type Deps struct {
	Theme     ui.Theme
	Msgs      i18n.Messages
	Store     storage.Storage
	ProfileID int64
	Items     []model.KanaItem
}

// group is a selectable practice set in the pre-session picker.
type group struct {
	label string
	match func(model.KanaItem) bool
}

// Model is the kana trainer screen. It first shows a group picker, then quizzes
// the chosen kana: it shows a character and asks the learner to choose its romaji
// reading from four options.
type Model struct {
	deps Deps
	rng  *rand.Rand

	groups      []group // selectable practice sets
	picking     bool    // true while showing the group picker
	groupCursor int

	deck []model.KanaItem // shuffled session order
	pool []string         // romaji of the selected group, for distractors

	index        int
	options      []string
	correct      int
	selected     int
	answered     bool
	correctCount int
	err          error

	width, height int
}

// New builds a kana trainer that opens on the group picker.
func New(deps Deps) Model {
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // not security-sensitive
	return Model{deps: deps, rng: rng, picking: true, groups: buildGroups(deps.Msgs)}
}

// buildGroups returns the practice sets: everything, then each syllabary split by
// category, matching the kana chart's pages.
func buildGroups(msg i18n.Messages) []group {
	cat := func(typ model.KanaType, cats []model.KanaCategory, label string) group {
		syllabary := msg.HiraganaLabel
		if typ == model.Katakana {
			syllabary = msg.KatakanaLabel
		}
		return group{
			label: fmt.Sprintf("%s · %s", syllabary, label),
			match: func(it model.KanaItem) bool {
				if it.Type != typ {
					return false
				}
				for _, c := range cats {
					if it.Category == c {
						return true
					}
				}
				return false
			},
		}
	}
	return []group{
		{label: msg.KanaGroupAll, match: func(model.KanaItem) bool { return true }},
		cat(model.Hiragana, []model.KanaCategory{model.Base}, msg.KanaBasic),
		cat(model.Hiragana, []model.KanaCategory{model.Dakuten, model.Handakuten}, msg.KanaVoiced),
		cat(model.Hiragana, []model.KanaCategory{model.Combo}, msg.KanaCombo),
		cat(model.Katakana, []model.KanaCategory{model.Base}, msg.KanaBasic),
		cat(model.Katakana, []model.KanaCategory{model.Dakuten, model.Handakuten}, msg.KanaVoiced),
		cat(model.Katakana, []model.KanaCategory{model.Combo}, msg.KanaCombo),
	}
}

// startSession filters the chosen group into a fresh shuffled deck and begins.
func (m Model) startSession() Model {
	g := m.groups[m.groupCursor]
	var items []model.KanaItem
	for _, it := range m.deps.Items {
		if g.match(it) {
			items = append(items, it)
		}
	}
	m.pool = make([]string, 0, len(items))
	for _, it := range items {
		m.pool = append(m.pool, it.Romaji)
	}
	m.deck = items
	m.rng.Shuffle(len(m.deck), func(i, j int) { m.deck[i], m.deck[j] = m.deck[j], m.deck[i] })
	m.index = 0
	m.correctCount = 0
	m.picking = false
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

	if m.picking {
		return m.handlePick(msg)
	}

	switch {
	case m.finished():
		if isConfirm(msg) {
			m.picking = true // back to the group picker
			return m, nil
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

func (m Model) handlePick(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.groupCursor > 0 {
			m.groupCursor--
		}
	case "down", "j":
		if m.groupCursor < len(m.groups)-1 {
			m.groupCursor++
		}
	}
	if ui.IsConfirmKey(msg) {
		m = m.startSession()
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
	correct := m.selected == m.correct
	if correct {
		m.correctCount++
	}
	if err := m.deps.Store.AddXP(context.Background(), m.deps.ProfileID, study.XPForAnswer(correct)); err != nil {
		m.err = err
	}
	return m
}

func isConfirm(msg tea.KeyPressMsg) bool {
	return ui.IsConfirmKey(msg)
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var content string
	switch {
	case m.picking:
		content = m.pickerView()
	case m.finished():
		content = m.summaryView()
	default:
		content = m.questionView()
	}
	view := tea.NewView(ui.Frame(m.deps.Theme, m.width, m.height, content))
	view.AltScreen = true
	return view
}

func (m Model) pickerView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.KanaTitle))
	b.WriteString("\n\n")
	for i, g := range m.groups {
		if i == m.groupCursor {
			b.WriteString(t.Selected.Render("▸ " + g.label))
		} else {
			b.WriteString(t.Normal.Render("  " + g.label))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.KanaPickHelp))
	return b.String()
}

func (m Model) questionView() string {
	t := m.deps.Theme
	header := fmt.Sprintf("%s  %d/%d", t.Title.Render(m.deps.Msgs.KanaTitle), m.index+1, len(m.deck))

	var lower strings.Builder
	lower.WriteString(m.deps.Msgs.KanaPrompt)
	lower.WriteString("\n\n")
	for i, opt := range m.options {
		line := fmt.Sprintf(" %d) %s", i+1, opt)
		switch {
		case m.answered && i == m.correct:
			lower.WriteString(t.Success.Render("✓" + line))
		case m.answered && i == m.selected:
			lower.WriteString(t.Error.Render("✗" + line))
		case i == m.selected:
			lower.WriteString(t.Selected.Render("▸" + line))
		default:
			lower.WriteString(t.Normal.Render(" " + line))
		}
		lower.WriteString("\n")
	}
	lower.WriteString("\n")
	if m.answered {
		lower.WriteString(t.Help.Render(m.deps.Msgs.ContinueHelp))
	} else {
		lower.WriteString(t.Help.Render(m.deps.Msgs.ChoiceHelp))
	}
	lowerStr := lower.String()

	// Render the kana as a large, prominent tile centered within the frame, not
	// within the dynamic help/options text below it.
	tile := m.bigKana(m.deck[m.index].Char)
	width := ui.FrameContentWidth(t, m.width)
	if tileWidth := lipgloss.Width(tile); tileWidth > width {
		width = tileWidth
	}
	tile = lipgloss.PlaceHorizontal(width, lipgloss.Center, tile)

	return lipgloss.JoinVertical(lipgloss.Left, header, "", tile, "", lowerStr)
}

// bigKana renders a kana character as a large, bordered focal tile. A terminal
// app cannot change the font size (that is the terminal's job), so prominence is
// achieved with a wide border and generous padding around the glyph.
func (m Model) bigKana(char string) string {
	return m.deps.Theme.Accent.
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		Padding(2, 6).
		Align(lipgloss.Center).
		Render(char)
}

func (m Model) summaryView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.SessionDone))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "%s: %d/%d\n\n", m.deps.Msgs.ScoreLabel, m.correctCount, len(m.deck))
	b.WriteString(t.Help.Render(m.deps.Msgs.RestartHelp))
	return b.String()
}
