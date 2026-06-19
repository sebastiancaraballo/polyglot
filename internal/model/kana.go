package model

// KanaType distinguishes the two Japanese syllabaries.
type KanaType string

const (
	Hiragana KanaType = "hiragana"
	Katakana KanaType = "katakana"
)

// Valid reports whether k is a recognized kana type.
func (k KanaType) Valid() bool {
	return k == Hiragana || k == Katakana
}

// KanaItem is a single kana character paired with its romaji reading.
type KanaItem struct {
	Char   string
	Romaji string
	Type   KanaType
}
