// Package assessment runs the end-of-level mock exam: a cross-curriculum
// retrieval-practice test (vocabulary, kana, grammar patterns) graded against
// the same 80% mastery band as the end-of-chapter challenge. Passing certifies
// the level; the result is persisted and never revoked (Mastery Learning). Like
// the challenge, every answer flows through the regular spaced-repetition and XP
// paths, so a failed attempt is still learning and retries are immediate.
package assessment

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

// blank marks the slot the learner must fill in a grammar-pattern question,
// matching the Rikai drill's placeholder.
const blank = "▁▁▁▁"

// Deps are the dependencies the assessment runner needs.
type Deps struct {
	Theme      ui.Theme
	Msgs       i18n.Messages
	Store      storage.Storage
	ProfileID  int64
	Level      model.JLPT
	Lessons    []model.Lesson
	Kana       []model.KanaItem
	Patterns   []model.Pattern
	Cards      map[string]model.Card // card ID -> card, for resolving pattern slots and romaji
	ShowRomaji bool
}

type phase int

const (
	phaseIntro    phase = iota // stating the mastery bar before the first question
	phaseQuestion              // answering the deck
	phaseResult                // pass/fail with a review list
)

// Model is the assessment runner.
type Model struct {
	deps Deps
	rng  *rand.Rand

	phase    phase
	deck     []study.AssessQuestion
	index    int
	selected int
	answered bool

	correctCount int
	missed       []study.AssessQuestion

	// Mastery signals, cached and refreshed as answers are graded.
	kanaProgress    map[string]model.KanaProgress
	patternProgress map[string]model.PatternProgress
	streakApplied   bool

	romaji map[string]string      // JP -> romaji, for option labels
	prior  model.AssessmentResult // best/passed loaded at start
	err    error

	width, height int
}

// New builds the assessment runner, loading prior mastery signals and the
// stored result, and sampling the deck.
func New(deps Deps) Model {
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // not security-sensitive
	m := Model{
		deps:            deps,
		rng:             rng,
		phase:           phaseIntro,
		kanaProgress:    map[string]model.KanaProgress{},
		patternProgress: map[string]model.PatternProgress{},
		romaji:          map[string]string{},
	}
	for _, c := range deps.Cards {
		m.romaji[c.JP] = c.Romaji
	}
	if deps.Store != nil {
		ctx := context.Background()
		if kp, err := deps.Store.GetKanaProgress(ctx, deps.ProfileID); err == nil {
			m.kanaProgress = kp
		} else {
			m.err = err
		}
		if pp, err := deps.Store.GetPatternProgress(ctx, deps.ProfileID); err == nil {
			m.patternProgress = pp
		} else {
			m.err = err
		}
		if r, err := deps.Store.GetAssessmentResult(ctx, deps.ProfileID, deps.Level); err == nil {
			m.prior = r
		} else {
			m.err = err
		}
	}
	m.deck = study.BuildAssessment(rng, deps.Level, deps.Lessons, deps.Kana, deps.Patterns, deps.Cards)
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

	switch m.phase {
	case phaseIntro:
		if ui.IsConfirmKey(msg) {
			m = m.start()
		}
	case phaseQuestion:
		if !m.answered {
			m = m.answerKey(msg)
		} else if ui.IsConfirmKey(msg) {
			m = m.advance()
		}
	case phaseResult:
		if ui.IsConfirmKey(msg) {
			if m.attemptPassed() {
				return m, nav.Back()
			}
			m = m.restart() // a failed attempt can be retried immediately
		}
	}
	return m, nil
}

// start moves from the intro into the question loop. An empty deck (no content
// to assess) goes straight to the result so the screen never indexes an empty
// deck.
func (m Model) start() Model {
	m.index = 0
	m.correctCount = 0
	m.selected = 0
	m.answered = false
	m.missed = nil
	if len(m.deck) == 0 {
		return m.finish()
	}
	m.phase = phaseQuestion
	return m
}

// restart re-samples a fresh deck and begins again.
func (m Model) restart() Model {
	m.deck = study.BuildAssessment(m.rng, m.deps.Level, m.deps.Lessons, m.deps.Kana, m.deps.Patterns, m.deps.Cards)
	return m.start()
}

