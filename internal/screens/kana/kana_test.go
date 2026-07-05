package kana

import (
	"context"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func newTestModel(t *testing.T) (Model, storage.Storage, int64) {
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

	m := Model{
		deps: Deps{
			Theme:     ui.NewTheme(true),
			Msgs:      i18n.ES,
			Store:     store,
			ProfileID: profile.ID,
		},
		deck:    []model.KanaItem{{Char: "あ", Romaji: "a", Type: model.Hiragana}},
		options: []string{"a"},
		correct: 0,
	}
	return m, store, profile.ID
}

func TestPickerStartsFilteredSession(t *testing.T) {
	items := []model.KanaItem{
		{Char: "あ", Romaji: "a", Type: model.Hiragana, Category: model.Base},
		{Char: "が", Romaji: "ga", Type: model.Hiragana, Category: model.Dakuten},
		{Char: "ぱ", Romaji: "pa", Type: model.Hiragana, Category: model.Handakuten},
		{Char: "きゃ", Romaji: "kya", Type: model.Hiragana, Category: model.Combo},
		{Char: "ア", Romaji: "a", Type: model.Katakana, Category: model.Base},
	}
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Items: items})
	if !m.picking {
		t.Fatal("trainer should open on the group picker")
	}

	// Move to "Hiragana · Dakuten / Handakuten" (index 2) and start.
	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	got := tm.(Model)
	if got.picking {
		t.Fatal("confirming a group should start the session")
	}
	if len(got.deck) != 2 {
		t.Fatalf("dakuten/handakuten deck size = %d, want 2", len(got.deck))
	}
	for _, it := range got.deck {
		if it.Category != model.Dakuten && it.Category != model.Handakuten {
			t.Errorf("deck contains out-of-group item %q (%s)", it.Char, it.Category)
		}
	}
}

func TestSpaceAnswersKanaQuestion(t *testing.T) {
	m, _, _ := newTestModel(t)

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	got := next.(Model)
	if !got.answered {
		t.Fatal("space should answer the current kana question")
	}
}

func TestAnsweringAwardsXP(t *testing.T) {
	m, store, profileID := newTestModel(t)

	// Selecting the correct option (key "1") answers and should award XP.
	next, _ := m.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	got := next.(Model)
	if got.err != nil {
		t.Fatalf("answering returned error: %v", got.err)
	}

	stats, err := store.GetStats(context.Background(), profileID)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.XP <= 0 {
		t.Errorf("XP after a correct answer = %d, want > 0", stats.XP)
	}
}

// gateItems is a minimal two-syllabary set whose base gojūon is one kana each,
// so a single mastered kana makes a syllabary fluent in tests.
var gateItems = []model.KanaItem{
	{Char: "あ", Romaji: "a", Type: model.Hiragana, Category: model.Base},
	{Char: "ア", Romaji: "a", Type: model.Katakana, Category: model.Base},
}

func katakanaGroupIndex(t *testing.T, m Model) int {
	t.Helper()
	for i, g := range m.groups {
		if strings.HasPrefix(g.label, i18n.ES.KatakanaLabel) {
			return i
		}
	}
	t.Fatal("no katakana group in picker")
	return 0
}

func TestKatakanaLockedUntilHiraganaFluent(t *testing.T) {
	store, profileID := newStore(t)
	// Skip the first-time intro so the picker (and its locked-group guard) is
	// what the confirm key exercises.
	if err := store.SetKanaOnboarded(context.Background(), profileID); err != nil {
		t.Fatalf("SetKanaOnboarded: %v", err)
	}
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: gateItems})

	idx := katakanaGroupIndex(t, m)
	if !m.groups[idx].locked {
		t.Fatal("katakana should be locked before hiragana fluency")
	}

	// Confirming a locked group must not start a session.
	m.groupCursor = idx
	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !tm.(Model).picking {
		t.Fatal("confirming a locked group should not start a session")
	}
}

func TestKatakanaUnlocksAfterHiraganaMastered(t *testing.T) {
	store, profileID := newStore(t)
	if err := store.SaveKanaProgress(context.Background(), profileID,
		model.KanaProgress{Char: "あ", Mastered: true}); err != nil {
		t.Fatalf("SaveKanaProgress: %v", err)
	}

	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: gateItems})
	idx := katakanaGroupIndex(t, m)
	if m.groups[idx].locked {
		t.Fatal("katakana should unlock once hiragana is fluent")
	}
}

func TestIntroShownOnFirstVisit(t *testing.T) {
	store, profileID := newStore(t)
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: gateItems})
	if !m.intro {
		t.Fatal("a first-time visitor should see the intro")
	}
	if m.picking {
		t.Fatal("the picker should not show while the intro is up")
	}
}

func TestIntroSkippedWhenAlreadyOnboarded(t *testing.T) {
	store, profileID := newStore(t)
	if err := store.SetKanaOnboarded(context.Background(), profileID); err != nil {
		t.Fatalf("SetKanaOnboarded: %v", err)
	}
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: gateItems})
	if m.intro {
		t.Fatal("a returning learner should not see the intro again")
	}
	if !m.picking {
		t.Fatal("the picker should show immediately for a returning learner")
	}
}

