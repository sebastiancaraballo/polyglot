package content

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"

	"gopkg.in/yaml.v3"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// Course is the fully loaded and validated content for a single language pair.
type Course struct {
	Pair    string
	Lessons []model.Lesson
	Kana    []model.KanaItem
}

// Load reads, parses, and validates the course for pair from fsys. Paths are
// resolved as content/<pair>/{lessons,kana}/*.yaml so that the same loader works
// for the embedded FS and for test filesystems.
func Load(fsys fs.FS, pair string) (*Course, error) {
	lessons, err := loadLessons(fsys, pair)
	if err != nil {
		return nil, err
	}
	kana, err := loadKana(fsys, pair)
	if err != nil {
		return nil, err
	}
	return &Course{Pair: pair, Lessons: lessons, Kana: kana}, nil
}

// lessonFile mirrors the on-disk YAML shape of a lesson. The source-language key
// is "es" for the v1 Spanish → Japanese pair.
type lessonFile struct {
	ID    string `yaml:"id"`
	Title string `yaml:"title"`
	JLPT  string `yaml:"jlpt"`
	Cards []struct {
		Source string `yaml:"es"`
		JP     string `yaml:"jp"`
		Romaji string `yaml:"romaji"`
		Notes  string `yaml:"notes"`
	} `yaml:"cards"`
}

func loadLessons(fsys fs.FS, pair string) ([]model.Lesson, error) {
	dir := path.Join("content", pair, "lessons")
	files, err := fs.Glob(fsys, path.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("glob lessons: %w", err)
	}
	sort.Strings(files)

	seen := make(map[string]bool)
	lessons := make([]model.Lesson, 0, len(files))
	for _, file := range files {
		lesson, err := parseLesson(fsys, file)
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

func parseLesson(fsys fs.FS, file string) (model.Lesson, error) {
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
	if len(lf.Cards) == 0 {
		return model.Lesson{}, fmt.Errorf("%s: lesson has no cards", file)
	}

	lesson := model.Lesson{ID: lf.ID, Title: lf.Title, JLPT: level}
	for i, c := range lf.Cards {
		if c.Source == "" || c.JP == "" || c.Romaji == "" {
			return model.Lesson{}, fmt.Errorf("%s: card %d is missing es, jp, or romaji", file, i+1)
		}
		lesson.Cards = append(lesson.Cards, model.Card{
			ID:     lf.ID + ":" + strconv.Itoa(i+1),
			Source: c.Source,
			JP:     c.JP,
			Romaji: c.Romaji,
			Notes:  c.Notes,
			JLPT:   level,
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
