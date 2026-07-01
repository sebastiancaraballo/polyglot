// Package content loads and validates the course content embedded into the binary:
// the language-agnostic communicative-function catalog (content/functions/*.yaml),
// per-language YAML lessons, kana tables, grammar patterns, and story chapters
// (content/<pair>/{lessons,kana,grammar,story}/*.yaml).
//
// Validation enforces that every function a lesson references resolves to the
// catalog, that JLPT and CEFR levels are recognized, that frequency ranks are
// non-negative, that every kana a card depends on is teachable (present in the
// kana tables), that a grammar pattern's slots only reference vocabulary the
// learner is actually taught, and that a story chapter's practice beats only
// reference lessons or kana types that exist. Kanji dependencies are tracked
// separately and currently deferred. Grammar and story content are optional per
// language pair.
package content
