package content

import (
	"fmt"
	"io/fs"
	"path"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// functionsFile mirrors the on-disk YAML shape of a functions catalog file.
type functionsFile struct {
	Functions []struct {
		ID          string `yaml:"id"`
		CEFR        string `yaml:"cefr"`
		Description string `yaml:"description"`
	} `yaml:"functions"`
}

// loadFunctions reads and validates the language-agnostic communicative-function
// catalog from content/functions/*.yaml. The catalog is shared across all language
// pairs (the universal "spine"). A missing or empty catalog is not an error here;
// lessons that reference an unknown function are rejected during lesson parsing.
func loadFunctions(fsys fs.FS) (model.FunctionCatalog, error) {
	files, err := fs.Glob(fsys, path.Join("content", "functions", "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("glob functions: %w", err)
	}
	sort.Strings(files)

	catalog := make(model.FunctionCatalog)
	for _, file := range files {
		data, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file, err)
		}
		var ff functionsFile
		if err := yaml.Unmarshal(data, &ff); err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}
		for i, fn := range ff.Functions {
			if fn.ID == "" {
				return nil, fmt.Errorf("%s: function %d is missing id", file, i+1)
			}
			if _, ok := catalog[fn.ID]; ok {
				return nil, fmt.Errorf("%s: duplicate function id %q", file, fn.ID)
			}
			level := model.CEFR(fn.CEFR)
			if !level.Valid() {
				return nil, fmt.Errorf("%s: function %q has invalid cefr level %q", file, fn.ID, fn.CEFR)
			}
			if fn.Description == "" {
				return nil, fmt.Errorf("%s: function %q is missing description", file, fn.ID)
			}
			catalog[fn.ID] = model.Function{ID: fn.ID, CEFR: level, Description: fn.Description}
		}
	}
	return catalog, nil
}