func TestDismissingIntroPersistsAndShowsPicker(t *testing.T) {
	store, profileID := newStore(t)
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: gateItems})

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := next.(Model)
	if got.intro {
		t.Fatal("confirming should dismiss the intro")
	}
	if !got.picking {
		t.Fatal("dismissing the intro should fall through to the picker")
	}

	prof, err := store.GetProfile(context.Background(), profileID)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if !prof.KanaOnboarded {
		t.Fatal("dismissing the intro should persist that it was seen")
	}

	// A subsequent visit no longer shows the intro.
	again := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: gateItems})
	if again.intro {
		t.Fatal("the intro should not reappear on a later visit")
	}
}

func TestLockedGroupHintShowsLiveProgress(t *testing.T) {
	store, profileID := newStore(t)
	if err := store.SetKanaOnboarded(context.Background(), profileID); err != nil {
		t.Fatalf("SetKanaOnboarded: %v", err)
	}
	m := New(Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: gateItems})
	m.groupCursor = katakanaGroupIndex(t, m) // a locked katakana group

	view := m.pickerView()
	// gateItems has one hiragana base kana, none mastered → "0/1".
	if !strings.Contains(view, "0/1") {
		t.Fatalf("locked-group hint should show live hiragana progress; view:\n%s", view)
	}
}

func allGroupIndex(t *testing.T, m Model) int {
	t.Helper()
	for i, g := range m.groups {
		if g.label == i18n.ES.KanaGroupAll {
			return i
		}
	}
	t.Fatal("no \"all\" group in picker")
	return 0
}

func TestAllGroupGatedLikeKatakana(t *testing.T) {
	// The "Todo" (All) group spans both syllabaries, so it must honor the same
	// hiragana→katakana gate as the katakana groups: locked before hiragana
	// fluency, unlocked after. Otherwise it lets a learner practice katakana
	// before the gate opens.
	store, profileID := newStore(t)
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: gateItems})
	idx := allGroupIndex(t, m)
	if !m.groups[idx].locked {
		t.Fatal("\"all\" group should be locked before hiragana fluency")
	}

	// Confirming the locked group must not start a session.
	m.groupCursor = idx
	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !tm.(Model).picking {
		t.Fatal("confirming the locked \"all\" group should not start a session")
	}

	// Once hiragana is fluent, the gate opens for the "all" group too.
	if err := store.SaveKanaProgress(context.Background(), profileID,
		model.KanaProgress{Char: "あ", Mastered: true}); err != nil {
		t.Fatalf("SaveKanaProgress: %v", err)
	}
	m2 := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: gateItems})
	if m2.groups[allGroupIndex(t, m2)].locked {
		t.Fatal("\"all\" group should unlock once hiragana is fluent")
	}
}

func TestAnsweringPersistsKanaProgress(t *testing.T) {
	store, profileID := newStore(t)
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: gateItems})

	// Start the hiragana base group and answer the only card correctly.
	m.groupCursor = 1 // "Hiragana · Básico"
	m = m.startSession()
	m.selected = m.correct
	m = m.reveal()

	got, err := store.GetKanaProgress(context.Background(), profileID)
	if err != nil {
		t.Fatalf("GetKanaProgress: %v", err)
	}
	if got["あ"].Attempts == 0 {
		t.Fatalf("answering did not persist kana progress: %+v", got)
	}
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

func TestKanaTilePositionStableAfterAnswer(t *testing.T) {
	m := Model{
		deps: Deps{
			Theme: ui.PlainTheme(),
			Msgs:  i18n.ES,
		},
		deck: []model.KanaItem{{Char: "ミ", Romaji: "mi", Type: model.Katakana}},
		options: []string{
			"mu",
			"sa",
			"mi",
			"ne",
		},
		correct:  2,
		selected: 0,
		width:    80,
		height:   24,
	}

	before := kanaTileColumn(t, m.questionView())
	m.selected = 3
	m.answered = true
	after := kanaTileColumn(t, m.questionView())

	if before != after {
		t.Fatalf("tile column moved from %d to %d after answering", before, after)
	}
}

func kanaTileColumn(t *testing.T, view string) int {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if idx := strings.IndexRune(line, '╭'); idx >= 0 {
			return lipgloss.Width(line[:idx])
		}
	}
	t.Fatal("view does not contain kana tile")
	return 0
}

func reverseFixtureItems() []model.KanaItem {
	return []model.KanaItem{
		{Char: "あ", Romaji: "a", Type: model.Hiragana, Category: model.Base},
		{Char: "い", Romaji: "i", Type: model.Hiragana, Category: model.Base},
		{Char: "う", Romaji: "u", Type: model.Hiragana, Category: model.Base},
		{Char: "え", Romaji: "e", Type: model.Hiragana, Category: model.Base},
		{Char: "お", Romaji: "o", Type: model.Hiragana, Category: model.Base},
	}
}

