# CLAUDE.md

Guidance for working in this repository. Read before making changes.

## Project

Polyglot is a cross-platform (macOS, Windows, Linux) interactive terminal app for
learning languages. v1 focuses on **Spanish → Japanese**. It ships as a single
self-contained binary. The architecture is built to add more language pairs later.

## Language conventions

- **Code, comments, identifiers, commit messages, PRs, and docs: English.**
- **User-facing UI strings: Spanish** (v1). Never hardcode UI strings in logic — put
  them in `internal/i18n` so more UI languages can be added later.

## Tech stack & import paths

- **Go 1.26+**, standard layout: `cmd/` (entry points) + `internal/` (packages).
- **TUI:** Charm's Bubble Tea / Lipgloss / Bubbles **v2**. ⚠️ The v2 modules use the
  `charm.land/...` import paths (e.g. `charm.land/bubbletea/v2`), **not**
  `github.com/charmbracelet/...`.
- **Persistence:** SQLite via `modernc.org/sqlite` (pure Go). Schema migrations via
  `github.com/pressly/goose/v3`, embedded with `go:embed`. The database lives under
  `os.UserConfigDir()/polyglot`. Local profiles: progress is keyed by `profile_id`.
- **Content:** YAML lessons + Markdown guides under `content/<pair>/` (v1: `es-ja/`),
  embedded with `go:embed`. Cards are tagged with their JLPT level.

## Hard constraints

- **License is MIT.** Only add **permissive** dependencies (MIT/BSD/Apache). No copyleft.
- **No CGO.** CGO breaks single-binary cross-compilation. Keep all deps pure Go
  (this is why we use `modernc.org/sqlite`, not the cgo SQLite driver).
- **Never commit a user database** (`*.db` is gitignored).

## Accessibility

Responsive layout (react to `WindowSizeMsg`, no fixed widths), high-contrast theme,
honor `NO_COLOR`, keyboard-first navigation, never rely on color alone (pair with
symbols/text), keep romaji visible alongside Japanese.

## Quality

- Format with `gofmt`; keep `go vet ./...` and `golangci-lint` clean.
- Tests: `go test ./...`. Prefer table-driven unit tests. Validate all YAML content in
  tests. Use `github.com/charmbracelet/x/exp/teatest` golden-file tests for TUI screens.
- Wrap errors with `%w`.

## Git & GitHub workflow

- **Commits:** concise imperative title, **no body/description**. One logical change per commit.
- **Pull requests:** the PR **description must include a changelog** (e.g. `Added` /
  `Changed` / `Fixed` sections following Keep a Changelog).
- **Versioning:** Semantic Versioning. Keep `CHANGELOG.md` updated
  ([Keep a Changelog](https://keepachangelog.com/) format).

## Common commands

```sh
go run ./cmd/polyglot     # run the app
go build ./...            # build
go test ./...             # tests
go vet ./...              # static checks
gofmt -l .               # formatting check
```
