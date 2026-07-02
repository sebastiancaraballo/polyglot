package content

import (
	"fmt"
	"unicode"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// kanaSet builds the set of teachable kana strings from the loaded kana tables.
// Combination kana (yōon, e.g. "きゃ") are stored as two-rune strings, so the set
// contains both single-rune and two-rune entries.
func kanaSet(items []model.KanaItem) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, it := range items {
		set[it.Char] = true
	}
	return set
}

// isKana reports whether r belongs to either Japanese syllabary.
func isKana(r rune) bool {
	return unicode.In(r, unicode.Hiragana, unicode.Katakana)
}

// isKanaMark reports whether r is a kana modifier with no standalone reading:
// the sokuon (small tsu っ/ッ), which geminates the following consonant, or the
// chōonpu (ー), which lengthens the preceding vowel. They modify an adjacent,
// separately-teachable kana rather than being syllables themselves, so they are
// always decodable once a learner knows kana and are not taught as their own
// items. (The chōonpu is already script "Common" and thus skipped as non-kana,
// but is listed here so the intent is explicit and robust.)
func isKanaMark(r rune) bool {
	switch r {
	case 'っ', 'ッ', 'ー':
		return true
	default:
		return false
	}
}

// checkKanaCoverage verifies that every kana used in jp is present in set, so no
// card depends on a character the learner cannot yet be taught. It tokenizes with
// longest match — trying a two-rune combination (yōon) before a single rune — so
// combos like "きゅう" decompose as "きゅ" + "う" rather than failing on a bare
// small kana. Non-kana runes (kanji, ASCII, punctuation) are skipped; kanji
// dependencies are tracked separately and deferred.
func checkKanaCoverage(jp string, set map[string]bool) error {
	runes := []rune(jp)
	for i := 0; i < len(runes); {
		r := runes[i]
		if isKanaMark(r) || !isKana(r) {
			i++
			continue
		}
		if i+1 < len(runes) {
			if pair := string(runes[i : i+2]); set[pair] {
				i += 2
				continue
			}
		}
		if set[string(r)] {
			i++
			continue
		}
		return fmt.Errorf("uses kana %q not present in the kana tables", string(r))
	}
	return nil
}
