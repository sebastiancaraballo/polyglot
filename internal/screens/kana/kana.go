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
	label  string
	match  func(model.KanaItem) bool
	locked bool // gated behind a prior syllabary's fluency
}

// Model is the kana trainer screen. It first shows a group picker, then quizzes
// the chosen kana: it shows a character and asks the learner to choose its romaji
// reading from four options. Answers are timed so the trainer can track progress
// toward automaticity (fast, accurate recognition), which drives the Foundations
// decoding gate.
type Model struct {
	deps Deps
	rng  *rand.Rand

	progress map[string]model.KanaProgress // per-kana automaticity, persisted
	gate     study.Gate                    // decoding gate derived from progress

	intro       bool    // true while showing the first-time intro, before the picker
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
	shownAt      time.Time // when the current question was first shown (for timing)
	err          error

	width, height int
}

// New builds a kana trainer. It loads the learner's saved kana progress so the
// picker can show mastery and gate katakana behind hiragana fluency, and opens on
// a one-time intro the first time the learner visits (otherwise on the picker).
func New(deps Deps) Model {
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // not security-sensitive
	m := Model{deps: deps, rng: rng, picking: true, progress: map[string]model.KanaProgress{}}
	if deps.Store != nil {
		if saved, err := deps.Store.GetKanaProgress(context.Background(), deps.ProfileID); err == nil {
			m.progress = saved
		} else {
			m.err = err
		}
		// Show the intro until the learner has seen it once. Read fresh from the
		// store: the screen is rebuilt on each visit, so this reflects a dismissal
		// from an earlier visit without any cross-screen plumbing.
		if prof, err := deps.Store.GetProfile(context.Background(), deps.ProfileID); err == nil {
			m.intro = !prof.KanaOnboarded
		}
	}
	m.picking = !m.intro
	m.gate = study.NewGate(deps.Items, m.progress)
	m.groups = buildGroups(deps.Msgs, m.gate)
	return m
}

// buildGroups returns the practice sets: everything, then each syllabary split by
// category, matching the kana chart's pages. Katakana groups are locked until
// hiragana is fluent, enforcing "hiragana, then katakana".
func buildGroups(msg i18n.Messages, gate study.Gate) []group {
	cat := func(typ model.KanaType, cats []model.KanaCategory, label string) group {
		syllabary := msg.HiraganaLabel
		if typ == model.Katakana {
			syllabary = msg.KatakanaLabel
		}
		return group{
			label:  fmt.Sprintf("%s · %s", syllabary, label),
			locked: typ == model.Katakana && !gate.KatakanaUnlocked(),
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
	m.shownAt = time.Now()
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

	if m.intro {
		if ui.IsConfirmKey(msg) {
			m = m.dismissIntro()
		}
		return m, nil
	}

	if m.picking {
		return m.handlePick(msg)
	}

	switch {
	case m.finished():
		if isConfirm(msg) {
			m.picking = true // back to the group picker
			m.groups = buildGroups(m.deps.Msgs, m.gate)
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

// dismissIntro closes the first-time intro, records that it has been seen so it
// does not reappear, and falls through to the group picker.
func (m Model) dismissIntro() Model {
	m.intro = false
	m.picking = true
	if m.deps.Store != nil {
		if err := m.deps.Store.SetKanaOnboarded(context.Background(), m.deps.ProfileID); err != nil {
			m.err = err
		}
	}
	return m
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
	if ui.IsConfirmKey(msg) && !m.groups[m.groupCursor].locked {
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
	m = m.recordAnswer(correct)
	return m
}

// recordAnswer folds the timed answer into the current kana's automaticity
// progress, persists it, and refreshes the decoding gate.
func (m Model) recordAnswer(correct bool) Model {
	if m.progress == nil {
		m.progress = map[string]model.KanaProgress{}
	}
	var elapsed time.Duration
	if !m.shownAt.IsZero() {
		elapsed = time.Since(m.shownAt)
	}
	char := m.deck[m.index].Char
	p := m.progress[char]
	p.Char = char
	p = study.GradeKana(p, correct, elapsed)
	m.progress[char] = p
	if m.deps.Store != nil {
		if err := m.deps.Store.SaveKanaProgress(context.Background(), m.deps.ProfileID, p); err != nil {
			m.err = err
		}
	}
	m.gate = study.NewGate(m.deps.Items, m.progress)
	return m
}

func isConfirm(msg tea.KeyPressMsg) bool {
	return ui.IsConfirmKey(msg)
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var content string
	switch {
	case m.intro:
		content = m.introView()
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

// introView is the first-time explanation of the kana trainer: the gated path
// (hiragana → katakana → reading) and what mastery means.
func (m Model) introView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.KanaIntroTitle))
	b.WriteString("\n\n")
	body := ui.WrapText(m.deps.Msgs.KanaIntroBody, ui.FrameContentWidth(t, m.width))
	b.WriteString(t.Normal.Render(body))
	b.WriteString("\n\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.KanaIntroHelp))
	return b.String()
}

func (m Model) pickerView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.KanaTitle))
	b.WriteString("\n\n")
	for i, g := range m.groups {
		label := g.label + m.groupSuffix(g)
		switch {
		case g.locked:
			b.WriteString(t.Subtle.Render("⊘ " + label))
		case i == m.groupCursor:
			b.WriteString(t.Selected.Render("▸ " + label))
		default:
			b.WriteString(t.Normal.Render("  " + label))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if m.groups[m.groupCursor].locked {
		hint := fmt.Sprintf(m.deps.Msgs.KanaUnlockHintFmt, m.gate.Hiragana.Mastered, m.gate.Hiragana.Total)
		b.WriteString(t.Subtle.Render(hint))
	} else {
		b.WriteString(t.Help.Render(m.deps.Msgs.KanaPickHelp))
	}
	b.WriteString("\n")
	b.WriteString(t.Subtle.Render(m.deps.Msgs.KanaMasteryNote))
	return b.String()
}

// groupSuffix annotates a picker row with its mastery: a fluent badge when every
// kana in the group is mastered, otherwise a "mastered/total" count.
func (m Model) groupSuffix(g group) string {
	var total, mastered int
	for _, it := range m.deps.Items {
		if !g.match(it) {
			continue
		}
		total++
		if m.progress[it.Char].Mastered {
			mastered++
		}
	}
	if total == 0 {
		return ""
	}
	if mastered >= total {
		return "  ✓ " + m.deps.Msgs.KanaFluent
	}
	return "  " + fmt.Sprintf(m.deps.Msgs.KanaMasteredFmt, mastered, total)
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
	if m.gate.ReadingUnlocked() {
		b.WriteString(t.Success.Render("✓ " + m.deps.Msgs.FluentBadge))
		b.WriteString("\n\n")
	}
	b.WriteString(t.Help.Render(m.deps.Msgs.RestartHelp))
	return b.String()
}
