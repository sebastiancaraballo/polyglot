package content

import (
	"strings"
	"testing"
	"testing/fstest"
	"unicode"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

func TestLoadEmbeddedCourse(t *testing.T) {
	course, err := LoadEmbedded(DefaultPair)
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}

	if len(course.Lessons) < 2 {
		t.Errorf("got %d lessons, want at least 2", len(course.Lessons))
	}
	if len(course.Kana) == 0 {
		t.Fatal("expected embedded kana")
	}

	// Every card must have a unique, non-empty ID and a valid JLPT level.
	seen := make(map[string]bool)
	for _, lesson := range course.Lessons {
		for _, card := range lesson.Cards {
			if card.ID == "" {
				t.Errorf("lesson %q has a card with an empty ID", lesson.ID)
			}
			if seen[card.ID] {
				t.Errorf("duplicate card ID %q", card.ID)
			}
			seen[card.ID] = true
			if !card.JLPT.Valid() {
				t.Errorf("card %q has invalid JLPT %q", card.ID, card.JLPT)
			}
		}
	}

	// Both syllabaries should be present.
	types := make(map[model.KanaType]int)
	for _, k := range course.Kana {
		if !k.Type.Valid() {
			t.Errorf("kana %q has invalid type %q", k.Char, k.Type)
		}
		types[k.Type]++
	}
	if types[model.Hiragana] == 0 || types[model.Katakana] == 0 {
		t.Errorf("expected both hiragana and katakana, got %v", types)
	}
}

func TestEmbeddedCourseUsesPronunciationRomajiForLongVowels(t *testing.T) {
	course, err := LoadEmbedded(DefaultPair)
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}

	want := map[string]struct {
		romaji string
		input  string
	}{
		"おはよう":  {romaji: "ohayō", input: "ohayou"},
		"ありがとう": {romaji: "arigatō", input: "arigatou"},
		"さようなら": {romaji: "sayōnara", input: "sayounara"},
		"きゅう":   {romaji: "kyū", input: "kyuu"},
		"じゅう":   {romaji: "jū", input: "juu"},
	}

	found := make(map[string]bool, len(want))
	for _, lesson := range course.Lessons {
		for _, card := range lesson.Cards {
			expected, ok := want[card.JP]
			if !ok {
				continue
			}
			found[card.JP] = true
			if card.Romaji != expected.romaji {
				t.Errorf("%s romaji = %q, want %q", card.JP, card.Romaji, expected.romaji)
			}
			if !strings.Contains(card.Notes, expected.input) {
				t.Errorf("%s notes = %q, want input form %q", card.JP, card.Notes, expected.input)
			}
		}
	}
	for jp := range want {
		if !found[jp] {
			t.Errorf("missing embedded card %q", jp)
		}
	}
}

func TestEmbeddedKanaCategories(t *testing.T) {
	course, err := LoadEmbedded(DefaultPair)
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}

	type key struct {
		typ model.KanaType
		cat model.KanaCategory
	}
	counts := make(map[key]int)
	for _, k := range course.Kana {
		if !k.Category.Valid() {
			t.Errorf("kana %q has invalid category %q", k.Char, k.Category)
		}
		counts[key{k.Type, k.Category}]++
	}

	for _, typ := range []model.KanaType{model.Hiragana, model.Katakana} {
		if got := counts[key{typ, model.Base}]; got != 46 {
			t.Errorf("%s base count = %d, want 46", typ, got)
		}
		if got := counts[key{typ, model.Dakuten}]; got != 20 {
			t.Errorf("%s dakuten count = %d, want 20", typ, got)
		}
		if got := counts[key{typ, model.Handakuten}]; got != 5 {
			t.Errorf("%s handakuten count = %d, want 5", typ, got)
		}
		if got := counts[key{typ, model.Combo}]; got != 33 {
			t.Errorf("%s combo count = %d, want 33", typ, got)
		}
	}
}

