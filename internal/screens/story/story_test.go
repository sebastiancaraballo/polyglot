package story

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

// testChapter is a small 4-beat chapter for tests: narration, dialogue, one
// vocab practice beat, and a closing dialogue.
var testChapter = model.Chapter{
	ID:    "test-chapter",
	Title: "Capítulo de prueba",
	Beats: []model.Beat{
		{Kind: model.Narration, Place: "Aquí", Source: "Es de día.", JP: "昼です。"},
		{Kind: model.Dialogue, Speaker: "Yui", Source: "Hola.", JP: "こんにちは。"},
		{Kind: model.Practice, Practice: model.PracticeVocab, RefID: "test-lesson"},
		{Kind: model.Dialogue, Speaker: "Yui", Source: "Adiós.", JP: "さようなら。"},
	},
}

// testChapter2 sits behind testChapter's mastery gate in the picker.
var testChapter2 = model.Chapter{
	ID:    "test-chapter-2",
	Title: "Segundo capítulo",
	Beats: []model.Beat{
		{Kind: model.Narration, Source: "Sigue.", JP: "続く。"},
	},
}

// noPracticeChapter has nothing to evaluate: completing it masters it.
var noPracticeChapter = model.Chapter{
	ID:    "no-practice",
	Title: "Sin práctica",
	Beats: []model.Beat{
		{Kind: model.Narration, Source: "Solo texto.", JP: "テキストだけ。"},
	},
}

var testLessons = []model.Lesson{
	{ID: "test-lesson", Title: "t", JLPT: model.N5, Cards: []model.Card{
		{ID: "test-lesson:1", Source: "Hola", JP: "こんにちは", Romaji: "konnichiwa"},
		{ID: "test-lesson:2", Source: "Adiós", JP: "さようなら", Romaji: "sayounara"},
	}},
}

var testKana = []model.KanaItem{
	{Char: "あ", Romaji: "a", Type: model.Hiragana, Category: model.Base},
	{Char: "い", Romaji: "i", Type: model.Hiragana, Category: model.Base},
}

func newStore(t *testing.T) (storage.Storage, int64) {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	profile, err := store.CreateProfile(context.Background(), "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	return store, profile.ID
}

func testDeps(store storage.Storage, profileID int64) Deps {
	return Deps{
		Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID,
		Chapters: []model.Chapter{testChapter, testChapter2}, Lessons: testLessons, Kana: testKana,
	}
}

// finishBeats plays testChapter's four beats to the end, answering its
// practice beat correctly, leaving the model at the challenge intro.
func finishBeats(t *testing.T, m Model) Model {
	t.Helper()
	m = m.advance() // narration -> dialogue
	m = m.advance() // dialogue -> practice
	m.selected = m.correct
	m = m.reveal()
	m = m.advance() // practice -> closing dialogue
	m = m.advance() // -> finished, challenge starts
	if m.challenge == nil || !m.challengeIntro {
		t.Fatal("finishing the last beat should open the challenge intro")
	}
	m.challengeIntro = false
	return m.setChallengeQuestion()
}

// answerChallengeQuestion reveals and records one challenge answer.
func answerChallengeQuestion(m Model, correct bool) Model {
	if correct {
		m.selected = m.correct
	} else {
		m.selected = (m.correct + 1) % len(m.options)
	}
	m = m.reveal()
	return m.recordChallengeAnswer()
}

// runChallenge answers every remaining challenge question with the given
// correctness.
func runChallenge(m Model, correct bool) Model {
	for !m.challengeFinished() {
		m = answerChallengeQuestion(m, correct)
	}
	return m
}

func TestEmptyViewWhenNoChapters(t *testing.T) {
	store, profileID := newStore(t)
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID})
	if !strings.Contains(m.View().Content, i18n.ES.StoryEmpty) {
		t.Error("empty state should show the StoryEmpty message")
	}
}

func TestPickerListsChapters(t *testing.T) {
	store, profileID := newStore(t)
	m := New(testDeps(store, profileID))
	if !m.picking {
		t.Fatal("story runner should open on the chapter picker")
	}
	if len(m.chapters) != 2 {
		t.Fatalf("chapters = %d, want 2", len(m.chapters))
	}
	if m.chapters[0].locked {
		t.Error("the first chapter must never be locked")
	}
	if !m.chapters[1].locked {
		t.Error("the second chapter should be locked until the first is mastered")
	}
}

func TestStartChapterEntersFirstBeat(t *testing.T) {
	store, profileID := newStore(t)
	m := New(testDeps(store, profileID))
	m = m.startChapter(0)
	if m.picking {
		t.Fatal("confirming a chapter should start it")
	}
	if m.beatIndex != 0 {
		t.Fatalf("beatIndex = %d, want 0", m.beatIndex)
	}
}

