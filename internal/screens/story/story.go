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
	locked   bool // gated behind mastering the previous chapter
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

	// End-of-chapter challenge state. The challenge reuses the practice
	// fields above so its answers are graded and persisted through the exact
	// same paths as practice beats.
	challenge       []study.ChallengeQuestion // nil = not in a challenge
	challengeIntro  bool                      // showing the pre-challenge explainer
	challengeIdx    int
	challengeRight  int
	challengeMissed []study.ChallengeQuestion
	chapterMastered bool // loaded at startChapter; preserved by advance, set by markMastered
	newlyMastered   bool // mastery was earned this run — gates the unlock announcement

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
	for i, c := range m.deps.Chapters {
		// Mastery gate: each chapter unlocks by mastering the previous one
		// (Mastery Learning: advance on demonstrated mastery, not on having
		// clicked through). The first chapter is always open.
		locked := i > 0 && !progress[m.deps.Chapters[i-1].ID].Mastered
		m.chapters = append(m.chapters, chapterEntry{chapter: c, progress: progress[c.ID], locked: locked})
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
	m.challenge = nil
	m.challengeIntro = false
	m.challengeIdx, m.challengeRight = 0, 0
	m.challengeMissed = nil
	m.chapterMastered = entry.progress.Mastered
	m.newlyMastered = false
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

// advance persists that the current beat was seen and moves to the next one;
// finishing the last beat opens the end-of-chapter challenge. Mastered is
// carried through so replaying an already-mastered chapter never revokes it.
func (m Model) advance() Model {
	m.beatIndex++
	if m.deps.Store != nil {
		p := model.StoryProgress{ChapterID: m.chapter.ID, BeatIndex: m.beatIndex, Completed: m.finished(), Mastered: m.chapterMastered}
		if err := m.deps.Store.SaveStoryProgress(context.Background(), m.deps.ProfileID, p); err != nil {
			m.err = err
		}
	}
	if m.finished() {
		return m.startChallenge()
	}
	return m.enterBeat()
}

// startChallenge builds the end-of-chapter retrieval challenge. A chapter
// with no practice beats has nothing to gate on, so completing it counts as
// mastering it and the challenge is skipped.
func (m Model) startChallenge() Model {
	qs := study.BuildChallenge(m.rng, m.chapter, m.deps.Lessons, m.deps.Kana)
	if len(qs) == 0 {
		return m.markMastered()
	}
	m.challenge = qs
	m.challengeIntro = true
	m.challengeIdx, m.challengeRight = 0, 0
	m.challengeMissed = nil
	return m
}

func (m Model) challengeFinished() bool { return m.challengeIdx >= len(m.challenge) }

func (m Model) challengePassed() bool {
	return study.ChallengePassed(m.challengeRight, len(m.challenge))
}

// setChallengeQuestion mirrors setPracticeQuestion, reading the current
// challenge question instead of the current beat, so the answer flows through
// the same answering and grading paths.
func (m Model) setChallengeQuestion() Model {
	q := m.challenge[m.challengeIdx]
	m.practiceKind = q.Practice
	switch q.Practice {
	case model.PracticeVocab:
		m.practiceCard = q.Card
		lesson := lessonByID(m.deps.Lessons, q.RefID)
		if lesson == nil {
			return m
		}
		pool := make([]string, 0, len(lesson.Cards))
		for _, c := range lesson.Cards {
			pool = append(pool, c.JP)
		}
		m.options, m.correct = study.Options(m.rng, q.Card.JP, pool, optionCount)
	case model.PracticeKana:
		m.practiceKana = q.Kana
		filtered := filterKana(m.deps.Kana, model.KanaType(q.RefID))
		pool := make([]string, 0, len(filtered))
		for _, k := range filtered {
			pool = append(pool, k.Romaji)
		}
		m.options, m.correct = study.Options(m.rng, q.Kana.Romaji, pool, optionCount)
	}
	m.selected = 0
	m.answered = false
	return m
}

// recordChallengeAnswer tallies the revealed answer and moves to the next
// question; passing the whole challenge masters the chapter.
func (m Model) recordChallengeAnswer() Model {
	if m.selected == m.correct {
		m.challengeRight++
	} else {
		m.challengeMissed = append(m.challengeMissed, m.challenge[m.challengeIdx])
	}
	m.challengeIdx++
	if !m.challengeFinished() {
		return m.setChallengeQuestion()
	}
	if m.challengePassed() {
		m = m.markMastered()
	}
	return m
}

// markMastered records that the chapter's challenge was passed (or that the
// chapter has nothing to evaluate). Mastery is never revoked.
func (m Model) markMastered() Model {
	if !m.chapterMastered {
		m.newlyMastered = true
	}
	m.chapterMastered = true
	if m.deps.Store != nil {
		p := model.StoryProgress{ChapterID: m.chapter.ID, BeatIndex: len(m.chapter.Beats), Completed: true, Mastered: true}
		if err := m.deps.Store.SaveStoryProgress(context.Background(), m.deps.ProfileID, p); err != nil {
			m.err = err
		}
	}
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

	if m.picking {
		return m.handlePick(msg)
	}

	// The challenge branches come before the finished() case: during a
	// challenge, finished() is already true and beatIndex is out of range.
	switch {
	case m.challenge != nil && m.challengeIntro:
		if ui.IsConfirmKey(msg) {
			m.challengeIntro = false
			m = m.setChallengeQuestion()
		}
	case m.challenge != nil && !m.challengeFinished():
		if !m.answered {
			m = m.answerKey(msg)
		} else if ui.IsConfirmKey(msg) {
			m = m.recordChallengeAnswer()
		}
	case m.challenge != nil && !m.challengePassed():
		if ui.IsConfirmKey(msg) {
			m = m.startChallenge() // retry with a fresh draw
		}
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
	if len(m.chapters) > 0 && ui.IsConfirmKey(msg) && !m.chapters[m.chapterCur].locked {
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
	case m.challenge != nil && m.challengeIntro:
		content = m.challengeIntroView()
	case m.challenge != nil && !m.challengeFinished():
		content = m.challengeView()
	case m.challenge != nil && !m.challengePassed():
		content = m.challengeFailView()
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
		switch {
		case entry.locked:
			b.WriteString(t.Subtle.Render("⊘ " + entry.chapter.Title))
		case i == m.chapterCur:
			b.WriteString(t.Selected.Render("▸ " + label))
		default:
			b.WriteString(t.Normal.Render("  " + label))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if m.chapters[m.chapterCur].locked {
		// The first chapter is never locked, so a locked cursor row always
		// has a predecessor to name.
		hint := fmt.Sprintf(m.deps.Msgs.StoryLockedHintFmt, m.chapters[m.chapterCur-1].chapter.Title)
		b.WriteString(t.Subtle.Render(hint))
	} else {
		b.WriteString(t.Help.Render(m.deps.Msgs.StoryPickHelp))
	}
	b.WriteString("\n")
	b.WriteString(t.Subtle.Render(m.deps.Msgs.StoryGateNote))
	return b.String()
}

func chapterSuffix(msgs i18n.Messages, entry chapterEntry) string {
	switch {
	case entry.progress.Mastered:
		return "  " + msgs.StoryMasteredBadge
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
	return m.questionView(m.chapter.Title, "", m.chapter.Beats[m.beatIndex].RefID)
}

func (m Model) challengeView() string {
	sub := fmt.Sprintf(m.deps.Msgs.StoryChallengeQFmt, m.challengeIdx+1, len(m.challenge))
	return m.questionView(m.deps.Msgs.StoryChallengeTitle, sub, m.challenge[m.challengeIdx].RefID)
}

// questionView renders the shared question body for practice beats and
// challenge questions: title (with an optional subtitle), the prompt, the
// options with reveal marks, and the phase help. refID names the pool the
// question was drawn from, for the vocab romaji labels.
func (m Model) questionView(title, subtitle, refID string) string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(title))
	if subtitle != "" {
		b.WriteString("  " + t.Subtle.Render(subtitle))
	}
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
		lesson := lessonByID(m.deps.Lessons, refID)
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

// challengeIntroView states the mastery bar before the first question, so the
// gate's rule is visible before it ever bites.
func (m Model) challengeIntroView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.StoryChallengeTitle))
	b.WriteString("\n\n")
	body := fmt.Sprintf(m.deps.Msgs.StoryChallengeIntroFmt, study.ChallengeNeeded(len(m.challenge)), len(m.challenge))
	b.WriteString(t.Normal.Render(ui.WrapText(body, ui.FrameContentWidth(t, m.width))))
	b.WriteString("\n\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.ContinueHelp))
	return b.String()
}

// challengeFailView shows the score against the stated bar, the missed items
// to review, and offers an immediate retry — a failed retrieval attempt is
// still practice (testing effect), so nothing is locked away.
func (m Model) challengeFailView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.StoryChallengeTitle))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, m.deps.Msgs.StoryChallengeFailFmt, m.challengeRight, len(m.challenge), study.ChallengeNeeded(len(m.challenge)))
	b.WriteString("\n")
	if len(m.challengeMissed) > 0 {
		b.WriteString("\n")
		b.WriteString(t.Subtle.Render(m.deps.Msgs.StoryChallengeMissedLbl))
		b.WriteString("\n")
		for _, q := range m.challengeMissed {
			b.WriteString(t.Normal.Render(missedLine(q, m.deps.ShowRomaji)))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.StoryChallengeRetryHelp))
	return b.String()
}