func TestKanaCategoryDefaultsToBase(t *testing.T) {
	course, err := Load(validFS(), "xx")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := course.Kana[0].Category; got != model.Base {
		t.Errorf("omitted category = %q, want %q", got, model.Base)
	}
}

func TestKanaInvalidCategory(t *testing.T) {
	fsys := fstest.MapFS{
		"content/xx/lessons/01.yaml": &fstest.MapFile{Data: []byte(
			"id: a\ntitle: t\njlpt: N5\ncards:\n  - es: Hola\n    jp: こんにちは\n    romaji: konnichiwa\n",
		)},
		"content/xx/kana/h.yaml": &fstest.MapFile{Data: []byte(
			"type: hiragana\nitems:\n  - char: が\n    romaji: ga\n    category: bogus\n",
		)},
	}
	if _, err := Load(fsys, "xx"); err == nil {
		t.Fatal("expected invalid category error, got nil")
	}
}

func validFS() fstest.MapFS {
	return fstest.MapFS{
		"content/functions/core.yaml": &fstest.MapFile{Data: []byte(
			"functions:\n  - id: greet-daytime\n    cefr: A1\n    description: Saludar.\n",
		)},
		"content/xx/lessons/01.yaml": &fstest.MapFile{Data: []byte(
			"id: greetings\ntitle: Saludos\njlpt: N5\nfunctions: [greet-daytime]\ncards:\n  - es: Hola\n    jp: こんにちは\n    romaji: konnichiwa\n",
		)},
		"content/xx/kana/h.yaml": &fstest.MapFile{Data: []byte(
			"type: hiragana\nitems:\n  - char: こ\n    romaji: ko\n  - char: ん\n    romaji: n\n  - char: に\n    romaji: ni\n  - char: ち\n    romaji: chi\n  - char: は\n    romaji: wa\n",
		)},
	}
}

func TestLoadValid(t *testing.T) {
	course, err := Load(validFS(), "xx")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(course.Lessons) != 1 || len(course.Lessons[0].Cards) != 1 {
		t.Fatalf("unexpected lessons: %+v", course.Lessons)
	}
	if got := course.Lessons[0].Cards[0].ID; got != "greetings:1" {
		t.Errorf("card ID = %q, want %q", got, "greetings:1")
	}
}

func TestLoadValidationErrors(t *testing.T) {
	tests := map[string]struct {
		lesson string
	}{
		"missing jp": {
			lesson: "id: a\ntitle: t\njlpt: N5\ncards:\n  - es: Hola\n    romaji: hola\n",
		},
		"invalid jlpt": {
			lesson: "id: a\ntitle: t\njlpt: N9\ncards:\n  - es: Hola\n    jp: こんにちは\n    romaji: konnichiwa\n",
		},
		"missing id": {
			lesson: "title: t\njlpt: N5\ncards:\n  - es: Hola\n    jp: こんにちは\n    romaji: konnichiwa\n",
		},
		"no cards": {
			lesson: "id: a\ntitle: t\njlpt: N5\ncards: []\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			fsys := fstest.MapFS{
				"content/xx/lessons/01.yaml": &fstest.MapFile{Data: []byte(tc.lesson)},
				"content/xx/kana/h.yaml":     &fstest.MapFile{Data: []byte("type: hiragana\nitems:\n  - char: あ\n    romaji: a\n")},
			}
			if _, err := Load(fsys, "xx"); err == nil {
				t.Fatal("expected a validation error, got nil")
			}
		})
	}
}

func TestLoadDuplicateLessonID(t *testing.T) {
	lesson := "id: dup\ntitle: t\njlpt: N5\ncards:\n  - es: Hola\n    jp: こんにちは\n    romaji: konnichiwa\n"
	fsys := fstest.MapFS{
		"content/xx/lessons/01.yaml": &fstest.MapFile{Data: []byte(lesson)},
		"content/xx/lessons/02.yaml": &fstest.MapFile{Data: []byte(lesson)},
		"content/xx/kana/h.yaml":     &fstest.MapFile{Data: []byte("type: hiragana\nitems:\n  - char: あ\n    romaji: a\n")},
	}
	if _, err := Load(fsys, "xx"); err == nil {
		t.Fatal("expected duplicate lesson id error, got nil")
	}
}

