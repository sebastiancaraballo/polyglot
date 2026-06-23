package content

import (
	"strings"
	"testing"
	"testing/fstest"

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
		"content/xx/lessons/01.yaml": &fstest.MapFile{Data: []byte(
			"id: greetings\ntitle: Saludos\njlpt: N5\ncards:\n  - es: Hola\n    jp: こんにちは\n    romaji: konnichiwa\n",
		)},
		"content/xx/kana/h.yaml": &fstest.MapFile{Data: []byte(
			"type: hiragana\nitems:\n  - char: あ\n    romaji: a\n",
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
