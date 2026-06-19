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

- 🎴 **Spaced-repetition flashcards (SRS)** — review vocabulary on an optimal schedule.
- かな **Kana trainer** — learn hiragana and katakana.
- ✓ **Multiple-choice quizzes** — quick reinforcement.
- 📊 **Progress & JLPT level** — a motivational badge showing your estimated JLPT level.
- 👤 **Local profiles** — multiple learners on the same machine, each with their own progress.
- ♿ **Accessible** — responsive layout, high-contrast theme, `NO_COLOR` support, keyboard-first.

Typing/romaji input is planned for v1.1.

## Installation

> Distribution artifacts (Homebrew, Scoop, prebuilt binaries) will be published with the
> first release. In the meantime:

```sh
go install github.com/sebastiancaraballo/polyglot/cmd/polyglot@latest
```

## Usage

```sh
polyglot
```

On first run you'll create a profile and go through a short onboarding that teaches the
keyboard controls with a guided exercise.

## Development

Requires Go 1.26+.

```sh
go run ./cmd/polyglot   # run the app
go test ./...           # run tests
go vet ./...            # static checks
gofmt -l .              # formatting check
```

## Documentation

Project documentation and a guide for authoring lessons live in the
[project Wiki](https://github.com/sebastiancaraballo/polyglot/wiki).

## Contributing

Contributions are welcome — especially new lessons. See the Wiki's lesson-authoring guide.

## License

[MIT](LICENSE) © 2026 Sebastián Caraballo