func TestLoadMissingDirectories(t *testing.T) {
	if _, err := Load(fstest.MapFS{}, "xx"); err == nil {
		t.Fatal("expected error when no lessons exist, got nil")
	}
}

const validKana = "type: hiragana\nitems:\n  - char: あ\n    romaji: a\n"

// validPatternLesson provides a single vocab card "a:1" for pattern tests to
// reference as a slot filler.
const validPatternLesson = "id: a\ntitle: t\njlpt: N5\ncards:\n  - es: Hola\n    jp: あ\n    romaji: a\n"

func TestLoadCurriculumErrors(t *testing.T) {
	tests := map[string]fstest.MapFS{
		"unknown function ref": {
			"content/functions/core.yaml": file("functions:\n  - id: greet\n    cefr: A1\n    description: d\n"),
			"content/xx/lessons/01.yaml":  file("id: a\ntitle: t\njlpt: N5\nfunctions: [nope]\ncards:\n  - es: Hola\n    jp: あ\n    romaji: a\n"),
			"content/xx/kana/h.yaml":      file(validKana),
		},
		"invalid cefr": {
			"content/functions/core.yaml": file("functions:\n  - id: greet\n    cefr: X9\n    description: d\n"),
			"content/xx/lessons/01.yaml":  file("id: a\ntitle: t\njlpt: N5\ncards:\n  - es: Hola\n    jp: あ\n    romaji: a\n"),
			"content/xx/kana/h.yaml":      file(validKana),
		},
		"duplicate function id": {
			"content/functions/core.yaml": file("functions:\n  - id: greet\n    cefr: A1\n    description: d\n  - id: greet\n    cefr: A2\n    description: e\n"),
			"content/xx/lessons/01.yaml":  file("id: a\ntitle: t\njlpt: N5\ncards:\n  - es: Hola\n    jp: あ\n    romaji: a\n"),
			"content/xx/kana/h.yaml":      file(validKana),
		},
		"function missing description": {
			"content/functions/core.yaml": file("functions:\n  - id: greet\n    cefr: A1\n"),
			"content/xx/lessons/01.yaml":  file("id: a\ntitle: t\njlpt: N5\ncards:\n  - es: Hola\n    jp: あ\n    romaji: a\n"),
			"content/xx/kana/h.yaml":      file(validKana),
		},
		"negative freq": {
			"content/xx/lessons/01.yaml": file("id: a\ntitle: t\njlpt: N5\ncards:\n  - es: Hola\n    jp: あ\n    romaji: a\n    freq: -1\n"),
			"content/xx/kana/h.yaml":     file(validKana),
		},
		"card uses unteachable kana": {
			"content/xx/lessons/01.yaml": file("id: a\ntitle: t\njlpt: N5\ncards:\n  - es: Hola\n    jp: そ\n    romaji: so\n"),
			"content/xx/kana/h.yaml":     file(validKana),
		},
		"pattern missing id": {
			"content/xx/lessons/01.yaml": file(validPatternLesson),
			"content/xx/kana/h.yaml":     file(validKana),
			"content/xx/grammar/01.yaml": file("title: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [a:1]\n"),
		},
		"pattern missing frame": {
			"content/xx/lessons/01.yaml": file(validPatternLesson),
			"content/xx/kana/h.yaml":     file(validKana),
			"content/xx/grammar/01.yaml": file("id: p\ntitle: t\njlpt: N5\nslots:\n  - name: X\n    cards: [a:1]\n"),
		},
		"pattern invalid jlpt": {
			"content/xx/lessons/01.yaml": file(validPatternLesson),
			"content/xx/kana/h.yaml":     file(validKana),
			"content/xx/grammar/01.yaml": file("id: p\ntitle: t\njlpt: N9\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [a:1]\n"),
		},
		"duplicate pattern id": {
			"content/xx/lessons/01.yaml": file(validPatternLesson),
			"content/xx/kana/h.yaml":     file(validKana),
			"content/xx/grammar/01.yaml": file("id: p\ntitle: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [a:1]\n"),
			"content/xx/grammar/02.yaml": file("id: p\ntitle: t2\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [a:1]\n"),
		},
		"pattern slot missing name": {
			"content/xx/lessons/01.yaml": file(validPatternLesson),
			"content/xx/kana/h.yaml":     file(validKana),
			"content/xx/grammar/01.yaml": file("id: p\ntitle: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - cards: [a:1]\n"),
		},
		"pattern slot with no candidate cards": {
			"content/xx/lessons/01.yaml": file(validPatternLesson),
			"content/xx/kana/h.yaml":     file(validKana),
			"content/xx/grammar/01.yaml": file("id: p\ntitle: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: []\n"),
		},
		"frame placeholder not declared as a slot": {
			"content/xx/lessons/01.yaml": file(validPatternLesson),
			"content/xx/kana/h.yaml":     file(validKana),
			"content/xx/grammar/01.yaml": file("id: p\ntitle: t\njlpt: N5\nframe: \"{Y}\"\nslots:\n  - name: X\n    cards: [a:1]\n"),
		},
		"declared slot not used in frame": {
			"content/xx/lessons/01.yaml": file(validPatternLesson),
			"content/xx/kana/h.yaml":     file(validKana),
			"content/xx/grammar/01.yaml": file("id: p\ntitle: t\njlpt: N5\nframe: \"hola\"\nslots:\n  - name: X\n    cards: [a:1]\n"),
		},
		"slot default not among its candidate cards": {
			"content/xx/lessons/01.yaml": file(validPatternLesson),
			"content/xx/kana/h.yaml":     file(validKana),
			"content/xx/grammar/01.yaml": file("id: p\ntitle: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [a:1]\n    default: a:2\n"),
		},
		"pattern slot references unknown card id": {
			"content/xx/lessons/01.yaml": file(validPatternLesson),
			"content/xx/kana/h.yaml":     file(validKana),
			"content/xx/grammar/01.yaml": file("id: p\ntitle: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [nope:1]\n"),
		},
	}

	for name, fsys := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(fsys, "xx"); err == nil {
				t.Fatal("expected a validation error, got nil")
			}
		})
	}
}

