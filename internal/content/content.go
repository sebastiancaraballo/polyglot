package content

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// Course is the fully loaded and validated content for a single language pair.
type Course struct {
	Pair     string
	Lessons  []model.Lesson
	Kana     []model.KanaItem
	Patterns []model.Pattern
	Chapters []model.Chapter
}

// Load reads, parses, and validates the course for pair from fsys. Paths are
// resolved as content/<pair>/{lessons,kana,grammar,story}/*.yaml so that the
// same loader works for the embedded FS and for test filesystems. The
// language-agnostic function catalog under content/functions/*.yaml is loaded
// once and used to resolve the communicative functions each lesson references.
func Load(fsys fs.FS, pair string) (*Course, error) {
	catalog, err := loadFunctions(fsys)
	if err != nil {
		return nil, err
	}
	lessons, err := loadLessons(fsys, pair, catalog)
	if err != nil {
		return nil, err
	}
	kana, err := loadKana(fsys, pair)
	if err != nil {
		return nil, err
	}
	patterns, err := loadPatterns(fsys, pair)
	if err != nil {
		return nil, err
	}
	// Every kana a card depends on must be teachable (present in the kana tables).
	set := kanaSet(kana)
	for _, lesson := range lessons {
		for _, card := range lesson.Cards {
			if err := checkKanaCoverage(card.JP, set); err != nil {
				return nil, fmt.Errorf("card %q %w", card.ID, err)
			}
		}
	}
	// Every pattern's slot fillers must be vocab the learner is actually taught
	// ("words before sentences").
	cardSet := cardIDSet(lessons)
	for _, p := range patterns {
		if err := checkVocabCoverage(p, cardSet); err != nil {
			return nil, fmt.Errorf("pattern %q %w", p.ID, err)
		}
	}

	chapters, err := loadChapters(fsys, pair)
	if err != nil {
		return nil, err
	}
	// Every practice beat must reference vocabulary or kana that actually exists.
	lessonIDs := lessonIDSet(lessons)
	kanaTypes := kanaTypesPresent(kana)
	for _, c := range chapters {
		if err := checkStoryCoverage(c, lessonIDs, kanaTypes); err != nil {
			return nil, fmt.Errorf("chapter %q: %w", c.ID, err)
		}
	}
	// A chapter may only practice material it (or an earlier chapter) presented.
	if err := checkStoryPresentation(chapters); err != nil {
		return nil, err
	}

	// Backfill each card's frequency rank from the target language's list, when
	// one ships (content/<lang>/frequency.tsv). Like grammar and story content,
	// the list is optional: a pair without one simply has unranked cards.
	if lang := targetLang(pair); lang != "" {
		entries, err := loadFrequency(fsys, lang)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			// No list for this language: leave cards unranked.
		case err != nil:
			return nil, err
		default:
			backfillFreq(lessons, freqIndex(entries))
		}
	}

	return &Course{Pair: pair, Lessons: lessons, Kana: kana, Patterns: patterns, Chapters: chapters}, nil
}

// targetLang extracts the target language from a pair like "es-ja" ("" when
// the pair has no source-target form).
func targetLang(pair string) string {
	if i := strings.LastIndexByte(pair, '-'); i >= 0 {
		return pair[i+1:]
	}
	return ""
}

// lessonFile mirrors the on-disk YAML shape of a lesson. The source-language key
// is "es" for the v1 Spanish → Japanese pair.
type lessonFile struct {
	ID        string   `yaml:"id"`
	Title     string   `yaml:"title"`
	JLPT      string   `yaml:"jlpt"`
	Functions []string `yaml:"functions"`
	Cards     []struct {
		Source string `yaml:"es"`
		JP     string `yaml:"jp"`
		Romaji string `yaml:"romaji"`
		Notes  string `yaml:"notes"`
		Freq   int    `yaml:"freq"`
	} `yaml:"cards"`
}

func loadLessons(fsys fs.FS, pair string, catalog model.FunctionCatalog) ([]model.Lesson, error) {
	dir := path.Join("content", pair, "lessons")
	files, err := fs.Glob(fsys, path.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("glob lessons: %w", err)
	}
	sort.Strings(files)

	seen := make(map[string]bool)
	lessons := make([]model.Lesson, 0, len(files))
	for _, file := range files {
		lesson, err := parseLesson(fsys, file, catalog)
		if err != nil {
			return nil, err
		}
		if seen[lesson.ID] {
			return nil, fmt.Errorf("%s: duplicate lesson id %q", file, lesson.ID)
		}
		seen[lesson.ID] = true
		lessons = append(lessons, lesson)
	}
	if len(lessons) == 0 {
		return nil, fmt.Errorf("no lessons found in %s", dir)
	}
	return lessons, nil
}

