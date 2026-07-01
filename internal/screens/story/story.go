package story

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

const optionCount = 4

// Deps are the dependencies the Katsudoo story runner needs.
type Deps struct {
	Theme      ui.Theme
	Msgs       i18n.Messages
	Store      storage.Storage
	ProfileID  int64
	Chapters   []model.Chapter
	Lessons    []model.Lesson   // resolves a vocab practice beat's RefID
	Kana       []model.KanaItem // resolves a kana practice beat's RefID
	ShowRomaji bool
}

// chapterEntry is a selectable chapter in the pre-session picker.
type chapterEntry struct {
	chapter  model.Chapter
	progress model.StoryProgress
}

// Model is the Katsudoo story runner. It first shows a chapter picker, then
// steps through the chosen chapter's beats: narration and dialogue are shown
// and advanced on confirm; a practice beat pauses for one inline check that
// reuses the same grading logic as the real kana trainer and quiz screens,
// without embedding their Bubble Tea models.
type Model struct {
	deps Deps
	rng  *rand.Rand

	kanaProgress map[string]model.KanaProgress // cached, refreshed on each kana practice answer

	picking    bool
	chapters   []chapterEntry
	chapterCur int

	chapter   model.Chapter
	beatIndex int

	options  []string
	correct  int
	selected int
	answered bool

	practiceKind model.PracticeKind
	practiceCard model.Card     // set when practiceKind == PracticeVocab
	practiceKana model.KanaItem // set when practiceKind == PracticeKana

	streakApplied bool
	err           error

	width, height int
}

// New builds the story runner, loading the learner's per-chapter progress so
// the picker can show where they left off.
func New(deps Deps) Model {
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // not security-sensitive
	m := Model{deps: deps, rng: rng, picking: true, kanaProgress: map[string]model.KanaProgress{}}
	if deps.Store != nil {
		if saved, err := deps.Store.GetKanaProgress(context.Background(), deps.ProfileID); err == nil {
			m.kanaProgress = saved
		} else {
			m.err = err
		}
	}
	return m.refreshChapters()
}

// refreshChapters (re)loads per-chapter progress from the store and rebuilds
// the picker entries, so returning to the picker after finishing a chapter
// reflects it immediately.
func (m Model) refreshChapters() Model {
	progress := map[string]model.StoryProgress{}
	if m.deps.Store != nil {
		if saved, err := m.deps.Store.GetStoryProgress(context.Background(), m.deps.ProfileID); err == nil {
			progress = saved
		} else {
			m.err = err
		}
	}
	m.chapters = make([]chapterEntry, 0, len(m.deps.Chapters))
	for _, c := range m.deps.Chapters {
		m.chapters = append(m.chapters, chapterEntry{chapter: c, progress: progress[c.ID]})
	}
	return m
}

// startChapter opens the chosen chapter, resuming at its saved beat unless it
// was already completed (in which case it replays from the start).
func (m Model) startChapter(i int) Model {
	entry := m.chapters[i]
	m.chapter = entry.chapter
	m.beatIndex = 0
	if !entry.progress.Completed && entry.progress.BeatIndex < len(m.chapter.Beats) {
		m.beatIndex = entry.progress.BeatIndex
	}
	m.picking = false
	m.streakApplied = false
	return m.enterBeat()
}

// enterBeat prepares whatever the current beat needs: a practice beat builds
// its question; narration/dialogue need no extra state.
func (m Model) enterBeat() Model {
	if m.finished() {
		return m
	}
	if m.chapter.Beats[m.beatIndex].Kind == model.Practice {
		m = m.setPracticeQuestion()
	}
	return m
}

func (m Model) finished() bool { return m.beatIndex >= len(m.chapter.Beats) }

