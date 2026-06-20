# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project scaffolding: module layout, license, documentation, and core dependencies.
- Continuous integration (GitHub Actions): tests, `go vet`, `gofmt`, and `golangci-lint` across Linux, macOS, and Windows.
- Domain models for local profiles, card scheduling state, and aggregate stats.
- SQLite-backed storage layer (`modernc.org/sqlite`, no CGO) with goose-managed, embedded schema migrations and WAL mode. Supports multiple local profiles, with progress and stats keyed per profile.
- Content loader (`internal/content`): parses and validates YAML lessons and kana tables, embedded into the binary via `go:embed`. Includes the v1 Spanish → Japanese course with starter N5 lessons (greetings, numbers) and full hiragana/katakana tables.
- Domain models for course content: `Card`, `Lesson`, `KanaItem`, `JLPT` levels, and `KanaType`.
- Spaced-repetition scheduler (`internal/srs`): a pure `Review` function with Again/Hard/Good/Easy grades (SM-2 style ease and interval growth), plus `NewCard`, `IsDue`, and `PreviewInterval` helpers.
- Interactive terminal UI foundation (Bubble Tea v2): a root router model, the main menu screen with a JLPT progress badge and study streak, a Spanish localization package (`internal/i18n`), and a theme/layout package (`internal/ui`) with a high-contrast variant, `NO_COLOR` support, responsive centering, and a progress bar.
- `Storage.CountLearnedCards` to report how many cards a profile has learned (for the progress badge).
- Study screens: kana trainer, spaced-repetition flashcards (reveal + 1–4 grading with next-interval previews), multiple-choice vocabulary quiz, and a statistics screen (JLPT progress, streak/record, kana totals).
- Screen routing: a `nav` package for navigation messages and a router that builds and switches screens; the menu now navigates to each study mode.
- Shared study logic (`internal/study`): multiple-choice option generation and study-streak bookkeeping, both unit-tested.
- Flashcards and quiz persist reviews through the spaced-repetition scheduler and update the daily streak.
- First-run onboarding (`internal/screens/onboarding`): teaches the keyboard controls and runs a guided sample exercise, then marks the profile as onboarded so it does not repeat. New profiles start in onboarding automatically.

### Changed
- Kana trainer: the prompted character is now shown in a large, bordered focal tile centered above the answer options for better readability.

### Fixed
- Japanese long-vowel romaji now uses pronunciation forms with macrons in lesson cards, with kana input forms documented in notes.
- Spacebar shortcuts now work with Bubble Tea v2 key names across the menu, onboarding, kana trainer, quiz, and flashcards screens.

[Unreleased]: https://github.com/sebastiancaraballo/polyglot/commits/main
