package content

import (
	"fmt"
	"io/fs"
	"path"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// storyFile mirrors the on-disk YAML shape of a story chapter.
type storyFile struct {
	ID    string     `yaml:"id"`
	Title string     `yaml:"title"`
	Beats []beatFile `yaml:"beats"`
}

// beatFile mirrors the on-disk YAML shape of a single beat.
type beatFile struct {
	Kind     string `yaml:"kind"`
	Speaker  string `yaml:"speaker"`
	Place    string `yaml:"place"`
	Source   string `yaml:"es"`
	JP       string `yaml:"jp"`
	Romaji   string `yaml:"romaji"`
	Practice string `yaml:"practice"`
	RefID    string `yaml:"ref_id"`
}

// loadChapters reads every story chapter for pair. Like grammar patterns,
// story content is optional: a pair with no content/<pair>/story directory
// yet is not an error, since Core must not force every language pair to ship
// story content on day one.
func loadChapters(fsys fs.FS, pair string) ([]model.Chapter, error) {
	dir := path.Join("content", pair, "story")
	files, err := fs.Glob(fsys, path.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("glob story: %w", err)
	}
	sort.Strings(files)

	seen := make(map[string]bool)
	var chapters []model.Chapter
	for _, file := range files {
		c, err := parseChapter(fsys, file)
		if err != nil {
			return nil, err
		}
		if seen[c.ID] {
			return nil, fmt.Errorf("%s: duplicate chapter id %q", file, c.ID)
		}
		seen[c.ID] = true
		chapters = append(chapters, c)
	}
	return chapters, nil
}

func parseChapter(fsys fs.FS, file string) (model.Chapter, error) {
	data, err := fs.ReadFile(fsys, file)
	if err != nil {
		return model.Chapter{}, fmt.Errorf("read %s: %w", file, err)
	}
	var sf storyFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return model.Chapter{}, fmt.Errorf("parse %s: %w", file, err)
	}
	if sf.ID == "" {
		return model.Chapter{}, fmt.Errorf("%s: missing chapter id", file)
	}
	if sf.Title == "" {
		return model.Chapter{}, fmt.Errorf("%s: missing chapter title", file)
	}
	if len(sf.Beats) == 0 {
		return model.Chapter{}, fmt.Errorf("%s: chapter has no beats", file)
	}

	beats := make([]model.Beat, 0, len(sf.Beats))
	for i, b := range sf.Beats {
		beat, err := parseBeat(b)
		if err != nil {
			return model.Chapter{}, fmt.Errorf("%s: beat %d: %w", file, i+1, err)
		}
		beats = append(beats, beat)
	}

	return model.Chapter{ID: sf.ID, Title: sf.Title, Beats: beats}, nil
}

func parseBeat(b beatFile) (model.Beat, error) {
	kind := model.BeatKind(b.Kind)
	if !kind.Valid() {
		return model.Beat{}, fmt.Errorf("invalid beat kind %q", b.Kind)
	}

	beat := model.Beat{Kind: kind, Speaker: b.Speaker, Place: b.Place}

	switch kind {
	case model.Narration, model.Dialogue:
		if b.Source == "" || b.JP == "" {
			return model.Beat{}, fmt.Errorf("%s beat is missing es or jp", kind)
		}
		if kind == model.Dialogue && b.Speaker == "" {
			return model.Beat{}, fmt.Errorf("dialogue beat is missing speaker")
		}
		beat.Source, beat.JP, beat.Romaji = b.Source, b.JP, b.Romaji
	case model.Present:
		practice := model.PracticeKind(b.Practice)
		if !practice.Valid() {
			return model.Beat{}, fmt.Errorf("invalid practice kind %q", b.Practice)
		}
		if b.RefID == "" {
			return model.Beat{}, fmt.Errorf("present beat is missing ref_id")
		}
		// The framing line (speaker/es/jp/romaji) is optional: a present beat
		// may set the scene diegetically, but its core content is the pool it
		// renders. es/jp travel together when given.
		if (b.Source == "") != (b.JP == "") {
			return model.Beat{}, fmt.Errorf("present beat framing needs both es and jp, or neither")
		}
		beat.Practice, beat.RefID = practice, b.RefID
		beat.Source, beat.JP, beat.Romaji = b.Source, b.JP, b.Romaji
	case model.Practice:
		practice := model.PracticeKind(b.Practice)
		if !practice.Valid() {
			return model.Beat{}, fmt.Errorf("invalid practice kind %q", b.Practice)
		}
		if b.RefID == "" {
			return model.Beat{}, fmt.Errorf("practice beat is missing ref_id")
		}
		if b.Source != "" || b.JP != "" || b.Romaji != "" || b.Speaker != "" {
			return model.Beat{}, fmt.Errorf("practice beat must not carry dialogue fields")
		}
		beat.Practice, beat.RefID = practice, b.RefID
	}
	return beat, nil
}