func TestAdvanceThroughNarrationAndDialogueReachesPractice(t *testing.T) {
	store, profileID := newStore(t)
	m := New(testDeps(store, profileID))
	m = m.startChapter(0)

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // narration -> dialogue
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // dialogue -> practice

	got := tm.(Model)
	if got.beatIndex != 2 {
		t.Fatalf("beatIndex = %d, want 2 (the practice beat)", got.beatIndex)
	}
	if len(got.options) == 0 {
		t.Fatal("entering a practice beat should build multiple-choice options")
	}
}

func TestAnsweringVocabPracticePersistsCardStateAndXP(t *testing.T) {
	store, profileID := newStore(t)
	m := New(testDeps(store, profileID))
	m = m.startChapter(0)
	m = m.advance() // narration -> dialogue
	m = m.advance() // dialogue -> practice

	m.selected = m.correct
	m = m.reveal()
	if m.err != nil {
		t.Fatalf("reveal returned error: %v", m.err)
	}

	stats, err := store.GetStats(context.Background(), profileID)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.XP <= 0 {
		t.Errorf("XP after a correct answer = %d, want > 0", stats.XP)
	}

	state, err := store.GetCardState(context.Background(), profileID, m.practiceCard.ID)
	if err != nil {
		t.Fatalf("GetCardState: %v", err)
	}
	if state.Reps == 0 {
		t.Fatal("answering correctly should record at least one rep")
	}
}

func TestAnsweringKanaPracticePersistsKanaProgress(t *testing.T) {
	store, profileID := newStore(t)
	kanaChapter := model.Chapter{
		ID: "kana-chapter", Title: "t",
		Beats: []model.Beat{{Kind: model.Practice, Practice: model.PracticeKana, RefID: "hiragana"}},
	}
	deps := Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID,
		Chapters: []model.Chapter{kanaChapter}, Kana: testKana}
	m := New(deps)
	m = m.startChapter(0)

	m.selected = m.correct
	m = m.reveal()
	if m.err != nil {
		t.Fatalf("reveal returned error: %v", m.err)
	}

	progress, err := store.GetKanaProgress(context.Background(), profileID)
	if err != nil {
		t.Fatalf("GetKanaProgress: %v", err)
	}
	if progress[m.practiceKana.Char].Attempts == 0 {
		t.Fatalf("answering did not persist kana progress: %+v", progress)
	}
}

func TestFinishingBeatsCompletesChapterAndStartsChallenge(t *testing.T) {
	store, profileID := newStore(t)
	m := New(testDeps(store, profileID))
	m = m.startChapter(0)
	m = finishBeats(t, m)

	progress, err := store.GetStoryProgress(context.Background(), profileID)
	if err != nil {
		t.Fatalf("GetStoryProgress: %v", err)
	}
	if !progress["test-chapter"].Completed {
		t.Fatalf("chapter should be marked completed: %+v", progress["test-chapter"])
	}
	if progress["test-chapter"].Mastered {
		t.Fatal("completion alone must not mark the chapter mastered")
	}
	if len(m.options) == 0 {
		t.Fatal("the challenge should have built its first question")
	}
}

func TestFailingChallengeKeepsNextChapterLocked(t *testing.T) {
	store, profileID := newStore(t)
	m := New(testDeps(store, profileID))
	m = m.startChapter(0)
	m = finishBeats(t, m)
	m = runChallenge(m, false)

	if m.challengePassed() {
		t.Fatal("all-wrong answers must not pass the challenge")
	}
	m = m.refreshChapters()
	if !m.chapters[1].locked {
		t.Fatal("failing the challenge should keep the next chapter locked")
	}
	progress, _ := store.GetStoryProgress(context.Background(), profileID)
	if progress["test-chapter"].Mastered {
		t.Fatal("a failed challenge must not persist mastery")
	}
}

func TestPassingChallengeMastersAndUnlocksNextChapter(t *testing.T) {
	store, profileID := newStore(t)
	m := New(testDeps(store, profileID))
	m = m.startChapter(0)
	m = finishBeats(t, m)
	m = runChallenge(m, true)

	if !m.challengePassed() {
		t.Fatal("all-correct answers should pass the challenge")
	}
	progress, err := store.GetStoryProgress(context.Background(), profileID)
	if err != nil {
		t.Fatalf("GetStoryProgress: %v", err)
	}
	if !progress["test-chapter"].Mastered {
		t.Fatalf("passing the challenge should persist mastery: %+v", progress["test-chapter"])
	}

	// Confirming on the done screen returns to a picker that reflects the
	// fresh mastery without rebuilding the screen.
	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := tm.(Model)
	if !got.picking {
		t.Fatal("confirming on the done screen should return to the picker")
	}
	if !got.chapters[0].progress.Mastered {
		t.Fatal("picker should show the chapter mastered")
	}
	if got.chapters[1].locked {
		t.Fatal("mastering chapter 1 should unlock chapter 2")
	}
}

