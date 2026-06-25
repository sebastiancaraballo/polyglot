package content

import (
	polyglot "github.com/sebastiancaraballo/polyglot"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// DefaultPair is the language pair shipped in v1.
const DefaultPair = "es-ja"

// LoadEmbedded loads a course from the content bundled into the binary.
func LoadEmbedded(pair string) (*Course, error) {
	return Load(polyglot.ContentFS, pair)
}

// LoadEmbeddedFrequency loads the word-frequency list for a target language from
// the content bundled into the binary (e.g. lang "ja").
func LoadEmbeddedFrequency(lang string) ([]model.FreqEntry, error) {
	return loadFrequency(polyglot.ContentFS, lang)
}
