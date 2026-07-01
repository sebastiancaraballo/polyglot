package rikai

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
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/study"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

const (
	optionCount = 4
	roundLimit  = 6
	blank       = "▁▁▁▁"
)

// Deps are the dependencies the Rikai screen needs.
type Deps struct {
	Theme      ui.Theme
	Msgs       i18n.Messages
	Store      storage.Storage
	ProfileID  int64
	Patterns   []model.Pattern
	Cards      map[string]model.Card // Card.ID -> Card, for frame rendering and options
	ShowRomaji bool
}

// patternEntry is a selectable pattern in the pre-session picker.
type patternEntry struct {
	pattern model.Pattern
	locked  bool // gated behind at least one known filler per slot
}

// round is one substitution drill: which slot varies, its correct filler, and
// the multiple-choice pool.
type round struct {
	slotIdx int
	correct model.Card
	options []model.Card
}

// Model is the Rikai screen. It first shows a pattern picker, then drills the
// chosen pattern: each round shows the frame with one slot blanked (the rest
// fixed at their default filler) and asks the learner to choose the missing
// word from a few known options.
type Model struct {
	deps Deps
	rng  *rand.Rand

	known    map[string]bool                  // Card.ID -> known (survived >=1 review)
	progress map[string]model.PatternProgress // "<patternID>:<slot>" -> progress

	picking    bool
	patterns   []patternEntry
	patternCur int

	pattern model.Pattern
	deck    []round
	index   int

	options       []string
	correct       int
	selected      int
	answered      bool
	correctCount  int
	streakApplied bool
	err           error

	width, height int
}

// New builds the Rikai screen, loading the learner's known-card set and
// per-pattern progress so the picker can gate and annotate each pattern.
func New(deps Deps) Model {
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // not security-sensitive
	m := Model{deps: deps, rng: rng, picking: true, known: map[string]bool{}, progress: map[string]model.PatternProgress{}}
	if deps.Store != nil {
		ctx := context.Background()
		if states, err := deps.Store.GetCardStates(ctx, deps.ProfileID); err == nil {
			for id, st := range states {
				m.known[id] = study.CardKnown(st)
			}
		} else {
			m.err = err
		}
		if saved, err := deps.Store.GetPatternProgress(ctx, deps.ProfileID); err == nil {
			m.progress = saved
		} else {
			m.err = err
		}
	}
	m.patterns = buildPatterns(deps.Patterns, m.known)
	return m
}

func buildPatterns(patterns []model.Pattern, known map[string]bool) []patternEntry {
	entries := make([]patternEntry, 0, len(patterns))
	for _, p := range patterns {
		entries = append(entries, patternEntry{pattern: p, locked: !study.PatternReady(p, known)})
	}
	return entries
}

// startSession builds a fixed-size deck for the chosen pattern, cycling which
// slot varies one round at a time.
func (m Model) startSession() Model {
	m.pattern = m.patterns[m.patternCur].pattern
	m.deck = make([]round, roundLimit)
	for i := range m.deck {
		slotIdx := study.VariableSlotIndex(len(m.pattern.Slots), i)
		slot := m.pattern.Slots[slotIdx]
		candidates := m.knownCards(slot)
		correct := candidates[m.rng.Intn(len(candidates))]
		m.deck[i] = round{slotIdx: slotIdx, correct: correct, options: candidates}
	}
	m.index = 0
	m.correctCount = 0
	m.picking = false
	return m.setQuestion()
}

// knownCards returns the slot's candidate cards the learner already knows.
func (m Model) knownCards(slot model.Slot) []model.Card {
	var cards []model.Card
	for _, id := range slot.CardIDs {
		if m.known[id] {
			cards = append(cards, m.deps.Cards[id])
		}
	}
	return cards
}