func TestArrowTogglesDirectionInPicker(t *testing.T) {
	store, profileID := newStore(t)
	if err := store.SetKanaOnboarded(context.Background(), profileID); err != nil {
		t.Fatalf("SetKanaOnboarded: %v", err)
	}
	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: reverseFixtureItems()})
	if m.reverse {
		t.Fatal("trainer should default to recognition (forward) direction")
	}
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if !next.(Model).reverse {
		t.Fatal("→ should flip to reverse direction")
	}
	next, _ = next.(Model).Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if next.(Model).reverse {
		t.Fatal("← should flip direction back to forward")
	}
}

func TestReverseSessionAsksForTheGlyph(t *testing.T) {
	m := Model{
		deps:    Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES},
		reverse: true,
	}
	m.rng = rand.New(rand.NewSource(1))
	// Build a reverse session over the fixture: the answer pool is the glyphs.
	items := reverseFixtureItems()
	m.deps.Items = items
	m.groups = buildGroups(i18n.ES, m.gate)
	m.groupCursor = allGroupIndex(t, m)
	// All group is unlocked only once hiragana is fluent; force it open for the
	// test by matching directly instead of going through the gate.
	m.deck = items
	m.pool = make([]string, 0, len(items))
	for _, it := range items {
		m.pool = append(m.pool, m.answer(it))
	}
	m = m.setQuestion()

	// The correct option and every distractor must be a glyph from the pool.
	glyphs := map[string]bool{}
	for _, it := range items {
		glyphs[it.Char] = true
	}
	for _, opt := range m.options {
		if !glyphs[opt] {
			t.Errorf("reverse option %q is not a kana glyph", opt)
		}
	}
	if m.options[m.correct] != m.deck[0].Char {
		t.Errorf("reverse correct option = %q, want the deck glyph %q", m.options[m.correct], m.deck[0].Char)
	}
	// The prompt shows the romaji, not the glyph.
	view := m.questionView()
	if !strings.Contains(view, m.deps.Msgs.KanaPromptReverse) {
		t.Errorf("reverse question should use the recall prompt; view:\n%s", view)
	}
}

func TestReverseAnswerRecordsMasteryByCharacter(t *testing.T) {
	store, profileID := newStore(t)
	items := reverseFixtureItems()
	m := Model{
		deps:    Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profileID, Items: items},
		reverse: true,
		rng:     rand.New(rand.NewSource(1)),
		deck:    []model.KanaItem{items[0]}, // あ
		pool:    []string{"あ", "い", "う", "え"},
	}
	m = m.setQuestion()
	m.selected = m.correct // answer correctly
	m = m.reveal()

	// Mastery is keyed by the character, direction-agnostic: the reverse answer
	// advances the same per-character streak the forward drill uses.
	got, err := store.GetKanaProgress(context.Background(), profileID)
	if err != nil {
		t.Fatalf("GetKanaProgress: %v", err)
	}
	p, ok := got["あ"]
	if !ok {
		t.Fatal("reverse answer should record progress under the glyph あ")
	}
	if p.Streak == 0 && p.Attempts == 0 {
		t.Fatal("reverse answer should advance the character's mastery streak")
	}
}

func TestQuestionViewFitsFrameBothDirections(t *testing.T) {
	th := ui.PlainTheme()
	const termW, termH = 80, 30
	maxW := ui.FrameContentWidth(th, termW)
	maxH := ui.FrameContentHeight(th, termH)
	items := reverseFixtureItems()

	for _, reverse := range []bool{false, true} {
		m := Model{
			deps:    Deps{Theme: th, Msgs: i18n.ES, Items: items},
			reverse: reverse,
			rng:     rand.New(rand.NewSource(1)),
			deck:    []model.KanaItem{items[0]},
			width:   termW, height: termH,
		}
		m.pool = make([]string, 0, len(items))
		for _, it := range items {
			m.pool = append(m.pool, m.answer(it))
		}
		m = m.setQuestion()
		for _, answered := range []bool{false, true} {
			m.answered = answered
			m.selected = 1
			content := m.questionView()
			if w := lipgloss.Width(content); w > maxW {
				t.Errorf("[WIDTH] reverse=%v answered=%v: %d > %d", reverse, answered, w, maxW)
			}
			if h := lipgloss.Height(content); h > maxH {
				t.Errorf("[HEIGHT] reverse=%v answered=%v: %d > %d", reverse, answered, h, maxH)
			}
		}
		// The picker (with the direction line) must also fit.
		m.picking = true
		m.groups = buildGroups(i18n.ES, m.gate)
		p := m.pickerView()
		if w := lipgloss.Width(p); w > maxW {
			t.Errorf("[WIDTH] picker reverse=%v: %d > %d", reverse, w, maxW)
		}
		if h := lipgloss.Height(p); h > maxH {
			t.Errorf("[HEIGHT] picker reverse=%v: %d > %d", reverse, h, maxH)
		}
	}
}
