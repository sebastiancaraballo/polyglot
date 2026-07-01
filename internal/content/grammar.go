package content

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// placeholderRe matches a "{name}" slot placeholder in a pattern frame.
var placeholderRe = regexp.MustCompile(`\{([A-Za-z][A-Za-z0-9_]*)\}`)

// patternFile mirrors the on-disk YAML shape of a grammar pattern.
type patternFile struct {
	ID    string `yaml:"id"`
	Title string `yaml:"title"`
	JLPT  string `yaml:"jlpt"`
	Frame string `yaml:"frame"`
	Notes string `yaml:"notes"`
	Slots []struct {
		Name    string   `yaml:"name"`
		Cards   []string `yaml:"cards"`
		Default string   `yaml:"default"`
	} `yaml:"slots"`
}

// loadPatterns reads every grammar pattern for pair. Unlike lessons and kana,
// grammar content is optional: a pair with no content/<pair>/grammar
// directory yet is not an error, since Core must not force every language
// pair to ship grammar content on day one.
func loadPatterns(fsys fs.FS, pair string) ([]model.Pattern, error) {
	dir := path.Join("content", pair, "grammar")
	files, err := fs.Glob(fsys, path.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("glob grammar: %w", err)
	}
	sort.Strings(files)

	seen := make(map[string]bool)
	var patterns []model.Pattern
	for _, file := range files {
		p, err := parsePattern(fsys, file)
		if err != nil {
			return nil, err
		}
		if seen[p.ID] {
			return nil, fmt.Errorf("%s: duplicate pattern id %q", file, p.ID)
		}
		seen[p.ID] = true
		patterns = append(patterns, p)
	}
	return patterns, nil
}

func parsePattern(fsys fs.FS, file string) (model.Pattern, error) {
	data, err := fs.ReadFile(fsys, file)
	if err != nil {
		return model.Pattern{}, fmt.Errorf("read %s: %w", file, err)
	}
	var pf patternFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return model.Pattern{}, fmt.Errorf("parse %s: %w", file, err)
	}
	if pf.ID == "" {
		return model.Pattern{}, fmt.Errorf("%s: missing pattern id", file)
	}
	if pf.Title == "" {
		return model.Pattern{}, fmt.Errorf("%s: missing pattern title", file)
	}
	if pf.Frame == "" {
		return model.Pattern{}, fmt.Errorf("%s: missing pattern frame", file)
	}
	level := model.JLPT(pf.JLPT)
	if !level.Valid() {
		return model.Pattern{}, fmt.Errorf("%s: invalid jlpt level %q", file, pf.JLPT)
	}
	if len(pf.Slots) == 0 {
		return model.Pattern{}, fmt.Errorf("%s: pattern has no slots", file)
	}

	slotNames := make(map[string]bool, len(pf.Slots))
	slots := make([]model.Slot, 0, len(pf.Slots))
	for i, s := range pf.Slots {
		if s.Name == "" {
			return model.Pattern{}, fmt.Errorf("%s: slot %d is missing a name", file, i+1)
		}
		if slotNames[s.Name] {
			return model.Pattern{}, fmt.Errorf("%s: duplicate slot name %q", file, s.Name)
		}
		slotNames[s.Name] = true
		if len(s.Cards) == 0 {
			return model.Pattern{}, fmt.Errorf("%s: slot %q has no candidate cards", file, s.Name)
		}
		def := s.Default
		if def == "" {
			def = s.Cards[0]
		} else if !contains(s.Cards, def) {
			return model.Pattern{}, fmt.Errorf("%s: slot %q default %q is not among its candidate cards", file, s.Name, def)
		}
		slots = append(slots, model.Slot{Name: s.Name, CardIDs: s.Cards, Default: def})
	}

	if err := checkFramePlaceholders(pf.Frame, slotNames); err != nil {
		return model.Pattern{}, fmt.Errorf("%s: %w", file, err)
	}

	return model.Pattern{
		ID:    pf.ID,
		Title: pf.Title,
		JLPT:  level,
		Frame: pf.Frame,
		Slots: slots,
		Notes: pf.Notes,
	}, nil
}

// checkFramePlaceholders verifies the frame's "{name}" placeholders are
// exactly the declared slot names: no undeclared placeholder, and no declared
// slot left unused.
func checkFramePlaceholders(frame string, slotNames map[string]bool) error {
	found := make(map[string]bool)
	for _, m := range placeholderRe.FindAllStringSubmatch(frame, -1) {
		name := m[1]
		if !slotNames[name] {
			return fmt.Errorf("frame references undeclared slot %q", name)
		}
		found[name] = true
	}
	for name := range slotNames {
		if !found[name] {
			return fmt.Errorf("slot %q is declared but never used in the frame", name)
		}
	}
	return nil
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