func (m Model) setPracticeQuestion() Model {
	beat := m.chapter.Beats[m.beatIndex]
	m.practiceKind = beat.Practice
	switch beat.Practice {
	case model.PracticeVocab:
		lesson := lessonByID(m.deps.Lessons, beat.RefID)
		if lesson == nil {
			return m
		}
		m.practiceCard = lesson.Cards[m.rng.Intn(len(lesson.Cards))]
		pool := make([]string, 0, len(lesson.Cards))
		for _, c := range lesson.Cards {
			pool = append(pool, c.JP)
		}
		m.options, m.correct = study.Options(m.rng, m.practiceCard.JP, pool, optionCount)
	case model.PracticeKana:
		filtered := filterKana(m.deps.Kana, model.KanaType(beat.RefID))
		if len(filtered) == 0 {
			return m
		}
		m.practiceKana = filtered[m.rng.Intn(len(filtered))]
		pool := make([]string, 0, len(filtered))
		for _, k := range filtered {
			pool = append(pool, k.Romaji)
		}
		m.options, m.correct = study.Options(m.rng, m.practiceKana.Romaji, pool, optionCount)
	}
	m.selected = 0
	m.answered = false
	return m
}

func lessonByID(lessons []model.Lesson, id string) *model.Lesson {
	for i := range lessons {
		if lessons[i].ID == id {
			return &lessons[i]
		}
	}
	return nil
}

func filterKana(kana []model.KanaItem, typ model.KanaType) []model.KanaItem {
	var out []model.KanaItem
	for _, k := range kana {
		if k.Type == typ {
			out = append(out, k)
		}
	}
	return out
}

// advance persists that the current beat was seen and moves to the next one.
func (m Model) advance() Model {
	m.beatIndex++
	if m.deps.Store != nil {
		p := model.StoryProgress{ChapterID: m.chapter.ID, BeatIndex: m.beatIndex, Completed: m.finished()}
		if err := m.deps.Store.SaveStoryProgress(context.Background(), m.deps.ProfileID, p); err != nil {
			m.err = err
		}
	}
	return m.enterBeat()
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

	if m.picking {
		return m.handlePick(msg)
	}

	switch {
	case m.finished():
		if ui.IsConfirmKey(msg) {
			m = m.refreshChapters()
			m.picking = true
		}
	case m.chapter.Beats[m.beatIndex].Kind == model.Practice && !m.answered:
		m = m.answerKey(msg)
	case ui.IsConfirmKey(msg):
		m = m.advance()
	}
	return m, nil
}

func (m Model) handlePick(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.chapterCur > 0 {
			m.chapterCur--
		}
	case "down", "j":
		if m.chapterCur < len(m.chapters)-1 {
			m.chapterCur++
		}
	}
	if len(m.chapters) > 0 && ui.IsConfirmKey(msg) {
		m = m.startChapter(m.chapterCur)
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
	switch m.practiceKind {
	case model.PracticeVocab:
		m = m.persistVocab(correct)
	case model.PracticeKana:
		m = m.persistKana(correct)
	}
	return m
}

// persistVocab grades the current vocab practice beat the same way quiz.go
// does: an SRS review plus XP.
func (m Model) persistVocab(correct bool) Model {
	if m.deps.Store == nil {
		return m
	}
	ctx := context.Background()
	state, err := m.deps.Store.GetCardState(ctx, m.deps.ProfileID, m.practiceCard.ID)
	if err != nil {
		state = srs.NewCard(m.practiceCard.ID)
	}
	grade := srs.Again
	if correct {
		grade = srs.Good
	}
	state = srs.Review(state, grade, time.Now())
	if err := m.deps.Store.SaveCardState(ctx, m.deps.ProfileID, state); err != nil {
		m.err = err
		return m
	}
	return m.awardXP(correct)
}

// persistKana grades the current kana practice beat the same way kana.go
// does: a correctness-only mastery streak. Untimed (elapsed=0): a diegetic
// story beat has no reason to time the answer, and untimed grading is already
// a first-class supported input.
func (m Model) persistKana(correct bool) Model {
	if m.deps.Store == nil {
		return m
	}
	char := m.practiceKana.Char
	p := m.kanaProgress[char]
	p.Char = char
	p = study.GradeKana(p, correct, 0)
	m.kanaProgress[char] = p
	if err := m.deps.Store.SaveKanaProgress(context.Background(), m.deps.ProfileID, p); err != nil {
		m.err = err
		return m
	}
	return m.awardXP(correct)
}

