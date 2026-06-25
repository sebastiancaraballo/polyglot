# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Project manifesto (`MANIFESTO.md`): the vision and principles behind Polyglot — a universal "story-driven" approach (universal spine + cultural skin), the native language as the learner's lens, evidence-based pedagogy, inclusion of low-resource languages, and a fully open (MIT) resource policy. Linked from the README; `CLAUDE.md` gains an "Operating model" section describing the parallel per-pair tracks coordinated through the GitHub Project board.
- Animated braille globe in the main menu header: a rotating Earth that rests facing Japan (the target language), spins a full turn, then rests again. Frames are generated offline from public-domain Natural Earth coastlines and embedded as braille (`internal/art`); the globe stays static on the resting frame when `NO_COLOR` is set.
- Experience points (XP): a single per-profile counter that grows with every interaction — quiz answers, flashcard grades, and kana trainer answers (correct answers earn more; flashcards scale by recall grade), plus a one-time bonus for completing onboarding. The total is shown on the menu badge and the stats screen.
- Named profile setup with Unicode name validation, active-profile persistence, and a profile switcher reachable from the main menu header.
- Settings actions for deleting only the active profile or deleting all app data, both behind explicit confirmations defaulting to Cancel. Wiping all data now returns to first-run profile setup.
- Settings: a per-profile "Show romaji" toggle (on by default) controlling whether romaji appears alongside Japanese — it adds romaji to the quiz answer options and governs the flashcard reveal.
- "Tabla de Kana": a browsable reference chart of every kana with its romaji, navigated with ← / → through six pages (hiragana and katakana, each split into base, dakuten/handakuten, and combinations).
- Full dakuten (が…), handakuten (ぱ…), and combination/yōon (きゃ…) kana for both hiragana and katakana, tagged with a category.
- Kana trainer group picker: choose what to practice (everything, or a syllabary split by base / dakuten·handakuten / combinations) before each session.

### Changed
- Main menu header redesigned around the globe: the rotating globe and the app/menu content now render as two vertically centered columns, with the keyboard help pinned to the bottom of the frame.
- Screens now render inside a fixed-size frame whose dimensions depend only on the terminal size, so the border no longer grows or shrinks with its content (e.g. when a quiz reveals an answer) or when moving between sections.

### Fixed
- Kana reference chart frame now hugs the table (header on top, table below, help at the bottom) instead of floating the table in the middle of a full-screen frame with large blank margins above and below it.
- Kana trainer character tile now stays centered when an answer is revealed.
- Flashcard grading options now render one per line so the labels do not wrap across the frame.

### Removed
- The JLPT progress indicator (menu badge and stats screen). The hardcoded N5 → N4 level didn't reflect real proficiency; XP and the words-learned count replace it as accurate progress signals.

## [0.0.1] - 2026-06-20

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
- Golden-file tests for the menu, onboarding, and stats screens (via `github.com/charmbracelet/x/exp/golden`), plus a `ui.PlainTheme` for deterministic, escape-free rendering.
- Release automation: a GoReleaser config and a tag-triggered GitHub Actions workflow that build cross-platform binaries (macOS/Windows/Linux, amd64/arm64), generate checksums, and publish a GitHub Release. README documents installing from releases or via `go install`.

### Changed
- Keyboard command labels in Spanish UI help text now use uppercase key names.
- Terminal UI labels now use text symbols instead of pictographic emoji, and the language pair tagline uses ISO language codes (`es → ja`) instead of country flags.
- Kana trainer: the prompted character is now shown in a large, bordered focal tile centered above the answer options for better readability.

### Fixed
- Japanese long-vowel romaji now uses pronunciation forms with macrons in lesson cards, with kana input forms documented in notes.
- Spacebar shortcuts now work with Bubble Tea v2 key names across the menu, onboarding, kana trainer, quiz, and flashcards screens.

[Unreleased]: https://github.com/sebastiancaraballo/polyglot/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/sebastiancaraballo/polyglot/releases/tag/v0.0.1
