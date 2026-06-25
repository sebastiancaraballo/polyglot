package study

import (
	"unicode"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// Decoder reports which Japanese text a learner can already *read*, given the
// kana they have driven to automaticity. It implements the progressive side of
// the Foundations gate: instead of locking all reading until every kana is
// fluent, the learner meets "decodable" words and sentences — those built only
// from kana they have already mastered — and the readable set grows as they do.
// This mirrors the decodable-texts approach to early reading.
type Decoder struct {
	combos   map[string]bool // valid two-rune kana strings (yōon, e.g. "きゃ")
	mastered map[string]bool // kana the learner has driven to automaticity
}

// NewDecoder builds a decoder from the course's kana set (so it can recognize
// multi-rune combos) and a learner's progress.
func NewDecoder(items []model.KanaItem, progress map[string]model.KanaProgress) Decoder {
	d := Decoder{combos: map[string]bool{}, mastered: map[string]bool{}}
	for _, it := range items {
		if len([]rune(it.Char)) == 2 {
			d.combos[it.Char] = true
		}
	}
	for char, p := range progress {
		if p.Mastered {
			d.mastered[char] = true
		}
	}
	return d
}

// Decodable reports whether jp can be read with only the learner's mastered
// kana. It tokenizes with longest match (a yōon combo before its parts), the
// same way the content loader validates kana coverage. A string with no kana, or
// containing kanji (whose decoding is a separate, later gate), is not decodable.
func (d Decoder) Decodable(jp string) bool {
	runes := []rune(jp)
	sawKana := false
	for i := 0; i < len(runes); {
		r := runes[i]
		switch {
		case isKana(r):
			if i+1 < len(runes) {
				if pair := string(runes[i : i+2]); d.combos[pair] {
					if !d.mastered[pair] {
						return false
					}
					sawKana = true
					i += 2
					continue
				}
			}
			if !d.mastered[string(r)] {
				return false
			}
			sawKana = true
			i++
		case unicode.In(r, unicode.Han):
			return false // kanji is not yet decodable
		default:
			i++ // skip punctuation, spaces, ASCII
		}
	}
	return sawKana
}

// isKana reports whether r belongs to either Japanese syllabary.
func isKana(r rune) bool {
	return unicode.In(r, unicode.Hiragana, unicode.Katakana)
}
