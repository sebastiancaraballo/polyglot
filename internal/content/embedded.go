package content

import polyglot "github.com/sebastiancaraballo/polyglot"

// DefaultPair is the language pair shipped in v1.
const DefaultPair = "es-ja"

// LoadEmbedded loads a course from the content bundled into the binary.
func LoadEmbedded(pair string) (*Course, error) {
	return Load(polyglot.ContentFS, pair)
}
