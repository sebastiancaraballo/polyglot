```
██████╗  ██████╗ ██╗  ██╗   ██╗ ██████╗ ██╗      ██████╗ ████████╗
██╔══██╗██╔═══██╗██║  ╚██╗ ██╔╝██╔════╝ ██║     ██╔═══██╗╚══██╔══╝
██████╔╝██║   ██║██║   ╚████╔╝ ██║  ███╗██║     ██║   ██║   ██║   
██╔═══╝ ██║   ██║██║    ╚██╔╝  ██║   ██║██║     ██║   ██║   ██║   
██║     ╚██████╔╝███████╗██║   ╚██████╔╝███████╗╚██████╔╝   ██║   
╚═╝      ╚═════╝ ╚══════╝╚═╝    ╚═════╝ ╚══════╝ ╚═════╝    ╚═╝   
```

Polyglot is an open-source, cross-platform terminal app for learning languages —
built so that learning a language feels like stepping into the world that speaks it.

## Vision

Every language expresses the same human core — to greet, to ask, to thank, to hope, to
say goodbye. Polyglot teaches that **shared human story**, then lets each language's
culture give it its own voice and color. Adding a language is telling the same story in a
new world, not starting over from scratch.

- **Learning a language is living that story.** Flashcards, quizzes, and the kana trainer
  are mechanics inside one narrative; progress is earned through **mastery**, not by
  clicking "next".
- **Your native language is the lens.** There is no universal "beginner" — the distance
  between any two languages is different, so Polyglot treats the learner's native language
  as a first-class input that shapes pacing and scaffolding.
- **Grounded in evidence, not intuition.** What to teach, and in what order, follows
  established standards, word-frequency data, and the science of how people learn.
- **Built for everyone, including the overlooked.** The same human core is what lets
  Polyglot reach **low-resource and endangered languages**, each wrapped in its own
  culture and dignity rather than treated as a lesser cousin of a bigger language.
- **Free and open, all the way down.** MIT-licensed — and that promise extends to
  *everything* in the project, not just the code but the lessons, word lists, and assets.

Read the full [Manifesto](MANIFESTO.md).

## Supported languages

| | Language pairs |
| --- | --- |
| **Available now** | Spanish → Japanese (v1) |
| **On the roadmap** | More pairs (English → Japanese, Spanish → English, …), reaching toward low-resource languages |

The interface is in **Spanish** for v1 — the learner's source language. The codebase,
comments, and identifiers are in English, following standard Go practices.

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

The MIT license covers the content and data too, not just the code. Third-party
assets bundled in or derived into the repository, with their licenses and
required attributions, are listed in [`NOTICE`](NOTICE) (generated from
`internal/license/assets.yaml`).