func TestLoadPatternValid(t *testing.T) {
	fsys := fstest.MapFS{
		"content/xx/lessons/01.yaml": file(validPatternLesson),
		"content/xx/kana/h.yaml":     file(validKana),
		"content/xx/grammar/01.yaml": file(
			"id: p\ntitle: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [a:1]\n    default: a:1\n",
		),
	}
	course, err := Load(fsys, "xx")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(course.Patterns) != 1 {
		t.Fatalf("got %d patterns, want 1", len(course.Patterns))
	}
	p := course.Patterns[0]
	if len(p.Slots) != 1 || p.Slots[0].Default != "a:1" {
		t.Errorf("unexpected slots: %+v", p.Slots)
	}
}

func TestPatternSlotDefaultDefaultsToFirstCard(t *testing.T) {
	fsys := fstest.MapFS{
		"content/xx/lessons/01.yaml": file(validPatternLesson),
		"content/xx/kana/h.yaml":     file(validKana),
		"content/xx/grammar/01.yaml": file(
			"id: p\ntitle: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [a:1]\n",
		),
	}
	course, err := Load(fsys, "xx")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := course.Patterns[0].Slots[0].Default; got != "a:1" {
		t.Errorf("omitted default = %q, want %q", got, "a:1")
	}
}

func TestLoadNoGrammarIsNotAnError(t *testing.T) {
	course, err := Load(validFS(), "xx")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(course.Patterns) != 0 {
		t.Errorf("got %d patterns, want 0", len(course.Patterns))
	}
}

