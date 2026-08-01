```
██████╗  ██████╗ ██╗  ██╗   ██╗ ██████╗ ██╗      ██████╗ ████████╗
██╔══██╗██╔═══██╗██║  ╚██╗ ██╔╝██╔════╝ ██║     ██╔═══██╗╚══██╔══╝
██████╔╝██║   ██║██║   ╚████╔╝ ██║  ███╗██║     ██║   ██║   ██║   
██╔═══╝ ██║   ██║██║    ╚██╔╝  ██║   ██║██║     ██║   ██║   ██║   
██║     ╚██████╔╝███████╗██║   ╚██████╔╝███████╗╚██████╔╝   ██║   
╚═╝      ╚═════╝ ╚══════╝╚═╝    ╚═════╝ ╚══════╝ ╚═════╝    ╚═╝   
```

Polyglot is an open-source, cross-platform, offline terminal app for learning languages.

## Supported languages

| | Language pairs |
| --- | --- |
| **Available now** | Spanish → Japanese (v1) |

The interface is in **Spanish** for v1 — the learner's source language. The codebase,
comments, and identifiers are in English.

## Installation

**Prebuilt binaries** for macOS, Windows, and Linux (amd64 and arm64) are attached to
each [GitHub Release](https://github.com/sebastiancaraballo/polyglot/releases). Download
the archive for your platform, extract it, and run the `polyglot` binary.

**With Cargo:**

```sh
cargo install --git https://github.com/sebastiancaraballo/polyglot polyglot-term
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

Requires Rust 1.90+.

```sh
cargo run -p polyglot-term                                 # run the app
cargo test --workspace                                     # run tests
cargo clippy --workspace --all-targets -- -D warnings      # lint
cargo fmt --all -- --check                                 # formatting check
```

The workspace is split in two crates: `core/` holds the language-agnostic engine
(content model, spaced repetition, study logic, storage) and `term/` the ratatui
terminal client. Keeping the engine UI-free is what lets a future mobile or
desktop client reuse it.

## License

[MIT](LICENSE) © 2026 Sebastián Caraballo

The MIT license covers the content and data too, not just the code. Third-party
assets bundled in or derived into the repository, with their licenses and
required attributions, are listed in [`NOTICE`](NOTICE) (generated from
`internal/license/assets.yaml`).