func (m Model) setQuestion() Model {
	if m.index >= len(m.deck) {
		return m
	}
	r := m.deck[m.index]
	pool := make([]string, 0, len(r.options))
	for _, c := range r.options {
		pool = append(pool, c.JP)
	}
	m.options, m.correct = study.Options(m.rng, r.correct.JP, pool, optionCount)
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
		if ui.IsConfirmKey(msg) {
			m.picking = true
			m.patterns = buildPatterns(m.deps.Patterns, m.known)
		}
	case m.answered:
		if ui.IsConfirmKey(msg) {
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
		if m.patternCur > 0 {
			m.patternCur--
		}
	case "down", "j":
		if m.patternCur < len(m.patterns)-1 {
			m.patternCur++
		}
	}
	if len(m.patterns) > 0 && ui.IsConfirmKey(msg) && !m.patterns[m.patternCur].locked {
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
	if err := m.persist(correct); err != nil {
		m.err = err
	}
	return m
}

// persist grades the current round's slot, saves the resulting progress, and
// awards XP and the daily study streak (once per session).
func (m *Model) persist(correct bool) error {
	if m.deps.Store == nil {
		return nil
	}
	ctx := context.Background()
	r := m.deck[m.index]
	slot := m.pattern.Slots[r.slotIdx]
	key := m.pattern.ID + ":" + slot.Name
	p := m.progress[key]
	p.PatternID, p.Slot = m.pattern.ID, slot.Name
	p = study.GradePatternSlot(p, correct)
	m.progress[key] = p
	if err := m.deps.Store.SavePatternProgress(ctx, m.deps.ProfileID, p); err != nil {
		return err
	}
	if err := m.deps.Store.AddXP(ctx, m.deps.ProfileID, study.XPForAnswer(correct)); err != nil {
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
	case len(m.patterns) == 0:
		content = m.emptyView()
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

func (m Model) emptyView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.RikaiTitle))
	b.WriteString("\n\n")
	b.WriteString(t.Subtle.Render(m.deps.Msgs.RikaiLocked))
	return b.String()
}

func (m Model) pickerView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.RikaiTitle))
	b.WriteString("\n\n")
	for i, entry := range m.patterns {
		label := entry.pattern.Title + m.patternSuffix(entry.pattern)
		switch {
		case entry.locked:
			b.WriteString(t.Subtle.Render("⊘ " + label))
		case i == m.patternCur:
			b.WriteString(t.Selected.Render("▸ " + label))
		default:
			b.WriteString(t.Normal.Render("  " + label))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if m.patterns[m.patternCur].locked {
		b.WriteString(t.Subtle.Render(m.deps.Msgs.RikaiUnlockHint))
	} else {
		b.WriteString(t.Help.Render(m.deps.Msgs.RikaiPickHelp))
	}
	b.WriteString("\n")
	b.WriteString(t.Subtle.Render(m.deps.Msgs.RikaiMasteryNote))
	return b.String()
}

// patternSuffix annotates a picker row with how many of its slots are mastered.
func (m Model) patternSuffix(p model.Pattern) string {
	if len(p.Slots) == 0 {
		return ""
	}
	mastered := 0
	for _, slot := range p.Slots {
		if m.progress[p.ID+":"+slot.Name].Mastered {
			mastered++
		}
	}
	if mastered >= len(p.Slots) {
		return "  ✓ " + m.deps.Msgs.RikaiPatternFluent
	}
	return "  " + fmt.Sprintf(m.deps.Msgs.RikaiMasteredFmt, mastered, len(p.Slots))
}

func (m Model) questionView() string {
	t := m.deps.Theme
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %d/%d\n\n", t.Title.Render(m.deps.Msgs.RikaiTitle), m.index+1, len(m.deck))

	r := m.deck[m.index]
	fill := make(map[string]string, len(m.pattern.Slots))
	for i, slot := range m.pattern.Slots {
		if i == r.slotIdx {
			fill[slot.Name] = blank
			continue
		}
		fill[slot.Name] = m.deps.Cards[slot.Default].JP
	}
	b.WriteString(t.Accent.Bold(true).Render(study.RenderFrame(m.pattern.Frame, fill)))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, m.deps.Msgs.RikaiQuestionFmt, r.correct.Source)
	b.WriteString("\n\n")

	for i, opt := range m.options {
		label := opt
		if m.deps.ShowRomaji {
			if romaji := m.romajiFor(opt); romaji != "" {
				label = fmt.Sprintf("%s (%s)", opt, romaji)
			}
		}
		line := fmt.Sprintf(" %d) %s", i+1, label)
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
	if m.answered {
		b.WriteString(t.Help.Render(m.deps.Msgs.ContinueHelp))
	} else {
		b.WriteString(t.Help.Render(m.deps.Msgs.ChoiceHelp))
	}
	return b.String()
}

// romajiFor looks up the romaji reading for a rendered JP option among the
// current round's candidates.
func (m Model) romajiFor(jp string) string {
	for _, c := range m.deck[m.index].options {
		if c.JP == jp {
			return c.Romaji
		}
	}
	return ""
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
