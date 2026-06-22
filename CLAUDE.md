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
  tests. For TUI screens, use golden-file tests via `github.com/charmbracelet/x/exp/golden`
  (`teatest` is Bubble Tea v1 only and is incompatible with the v2 models): drive the model
  through `Update`/`View().Content` and snapshot it. Render with `ui.PlainTheme()` for stable,
  escape-free output, and regenerate goldens with `go test ./... -update`.
- Wrap errors with `%w`.

## Git & GitHub workflow

- **Commits:** concise imperative title, **no body/description**. One logical change per commit.
- **Pull requests:** the PR **description must include a changelog** (e.g. `Added` /
  `Changed` / `Fixed` sections following Keep a Changelog).
- **Versioning:** Semantic Versioning. Keep `CHANGELOG.md` updated
  ([Keep a Changelog](https://keepachangelog.com/) format).
- **Project board:** when a PR is merged, move its corresponding item in the GitHub
  Project (Projects v3, user project #3) from its current status (e.g. `Todo`) to
  `Done`. Use the `gh` GraphQL API (`updateProjectV2ItemFieldValue` on the Status
  field) to set it.

### Worktrees

- Start each new feature in its own git worktree so the working copy and branch
  stay isolated: `git worktree add ../polyglot-<feature> -b <feature>`, and
  `git worktree remove ../polyglot-<feature>` once the PR is merged.
- No extra data isolation is configured, and none is needed: tests open their own
  throwaway databases via `t.TempDir()`, and the app's real database
  (`os.UserConfigDir()/polyglot`) is only touched by `go run ./cmd/polyglot`, which
  is not part of this workflow. If running the app locally from a worktree ever
  becomes necessary, add a data-directory override first so it doesn't share the
  real database.
- **At the end of an implementation done in a worktree, always finish the reply
  with the copy-paste command to run the app from that worktree**, so it's easy to
  try locally:

  ```sh
  cd ../polyglot-<feature> && go run ./cmd/polyglot
  ```

  (Note: this uses the real database at `os.UserConfigDir()/polyglot` per the
  caveat above.)

## Common commands

```sh
go run ./cmd/polyglot     # run the app
go build ./...            # build
go test ./...             # tests
go vet ./...              # static checks
gofmt -l .               # formatting check
```
