# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Foundations decoding gate (Hiragana & Katakana to automaticity): the kana trainer now times each answer and tracks per-kana progress toward *automaticity* — a run of correct, fast answers (`internal/study.GradeKana`), persisted per profile in a new `kana_progress` table. Grounded in the Simple View of Reading (decoding must precede comprehension), the curriculum is gated: katakana practice is locked until the hiragana base gojūon is fluent, and reading activities (Flashcards, Quiz) are locked on the main menu until both syllabaries are fluent (`internal/study.Gate`). The kana picker shows per-group mastery counts and a fluency badge; locked items show a lock marker and an explanatory hint.
- Resource licensing & attribution system: a machine-readable manifest of every third-party asset (`internal/license/assets.yaml`) with its source, license, and required attribution. The repo-root `NOTICE` is generated from it (`go run ./tools/gennotice`), and a test fails CI if any asset lacks complete provenance, carries a non-permissive license (rejecting NonCommercial, ShareAlike/copyleft), or if `NOTICE` has drifted from the manifest. Seeds the public-domain Natural Earth coastlines already used by the globe.
- Japanese word-frequency list (`content/ja/frequency.tsv`, top 10,000 words) derived from the Tatoeba Project corpus (CC BY 2.0 FR, attribution recorded in `NOTICE`). Derivation runs offline via `tools/genfreq` (a separate module so the kagome tokenizer and IPADIC dictionary stay out of the shipped binary). The list is parsed and validated but not yet wired into card scheduling; the rationale and rejected alternatives are recorded in `docs/adr/0001-japanese-frequency-list.md`.
- Cross-curriculum spaced-repetition review queue (`internal/review`): a shared, UI-free scheduler that turns the whole curriculum — vocabulary and kana (and later grammar) — into a single study session. It loads each item's scheduling state, keeps only the items currently due, and orders them most-overdue first within each strand while interleaving strands round-robin (interleaving aids retention), capped per session. A new "Repaso" menu entry runs this mixed-strand session; kana now participates in spaced repetition (keyed as `kana:<char>`), and the flashcard-style screen renders any strand, so the existing vocabulary-only "Flashcards" entry now builds its queue through the same engine instead of its own inline logic.
- Curriculum content model foundation: a language-agnostic catalog of communicative functions (`content/functions/*.yaml`), each graded with a CEFR level (new `model.CEFR`), that per-language lessons reference by ID — separating the universal "spine" from the per-language "skin". Cards gain optional communicative-function tags (inherited from their lesson) and an optional frequency rank. The loader now resolves function references, validates CEFR levels and frequency ranks, and verifies that every kana a card depends on is teachable (present in the kana tables, with longest-match tokenization so yōon combos like きゅう are handled). Kanji dependencies and frequency backfill are deferred to follow-up work.
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
- The learned-progress figure now counts the whole curriculum: its label is "tarjetas aprendidas" (cards learned) and its total includes kana as well as vocabulary, keeping it coherent now that kana is scheduled by spaced repetition.
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
