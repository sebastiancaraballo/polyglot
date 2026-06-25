# CLAUDE.md

Guidance for working in this repository. Read before making changes.

## Workflow checklist

Follow this order for **every** change. The ★ steps are process steps that nothing
will compile-fail or auto-catch, so they are the ones most often missed — do not
skip them. Each item links to its authoritative detail below.

1. ★ **Before editing any file, create a worktree** for the change:
   `git worktree add ../polyglot-<feature> -b <feature>`. See [Worktrees](#worktrees).
2. Write code, comments, and commits in English; keep user-facing strings in
   `internal/i18n` (Spanish). See [Language conventions](#language-conventions).
3. Keep it pure Go — no CGO, permissive deps only. See [Hard constraints](#hard-constraints).
4. ★ **Update `CHANGELOG.md` and any affected docs (e.g. README) in the same change.**
5. ★ **Before finishing, run and pass** `gofmt -l .`, `go vet ./...`,
   `golangci-lint run ./...`, and `go test ./...` (regenerate goldens with
   `-update`). Note: `golangci-lint` (it runs staticcheck) catches issues `go vet`
   does not, and CI fails on them — run it locally too. See [Quality](#quality).
6. The PR **description must include a Keep a Changelog changelog**. See
   [Git & GitHub workflow](#git--github-workflow).
7. ★ **Never merge a PR unless explicitly asked to.**
8. After a PR is merged, move its board item to `Done`. See [Git & GitHub workflow](#git--github-workflow).
9. ★ **When the change was made in a worktree, end the reply with the copy-paste
   run command** for that worktree. See [Worktrees](#worktrees).

## Project

Polyglot is a cross-platform (macOS, Windows, Linux) interactive terminal app for
learning languages. v1 focuses on **Spanish → Japanese**. It ships as a single
self-contained binary. The architecture is built to add more language pairs later.

## Operating model

Development is organized as a set of parallel **tracks**, coordinated through the GitHub
**Project board** (Projects v3, user project #3) — the single source of truth for what to
work on next.

### Tracks

- **Core / Platform** (built once; a prerequisite for everything): the engine and shared
  building blocks — content model/schema, spaced-repetition system, Story Mode framework,
  mastery gates, and the resource-licensing/attribution tooling. Built once and reused by
  every language pair.
- **Language-pair tracks** (parallel, one per pair — e.g. `es→ja`, `ja→es`, …): build the
  content and cultural skin on top of Core. Each pair is its own learning experience,
  because the learner's native language (L1) shapes it.

Each track advances in its own git worktree (see [Worktrees](#worktrees)). Board items
carry the track they belong to.

### Session start ritual

1. Read this file and `MANIFESTO.md` for direction.
2. Open the Project board and pick the **next task**: the highest-priority item in the
   active track whose dependencies are met (Core before pair work). If no track is
   specified, surface the top candidates and confirm before starting.
3. Claim it — set its Status to **In Progress** via the `gh` GraphQL API.
4. Follow the [Workflow checklist](#workflow-checklist).
5. If the roadmap needs a task that isn't on the board yet, propose it (and add the board
   item) rather than going off-plan silently.

### Pedagogical grounding

Ground learning-design decisions in published standards — the **CEFR** plus the relevant
per-language proficiency framework (e.g. JLPT for Japanese) and frequency data — and in
learning science (cognitive load, decoding before reading, retrieval practice, spaced
repetition), not in intuition. The rationale lives in `MANIFESTO.md`.

### Open-source citizenship

Follow the practices in <https://opensource.guide/>: maintain the project's open-source
docs (`LICENSE`, `README`, `CONTRIBUTING`, a code of conduct), creating them when missing;
write clear issues and PRs; be welcoming and document decisions.

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

- **License is MIT — and it applies to everything in the repo, not just code.** Only add
  **permissive** code dependencies *and content/assets*: frequency lists, corpora,
  dictionaries, fonts, audio, images, lesson text. **Allowed:** Public Domain / CC0,
  CC BY (with attribution), MIT / BSD / Apache. **Rejected:** non-commercial (CC BY-NC),
  copyleft / share-alike (CC BY-SA, GPL), and anything with unclear provenance. Record
  every external asset's source, license, and required attribution (e.g. in `NOTICE`).
  When a license is unclear or incompatible, stop and flag it — prefer public-domain
  sources or content we author ourselves; don't assume "it's just facts".
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
- **Merging:** never merge a PR unless explicitly asked to.
- **Versioning:** Semantic Versioning. Keep `CHANGELOG.md` updated
  ([Keep a Changelog](https://keepachangelog.com/) format).
- **Project board:** when a PR is merged, move its corresponding item in the GitHub
  Project (Projects v3, user project #3) from its current status (e.g. `Todo`) to
  `Done`. Use the `gh` GraphQL API (`updateProjectV2ItemFieldValue` on the Status
  field) to set it.

### Worktrees

- **Before touching any file, start the change in its own git worktree** so the
  working copy and branch stay isolated: `git worktree add ../polyglot-<feature> -b <feature>`,
  and `git worktree remove ../polyglot-<feature>` once the PR is merged. This is
  step 1 for every change, including docs-only ones like editing this file.
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
golangci-lint run ./...  # lint (staticcheck etc.; stricter than go vet, matches CI)
gofmt -l .               # formatting check
```