// awardXP grants XP for the answer and applies the daily study streak once
// per screen visit, mirroring quiz.go's persist.
func (m Model) awardXP(correct bool) Model {
	ctx := context.Background()
	if err := m.deps.Store.AddXP(ctx, m.deps.ProfileID, study.XPForAnswer(correct)); err != nil {
		m.err = err
		return m
	}
	if !m.streakApplied {
		stats, err := m.deps.Store.GetStats(ctx, m.deps.ProfileID)
		if err != nil {
			m.err = err
			return m
		}
		if err := m.deps.Store.SaveStats(ctx, m.deps.ProfileID, study.UpdateStreak(stats, time.Now())); err != nil {
			m.err = err
			return m
		}
		m.streakApplied = true
	}
	return m
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var content string
	switch {
	case len(m.chapters) == 0:
		content = m.emptyView()
	case m.picking:
		content = m.pickerView()
	case m.finished():
		content = m.doneView()
	case m.chapter.Beats[m.beatIndex].Kind == model.Practice:
		content = m.practiceView()
	default:
		content = m.beatView()
	}
	view := tea.NewView(ui.Frame(m.deps.Theme, m.width, m.height, content))
	view.AltScreen = true
	return view
}

func (m Model) emptyView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.StoryTitle))
	b.WriteString("\n\n")
	b.WriteString(t.Subtle.Render(m.deps.Msgs.StoryEmpty))
	return b.String()
}

func (m Model) pickerView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.StoryTitle))
	b.WriteString("\n\n")
	for i, entry := range m.chapters {
		label := entry.chapter.Title + chapterSuffix(m.deps.Msgs, entry)
		switch i {
		case m.chapterCur:
			b.WriteString(t.Selected.Render("▸ " + label))
		default:
			b.WriteString(t.Normal.Render("  " + label))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.StoryPickHelp))
	return b.String()
}

func chapterSuffix(msgs i18n.Messages, entry chapterEntry) string {
	switch {
	case entry.progress.Completed:
		return "  " + msgs.StoryCompleteBadge
	case entry.progress.BeatIndex > 0:
		return "  " + fmt.Sprintf(msgs.StoryProgressFmt, entry.progress.BeatIndex, len(entry.chapter.Beats))
	default:
		return ""
	}
}

func (m Model) beatView() string {
	t := m.deps.Theme
	beat := m.chapter.Beats[m.beatIndex]
	var b strings.Builder
	b.WriteString(t.Title.Render(m.chapter.Title))
	b.WriteString("\n\n")
	if beat.Place != "" {
		b.WriteString(t.Subtle.Render(beat.Place))
		b.WriteString("\n\n")
	}
	if beat.Kind == model.Dialogue {
		b.WriteString(t.Accent.Bold(true).Render(beat.Speaker))
		b.WriteString("\n")
	}
	b.WriteString(t.Normal.Render(beat.JP))
	if m.deps.ShowRomaji && beat.Romaji != "" {
		fmt.Fprintf(&b, " (%s)", beat.Romaji)
	}
	b.WriteString("\n")
	b.WriteString(t.Subtle.Render(beat.Source))
	b.WriteString("\n\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.ContinueHelp))
	return b.String()
}

func (m Model) practiceView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.chapter.Title))
	b.WriteString("\n\n")

	switch m.practiceKind {
	case model.PracticeVocab:
		fmt.Fprintf(&b, m.deps.Msgs.QuizQuestionFmt, m.practiceCard.Source)
	case model.PracticeKana:
		b.WriteString(m.deps.Msgs.KanaPrompt)
		b.WriteString("\n\n")
		b.WriteString(t.Accent.Bold(true).Render(m.practiceKana.Char))
	}
	b.WriteString("\n\n")

	romaji := map[string]string{}
	if m.practiceKind == model.PracticeVocab {
		lesson := lessonByID(m.deps.Lessons, m.chapter.Beats[m.beatIndex].RefID)
		if lesson != nil {
			for _, c := range lesson.Cards {
				romaji[c.JP] = c.Romaji
			}
		}
	}
	for i, opt := range m.options {
		label := opt
		if m.deps.ShowRomaji && m.practiceKind == model.PracticeVocab {
			if r := romaji[opt]; r != "" {
				label = fmt.Sprintf("%s (%s)", opt, r)
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

func (m Model) doneView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.StoryDoneTitle))
	b.WriteString("\n\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.StoryDoneNext))
	return b.String()
}
