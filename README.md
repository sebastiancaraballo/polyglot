# Polyglot

> An interactive terminal app to learn languages, one terminal session at a time.

Polyglot is a cross-platform (macOS, Windows, Linux), open-source TUI for learning
languages. The first version focuses on **Spanish → Japanese**, with spaced-repetition
flashcards, a kana trainer, and multiple-choice quizzes. The architecture is built to
add more language pairs later (English → Japanese, Spanish → English, …).

The application ships as a **single self-contained binary** — no runtime to install.

> **Note:** The user interface is in **Spanish** (the learner's source language for v1).
> The codebase, comments, and identifiers are in English following standard Go practices.

## Features (v1)

- ▣ **Spaced-repetition flashcards (SRS)** — review vocabulary on an optimal schedule.
- ◇ **Kana trainer** — learn hiragana and katakana.
- ✓ **Multiple-choice quizzes** — quick reinforcement.
- ▤ **Progress, XP & streaks** — earn experience points for every answer, keep a daily streak, and track words learned.
- ◎ **Local profiles** — named learners with a profile switcher and per-profile progress.
- ⚙ **Settings** — delete the active profile or all app data (with confirmation) to start fresh.
- ※ **Accessible** — responsive layout, high-contrast theme, `NO_COLOR` support, keyboard-first.

Typing/romaji input is planned for v1.1.

## Installation

**Prebuilt binaries** for macOS, Windows, and Linux (amd64 and arm64) are attached to
each [GitHub Release](https://github.com/sebastiancaraballo/polyglot/releases). Download
the archive for your platform, extract it, and run the `polyglot` binary.

**With Go:**

```sh
go install github.com/sebastiancaraballo/polyglot/cmd/polyglot@latest
```

> Homebrew and Scoop packages are planned for a future release.

## Usage

```sh
polyglot
```

On first run you'll create a named profile and go through a short onboarding that teaches
the keyboard controls with a guided exercise. Use the profile header in the main menu to
switch learners or create another profile.

## Development

Requires Go 1.26+.

```sh
go run ./cmd/polyglot   # run the app
go test ./...           # run tests
go vet ./...            # static checks
gofmt -l .              # formatting check
```

## License

[MIT](LICENSE) © 2026 Sebastián Caraballo
