// Package content loads and validates the course content embedded into the binary:
// the language-agnostic communicative-function catalog (content/functions/*.yaml),
// per-language YAML lessons, and kana tables
// (content/<pair>/{lessons,kana}/*.yaml).
//
// Validation enforces that every function a lesson references resolves to the
// catalog, that JLPT and CEFR levels are recognized, that frequency ranks are
// non-negative, and that every kana a card depends on is teachable (present in the
// kana tables). Kanji dependencies are tracked separately and currently deferred.
package content
