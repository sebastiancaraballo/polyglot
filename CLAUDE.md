# CLAUDE.md

Guidance for working in this repository. Read before making changes.

## Stack

Polyglot is a **Rust** workspace: `core/` is the language-agnostic engine (content
model, spaced repetition, study logic, storage) and `term/` the ratatui terminal
client. The engine is UI-free on purpose — it is what a future mobile/desktop
client reuses over FFI (see issue `#58`).

The Go implementation this replaced was retired in the cutover; its history is in
the git log. The only Go left in the repo is `tools/`, three offline asset
generators (`genfreq`, `genglobe`, `gennotice`) that are not part of the binary
and have no Rust equivalent yet.

```sh
cargo run -p polyglot-term                                 # run the app
cargo build --workspace
cargo test --workspace
cargo clippy --workspace --all-targets -- -D warnings
cargo fmt --all
```

## Workflow checklist

Follow this order for **every** change. The ★ steps are process steps that nothing
will compile-fail or auto-catch, so they are the ones most often missed — do not
skip them. Each item links to its authoritative detail below.

1. ★ **Before editing any file, create a worktree** for the change:
   `git worktree add ../polyglot-<feature> -b <feature>`. See [Worktrees](#worktrees).
2. Write code, comments, and commits in English; keep user-facing strings in
   `core/src/i18n.rs` (Spanish). See [Language conventions](#language-conventions).
3. Only permissive dependencies. See [Hard constraints](#hard-constraints).
4. ★ **Update `CHANGELOG.md` and any affected docs (e.g. README) in the same change.**
5. ★ **Before finishing, run and pass** `cargo fmt --all`, `cargo clippy
   --workspace --all-targets -- -D warnings`, and `cargo test --workspace`
   (regenerate snapshots with `INSTA_UPDATE=always`, and review every changed
   snapshot before accepting it). See [Quality](#quality).
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
  them in `core/src/i18n.rs` so more UI languages can be added later.

## Tech stack & layout

- **Rust 1.90+**, a Cargo workspace: `core/` (engine, no UI) + `term/` (ratatui TUI).
- **TUI:** [ratatui](https://ratatui.rs) with the crossterm backend. Screens are plain
  structs with `handle`/`render`; navigation is a screen stack driven by `Transition`.
- **Persistence:** SQLite via `rusqlite` (bundled, no system dependency). Schema
  migrations are a `user_version`-tracked runner in `core/src/storage/migrations.rs`,
  which also adopts databases created by the retired Go implementation's `goose`
  migrations — never renumber an existing migration. The database lives under the
  platform config dir (`dirs::config_dir()/polyglot`); `POLYGLOT_DB` overrides it,
  which is how you run against a scratch database instead of your real progress.
  Local profiles: progress is keyed by `profile_id`.
- **Content:** YAML lessons + Markdown guides under `content/<pair>/` (v1: `es-ja/`),
  embedded with `include_dir`. Cards are tagged with their JLPT level.

## Hard constraints

- **License is MIT — and it applies to everything in the repo, not just code.** Only add
  **permissive** code dependencies *and content/assets*: frequency lists, corpora,
  dictionaries, fonts, audio, images, lesson text. **Allowed:** Public Domain / CC0,
  CC BY (with attribution), MIT / BSD / Apache. **Rejected:** non-commercial (CC BY-NC),
  copyleft / share-alike (CC BY-SA, GPL), and anything with unclear provenance. Record
  every external asset's source, license, and required attribution (e.g. in `NOTICE`).
  When a license is unclear or incompatible, stop and flag it — prefer public-domain
  sources or content we author ourselves; don't assume "it's just facts".
- **Ship a single self-contained binary.** No system libraries at runtime (this is
  why `rusqlite` is used with its `bundled` feature).
- **Never commit a user database** (`*.db` is gitignored).

## Accessibility

Responsive layout (ratatui reflows on resize; lay out against the frame's content
rect, never a fixed width), high-contrast theme,
honor `NO_COLOR`, keyboard-first navigation, never rely on color alone (pair with
symbols/text), keep romaji visible alongside Japanese.

## Quality

- Format with `cargo fmt --all`; keep `cargo clippy --workspace --all-targets --
  -D warnings` clean — CI fails on either.
- Tests: `cargo test --workspace`. Prefer table-driven unit tests. Validate all YAML
  content in tests. For TUI screens, snapshot the rendered frame with `insta` via
  `testutil::snapshot` (plain theme, escape-free output); screens that draw with an
  RNG get behavior tests in their own module instead. Regenerate snapshots with
  `INSTA_UPDATE=always cargo test`, and **read every changed snapshot** before
  accepting it — an unreviewed snapshot records a bug as if it were the spec.
- Accessibility is testable: assert symbols and text, never color alone.

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
- Tests open their own in-memory databases, so they never touch real progress. The
  app's real database is only used by `cargo run -p polyglot-term`; set `POLYGLOT_DB`
  to a scratch path when running from a worktree.
- **At the end of an implementation done in a worktree, always finish the reply
  with the copy-paste command to run the app from that worktree**, so it's easy to
  try locally:

  ```sh
  cd ../polyglot-<feature> && cargo run -p polyglot-term
  ```

  (Note: this uses the real database per the caveat above. Set `POLYGLOT_DB` to a
  scratch path to leave your own progress alone.)

## Common commands

```sh
cargo run -p polyglot-term                             # run the app
cargo build --workspace                                # build
cargo test --workspace                                 # tests
cargo clippy --workspace --all-targets -- -D warnings  # lint (CI fails on warnings)
cargo fmt --all                                        # format
INSTA_UPDATE=always cargo test --workspace             # refresh TUI snapshots
```