func parseLesson(fsys fs.FS, file string, catalog model.FunctionCatalog) (model.Lesson, error) {
	data, err := fs.ReadFile(fsys, file)
	if err != nil {
		return model.Lesson{}, fmt.Errorf("read %s: %w", file, err)
	}
	var lf lessonFile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return model.Lesson{}, fmt.Errorf("parse %s: %w", file, err)
	}
	if lf.ID == "" {
		return model.Lesson{}, fmt.Errorf("%s: missing lesson id", file)
	}
	if lf.Title == "" {
		return model.Lesson{}, fmt.Errorf("%s: missing lesson title", file)
	}
	level := model.JLPT(lf.JLPT)
	if !level.Valid() {
		return model.Lesson{}, fmt.Errorf("%s: invalid jlpt level %q", file, lf.JLPT)
	}
	for _, fn := range lf.Functions {
		if _, ok := catalog.Lookup(fn); !ok {
			return model.Lesson{}, fmt.Errorf("%s: unknown function %q", file, fn)
		}
	}
	if len(lf.Cards) == 0 {
		return model.Lesson{}, fmt.Errorf("%s: lesson has no cards", file)
	}

	lesson := model.Lesson{ID: lf.ID, Title: lf.Title, JLPT: level, Functions: lf.Functions}
	for i, c := range lf.Cards {
		if c.Source == "" || c.JP == "" || c.Romaji == "" {
			return model.Lesson{}, fmt.Errorf("%s: card %d is missing es, jp, or romaji", file, i+1)
		}
		if c.Freq < 0 {
			return model.Lesson{}, fmt.Errorf("%s: card %d has negative freq %d", file, i+1, c.Freq)
		}
		lesson.Cards = append(lesson.Cards, model.Card{
			ID:        lf.ID + ":" + strconv.Itoa(i+1),
			Source:    c.Source,
			JP:        c.JP,
			Romaji:    c.Romaji,
			Notes:     c.Notes,
			JLPT:      level,
			Functions: lf.Functions,
			Freq:      c.Freq,
		})
	}
	return lesson, nil
}

// kanaFile mirrors the on-disk YAML shape of a kana table.
type kanaFile struct {
	Type  string `yaml:"type"`
	Items []struct {
		Char     string `yaml:"char"`
		Romaji   string `yaml:"romaji"`
		Category string `yaml:"category"`
	} `yaml:"items"`
}

func loadKana(fsys fs.FS, pair string) ([]model.KanaItem, error) {
	dir := path.Join("content", pair, "kana")
	files, err := fs.Glob(fsys, path.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("glob kana: %w", err)
	}
	sort.Strings(files)

	var items []model.KanaItem
	for _, file := range files {
		parsed, err := parseKana(fsys, file)
		if err != nil {
			return nil, err
		}
		items = append(items, parsed...)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no kana found in content/%s/kana", pair)
	}
	return items, nil
}

func parseKana(fsys fs.FS, file string) ([]model.KanaItem, error) {
	data, err := fs.ReadFile(fsys, file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", file, err)
	}
	var kf kanaFile
	if err := yaml.Unmarshal(data, &kf); err != nil {
		return nil, fmt.Errorf("parse %s: %w", file, err)
	}
	kt := model.KanaType(kf.Type)
	if !kt.Valid() {
		return nil, fmt.Errorf("%s: invalid kana type %q", file, kf.Type)
	}
	if len(kf.Items) == 0 {
		return nil, fmt.Errorf("%s: no kana items", file)
	}

	items := make([]model.KanaItem, 0, len(kf.Items))
	for i, it := range kf.Items {
		if it.Char == "" || it.Romaji == "" {
			return nil, fmt.Errorf("%s: item %d is missing char or romaji", file, i+1)
		}
		category := model.KanaCategory(it.Category)
		if it.Category == "" {
			category = model.Base
		}
		if !category.Valid() {
			return nil, fmt.Errorf("%s: item %d has invalid category %q", file, i+1, it.Category)
		}
		items = append(items, model.KanaItem{Char: it.Char, Romaji: it.Romaji, Type: kt, Category: category})
	}
	return items, nil
}
