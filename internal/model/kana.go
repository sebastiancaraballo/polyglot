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

// KanaCategory groups kana by how they are formed: the base gojūon, voiced
// (dakuten) and semi-voiced (handakuten) variants, and palatalized combinations
// (yōon).
type KanaCategory string

const (
	Base       KanaCategory = "base"
	Dakuten    KanaCategory = "dakuten"
	Handakuten KanaCategory = "handakuten"
	Combo      KanaCategory = "combo"
)

// Valid reports whether c is a recognized kana category.
func (c KanaCategory) Valid() bool {
	switch c {
	case Base, Dakuten, Handakuten, Combo:
		return true
	default:
		return false
	}
}

// KanaItem is a single kana character paired with its romaji reading.
type KanaItem struct {
	Char     string
	Romaji   string
	Type     KanaType
	Category KanaCategory
}