func TestEmbeddedGrammarPatterns(t *testing.T) {
	course, err := LoadEmbedded(DefaultPair)
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	if len(course.Patterns) == 0 {
		t.Fatal("expected at least one embedded grammar pattern")
	}
	cardSet := cardIDSet(course.Lessons)
	for _, p := range course.Patterns {
		if !p.JLPT.Valid() {
			t.Errorf("pattern %q has invalid JLPT %q", p.ID, p.JLPT)
		}
		if err := checkVocabCoverage(p, cardSet); err != nil {
			t.Errorf("pattern %q: %v", p.ID, err)
		}
		slotNames := make(map[string]bool, len(p.Slots))
		for _, s := range p.Slots {
			slotNames[s.Name] = true
		}
		if err := checkFramePlaceholders(p.Frame, slotNames); err != nil {
			t.Errorf("pattern %q: %v", p.ID, err)
		}
	}
}

func TestFreqOptional(t *testing.T) {
	course, err := Load(validFS(), "xx")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := course.Lessons[0].Cards[0].Freq; got != 0 {
		t.Errorf("omitted freq = %d, want 0", got)
	}
}

func TestEmbeddedLessonsReferenceKnownFunctions(t *testing.T) {
	course, err := LoadEmbedded(DefaultPair)
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	// Load already rejects unresolved function refs, so success implies every
	// referenced function exists. Assert each lesson declares at least one and
	// that its cards inherit them.
	for _, lesson := range course.Lessons {
		if len(lesson.Functions) == 0 {
			t.Errorf("lesson %q references no functions", lesson.ID)
		}
		for _, card := range lesson.Cards {
			if len(card.Functions) != len(lesson.Functions) {
				t.Errorf("card %q functions = %v, want lesson functions %v", card.ID, card.Functions, lesson.Functions)
			}
		}
	}
}

func TestEmbeddedKanaCoverage(t *testing.T) {
	course, err := LoadEmbedded(DefaultPair)
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	set := kanaSet(course.Kana)
	for _, lesson := range course.Lessons {
		for _, card := range lesson.Cards {
			if err := checkKanaCoverage(card.JP, set); err != nil {
				t.Errorf("card %q (%s): %v", card.ID, card.JP, err)
			}
		}
	}
}

func TestEmbeddedContentIsKanjiFree(t *testing.T) {
	// Kanji dependencies are deferred; the kana-coverage check skips non-kana
	// runes, so this pins the assumption that current content carries no kanji.
	course, err := LoadEmbedded(DefaultPair)
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	for _, lesson := range course.Lessons {
		for _, card := range lesson.Cards {
			for _, r := range card.JP {
				if unicode.Is(unicode.Han, r) {
					t.Errorf("card %q contains kanji %q; kanji support is not implemented yet", card.ID, string(r))
				}
			}
		}
	}
}

func file(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }

func TestEmbeddedFrequencyList(t *testing.T) {
	entries, err := LoadEmbeddedFrequency("ja")
	if err != nil {
		t.Fatalf("LoadEmbeddedFrequency: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected a non-empty frequency list")
	}

	seen := make(map[string]bool, len(entries))
	prevCount := entries[0].Count
	for i, e := range entries {
		if e.Rank != i+1 {
			t.Errorf("entry %d has rank %d, want %d (ranks must be contiguous and ascending)", i, e.Rank, i+1)
		}
		if e.Count <= 0 {
			t.Errorf("entry %q has non-positive count %d", e.Word, e.Count)
		}
		if e.Count > prevCount {
			t.Errorf("entry %q count %d exceeds previous %d (must be non-increasing)", e.Word, e.Count, prevCount)
		}
		prevCount = e.Count
		if e.Word == "" {
			t.Errorf("entry rank %d has a blank word", e.Rank)
		}
		if seen[e.Word] {
			t.Errorf("duplicate word %q in frequency list", e.Word)
		}
		seen[e.Word] = true
		if !isJapanese(e.Word) {
			t.Errorf("word %q is not Japanese (kana/kanji)", e.Word)
		}
	}
}

func isJapanese(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han) {
			return true
		}
	}
	return false
}