func (m Model) answerKey(msg tea.KeyPressMsg) Model {
	opts := m.deck[m.index].Options
	switch msg.String() {
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		if m.selected < len(opts)-1 {
			m.selected++
		}
	case "1", "2", "3", "4":
		i := int(msg.String()[0] - '1')
		if i < len(opts) {
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
	q := m.deck[m.index]
	correct := m.selected == q.Correct
	if correct {
		m.correctCount++
	} else {
		m.missed = append(m.missed, q)
	}
	if err := m.grade(q, correct); err != nil {
		m.err = err
	}
	return m
}

func (m Model) advance() Model {
	m.index++
	m.selected = 0
	m.answered = false
	if m.index >= len(m.deck) {
		return m.finish()
	}
	return m
}

// finish records the attempt: the pass is sticky and the best score is kept.
func (m Model) finish() Model {
	m.phase = phaseResult
	if m.deps.Store == nil {
		return m
	}
	r := m.prior
	r.Level = m.deps.Level
	r.Passed = r.Passed || m.attemptPassed()
	if r.TakenAt.IsZero() || m.correctCount > r.BestCorrect {
		r.BestCorrect = m.correctCount
		r.Total = len(m.deck)
	}
	r.TakenAt = time.Now()
	if err := m.deps.Store.SaveAssessmentResult(context.Background(), m.deps.ProfileID, r); err != nil {
		m.err = err
		return m
	}
	m.prior = r
	return m
}

func (m Model) attemptPassed() bool {
	return study.ChallengePassed(m.correctCount, len(m.deck))
}

// grade folds one answer into the learner's mastery signals through the same
// paths as regular practice: vocab and pattern fillers as SRS reviews, kana as
// an automaticity streak, pattern slots as a correctness streak, plus XP and
// the daily study streak.
func (m *Model) grade(q study.AssessQuestion, correct bool) error {
	if m.deps.Store == nil {
		return nil
	}
	ctx := context.Background()
	switch q.Kind {
	case study.AssessKana:
		p := m.kanaProgress[q.Kana.Char]
		p.Char = q.Kana.Char
		p = study.GradeKana(p, correct, 0)
		m.kanaProgress[q.Kana.Char] = p
		if err := m.deps.Store.SaveKanaProgress(ctx, m.deps.ProfileID, p); err != nil {
			return err
		}
	case study.AssessPattern:
		slot := q.Pattern.Slots[q.SlotIdx]
		key := q.Pattern.ID + ":" + slot.Name
		p := m.patternProgress[key]
		p.PatternID, p.Slot = q.Pattern.ID, slot.Name
		p = study.GradePatternSlot(p, correct)
		m.patternProgress[key] = p
		if err := m.deps.Store.SavePatternProgress(ctx, m.deps.ProfileID, p); err != nil {
			return err
		}
		if err := m.reviewCard(ctx, q.Card, correct); err != nil {
			return err
		}
	default: // AssessVocab
		if err := m.reviewCard(ctx, q.Card, correct); err != nil {
			return err
		}
	}
	return m.awardXP(ctx, correct)
}

func (m *Model) reviewCard(ctx context.Context, card model.Card, correct bool) error {
	state, err := m.deps.Store.GetCardState(ctx, m.deps.ProfileID, card.ID)
	if err != nil {
		state = srs.NewCard(card.ID)
	}
	grade := srs.Again
	if correct {
		grade = srs.Good
	}
	state = srs.Review(state, grade, time.Now())
	return m.deps.Store.SaveCardState(ctx, m.deps.ProfileID, state)
}

func (m *Model) awardXP(ctx context.Context, correct bool) error {
	if err := m.deps.Store.AddXP(ctx, m.deps.ProfileID, study.XPForAnswer(correct)); err != nil {
		return err
	}
	if m.streakApplied {
		return nil
	}
	stats, err := m.deps.Store.GetStats(ctx, m.deps.ProfileID)
	if err != nil {
		return err
	}
	if err := m.deps.Store.SaveStats(ctx, m.deps.ProfileID, study.UpdateStreak(stats, time.Now())); err != nil {
		return err
	}
	m.streakApplied = true
	return nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var content string
	switch m.phase {
	case phaseResult:
		content = m.resultView()
	case phaseQuestion:
		content = m.questionView()
	default:
		content = m.introView()
	}
	view := tea.NewView(ui.Frame(m.deps.Theme, m.width, m.height, content))
	view.AltScreen = true
	return view
}

func (m Model) introView() string {
	t := m.deps.Theme
	width := ui.FrameContentWidth(t, m.width)
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.AssessmentTitle))
	b.WriteString("\n\n")
	body := fmt.Sprintf(m.deps.Msgs.AssessmentIntroFmt, study.ChallengeNeeded(len(m.deck)), len(m.deck))
	b.WriteString(t.Normal.Render(ui.WrapText(body, width)))
	b.WriteString("\n")
	if !m.prior.TakenAt.IsZero() {
		best := fmt.Sprintf(m.deps.Msgs.AssessmentBestFmt, m.prior.BestCorrect, m.prior.Total)
		if m.prior.Passed {
			best += " " + m.deps.Msgs.AssessmentPassedBadge
		}
		b.WriteString("\n")
		b.WriteString(t.Subtle.Render(ui.WrapText(best, width)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.ContinueHelp))
	return b.String()
}

func (m Model) questionView() string {
	t := m.deps.Theme
	width := ui.FrameContentWidth(t, m.width)
	q := m.deck[m.index]
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %d/%d\n\n", t.Title.Render(m.deps.Msgs.AssessmentTitle), m.index+1, len(m.deck))

	switch q.Kind {
	case study.AssessKana:
		b.WriteString(ui.WrapText(m.deps.Msgs.KanaPrompt, width))
		b.WriteString("\n\n")
		b.WriteString(t.Accent.Bold(true).Render(q.Kana.Char))
	case study.AssessPattern:
		fill := map[string]string{q.Pattern.Slots[q.SlotIdx].Name: blank}
		for name, jp := range q.Fill {
			fill[name] = jp
		}
		b.WriteString(t.Accent.Bold(true).Render(ui.WrapText(study.RenderFrame(q.Pattern.Frame, fill), width)))
		b.WriteString("\n\n")
		b.WriteString(ui.WrapText(fmt.Sprintf(m.deps.Msgs.AssessmentPatternPromptFmt, q.Card.Source), width))
	default: // AssessVocab
		b.WriteString(ui.WrapText(fmt.Sprintf(m.deps.Msgs.QuizQuestionFmt, q.Card.Source), width))
	}
	b.WriteString("\n\n")

	for i, opt := range q.Options {
		label := opt
		if m.deps.ShowRomaji && q.Kind != study.AssessKana {
			if r := m.romaji[opt]; r != "" {
				label = fmt.Sprintf("%s (%s)", opt, r)
			}
		}
		line := fmt.Sprintf(" %d) %s", i+1, label)
		switch {
		case m.answered && i == q.Correct:
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

func (m Model) resultView() string {
	t := m.deps.Theme
	width := ui.FrameContentWidth(t, m.width)
	passed := m.attemptPassed()

	var b strings.Builder
	if passed {
		b.WriteString(t.Success.Render(m.deps.Msgs.AssessmentPassTitle))
		b.WriteString("\n\n")
		b.WriteString(ui.WrapText(fmt.Sprintf(m.deps.Msgs.AssessmentPassFmt, m.correctCount, len(m.deck)), width))
	} else {
		b.WriteString(t.Title.Render(m.deps.Msgs.AssessmentFailTitle))
		b.WriteString("\n\n")
		b.WriteString(ui.WrapText(fmt.Sprintf(m.deps.Msgs.AssessmentFailFmt,
			m.correctCount, len(m.deck), study.ChallengeNeeded(len(m.deck))), width))
	}
	b.WriteString("\n")

	help := m.deps.Msgs.AssessmentDoneHelp
	if !passed {
		help = m.deps.Msgs.AssessmentRetryHelp
	}

	if !passed && len(m.missed) > 0 {
		b.WriteString("\n")
		b.WriteString(t.Subtle.Render(m.deps.Msgs.AssessmentMissedLbl))
		b.WriteString("\n")
		b.WriteString(m.missedList(width, lipglossHeight(b.String())+2))
	}
	b.WriteString("\n")
	b.WriteString(t.Help.Render(help))
	return b.String()
}

// missedList renders the review list, capped by the height still available in
// the frame so the result never overflows: it adds wrapped item lines while
// they fit, then a "+N más" note for the remainder.
func (m Model) missedList(width, used int) string {
	avail := ui.FrameContentHeight(m.deps.Theme, m.height) - used - 1 // reserve the help row
	if avail < 1 {
		avail = 1
	}
	var b strings.Builder
	shown, height := 0, 0
	for _, q := range m.missed {
		line := ui.WrapText(missedLine(q, m.deps.ShowRomaji), width)
		h := lipglossHeight(line)
		// Leave a row for the "+N más" note when more items remain.
		remaining := len(m.missed) - shown
		reserve := 0
		if remaining > 1 {
			reserve = 1
		}
		if height+h > avail-reserve && shown > 0 {
			break
		}
		b.WriteString(line)
		b.WriteString("\n")
		height += h
		shown++
	}
	if shown < len(m.missed) {
		b.WriteString(m.deps.Theme.Subtle.Render(fmt.Sprintf(m.deps.Msgs.AssessmentMoreFmt, len(m.missed)-shown)))
		b.WriteString("\n")
	}
	return b.String()
}

// missedLine formats one missed item with its answer; the learner just failed
// it, so the reading is shown.
func missedLine(q study.AssessQuestion, showRomaji bool) string {
	switch q.Kind {
	case study.AssessKana:
		return fmt.Sprintf("%s (%s)", q.Kana.Char, q.Kana.Romaji)
	default: // vocab, or a pattern's correct filler
		if showRomaji && q.Card.Romaji != "" {
			return fmt.Sprintf("%s (%s) — %s", q.Card.JP, q.Card.Romaji, q.Card.Source)
		}
		return fmt.Sprintf("%s — %s", q.Card.JP, q.Card.Source)
	}
}

// lipglossHeight counts the display rows a string occupies (built from WrapText
// output, so hard newlines are the only breaks).
func lipglossHeight(s string) int { return strings.Count(s, "\n") + 1 }
