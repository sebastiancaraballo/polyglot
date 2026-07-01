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
		Chapters: []model.Chapter{testChapter}, Lessons: testLessons, Kana: testKana,
	}
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
	if len(m.chapters) != 1 {
		t.Fatalf("chapters = %d, want 1", len(m.chapters))
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

func TestChapterCompletionMarksProgress(t *testing.T) {
	store, profileID := newStore(t)
	m := New(testDeps(store, profileID))
	m = m.startChapter(0)
	m = m.advance() // -> dialogue
	m = m.advance() // -> practice
	m.selected = m.correct
	m = m.reveal()
	m = m.advance() // -> closing dialogue
	m = m.advance() // -> finished

	if !m.finished() {
		t.Fatal("chapter should be finished after its last beat")
	}

	progress, err := store.GetStoryProgress(context.Background(), profileID)
	if err != nil {
		t.Fatalf("GetStoryProgress: %v", err)
	}
	if !progress["test-chapter"].Completed {
		t.Fatalf("chapter should be marked completed: %+v", progress["test-chapter"])
	}
}

func TestPickerRefreshesAfterCompletingChapter(t *testing.T) {
	store, profileID := newStore(t)
	m := New(testDeps(store, profileID))
	m = m.startChapter(0)
	m = m.advance() // -> dialogue
	m = m.advance() // -> practice
	m.selected = m.correct
	m = m.reveal()
	m = m.advance() // -> closing dialogue
	m = m.advance() // -> finished

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // finished -> back to picker

	got := tm.(Model)
	if !got.picking {
		t.Fatal("confirming on the finished screen should return to the picker")
	}
	if !got.chapters[0].progress.Completed {
		t.Fatal("picker should reflect the just-completed chapter without rebuilding the screen")
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