// missedLine formats one missed challenge item with its answer: the learner
// just failed it, so the reading is shown, not hidden.
func missedLine(q study.ChallengeQuestion, showRomaji bool) string {
	switch q.Practice {
	case model.PracticeKana:
		return fmt.Sprintf("%s (%s)", q.Kana.Char, q.Kana.Romaji)
	default:
		if showRomaji && q.Card.Romaji != "" {
			return fmt.Sprintf("%s (%s) — %s", q.Card.JP, q.Card.Romaji, q.Card.Source)
		}
		return fmt.Sprintf("%s — %s", q.Card.JP, q.Card.Source)
	}
}

func (m Model) doneView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.StoryDoneTitle))
	b.WriteString("\n\n")
	if m.challenge != nil {
		fmt.Fprintf(&b, m.deps.Msgs.StoryChallengePassFmt, m.challengeRight, len(m.challenge))
		b.WriteString("\n")
	}
	if m.newlyMastered {
		if next := m.nextChapterTitle(); next != "" {
			b.WriteString(t.Success.Render(fmt.Sprintf(m.deps.Msgs.StoryUnlockedFmt, next)))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.StoryDoneNext))
	return b.String()
}

// nextChapterTitle names the chapter after the current one, or "" at the end.
func (m Model) nextChapterTitle() string {
	for i, c := range m.deps.Chapters {
		if c.ID == m.chapter.ID && i+1 < len(m.deps.Chapters) {
			return m.deps.Chapters[i+1].Title
		}
	}
	return ""
}