func TestChallengeRetryRebuildsQuestions(t *testing.T) {
	store, profileID := newStore(t)
	m := New(testDeps(store, profileID))
	m = m.startChapter(0)
	m = finishBeats(t, m)
	m = runChallenge(m, false)

	// Confirm on the fail view retries with a fresh challenge.
	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := tm.(Model)
	if got.challenge == nil || !got.challengeIntro {
		t.Fatal("confirming on the fail view should restart the challenge at its intro")
	}
	if got.challengeRight != 0 || got.challengeIdx != 0 {
		t.Fatal("a retry should reset the challenge tally")
	}
}

func TestNoPracticeChapterAutoMasters(t *testing.T) {
	store, profileID := newStore(t)
	deps := Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID,
		Chapters: []model.Chapter{noPracticeChapter}, Lessons: testLessons, Kana: testKana}
	m := New(deps)
	m = m.startChapter(0)
	m = m.advance() // the single narration beat -> finished

	if m.challenge != nil {
		t.Fatal("a chapter with no practice beats should skip the challenge")
	}
	progress, _ := store.GetStoryProgress(context.Background(), profileID)
	if !progress["no-practice"].Mastered {
		t.Fatalf("completing a no-practice chapter should master it: %+v", progress["no-practice"])
	}
}

// Regression: replaying an already-mastered chapter must never overwrite its
// persisted mastery, even though advance() writes fresh progress each beat.
func TestReplayPreservesMastered(t *testing.T) {
	store, profileID := newStore(t)
	if err := store.SaveStoryProgress(context.Background(), profileID,
		model.StoryProgress{ChapterID: "test-chapter", BeatIndex: 4, Completed: true, Mastered: true}); err != nil {
		t.Fatalf("SaveStoryProgress: %v", err)
	}

	m := New(testDeps(store, profileID))
	m = m.startChapter(0)
	_ = m.advance() // one beat into the replay persists fresh progress

	progress, _ := store.GetStoryProgress(context.Background(), profileID)
	if !progress["test-chapter"].Mastered {
		t.Fatal("replaying a mastered chapter must not revoke its mastery")
	}
}

func TestConfirmOnLockedChapterDoesNothing(t *testing.T) {
	store, profileID := newStore(t)
	m := New(testDeps(store, profileID))
	m.chapterCur = 1 // the locked second chapter

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !tm.(Model).picking {
		t.Fatal("confirming a locked chapter must not start it")
	}
}

func TestLockedChapterRendersHintAndGateNote(t *testing.T) {
	store, profileID := newStore(t)
	m := New(testDeps(store, profileID))
	m.chapterCur = 1

	view := m.pickerView()
	if !strings.Contains(view, "⊘") {
		t.Error("locked chapter should render the lock glyph")
	}
	if !strings.Contains(view, testChapter.Title) || !strings.Contains(view, "Supera el reto") {
		t.Errorf("cursor on a locked chapter should explain the unlock, naming the previous chapter; view:\n%s", view)
	}
	if !strings.Contains(view, i18n.ES.StoryGateNote) {
		t.Error("picker should always state the gating rule")
	}
}

func TestResumeSkipsAlreadySeenBeats(t *testing.T) {
	store, profileID := newStore(t)
	if err := store.SaveStoryProgress(context.Background(), profileID,
		model.StoryProgress{ChapterID: "test-chapter", BeatIndex: 2}); err != nil {
		t.Fatalf("SaveStoryProgress: %v", err)
	}

	m := New(testDeps(store, profileID))
	m = m.startChapter(0)
	if m.beatIndex != 2 {
		t.Fatalf("beatIndex = %d, want 2 (resumed)", m.beatIndex)
	}
}

func TestCompletedChapterRestartsFromScratch(t *testing.T) {
	store, profileID := newStore(t)
	if err := store.SaveStoryProgress(context.Background(), profileID,
		model.StoryProgress{ChapterID: "test-chapter", BeatIndex: 4, Completed: true}); err != nil {
		t.Fatalf("SaveStoryProgress: %v", err)
	}

	m := New(testDeps(store, profileID))
	m = m.startChapter(0)
	if m.beatIndex != 0 {
		t.Fatalf("beatIndex = %d, want 0 (replaying a completed chapter)", m.beatIndex)
	}
}
