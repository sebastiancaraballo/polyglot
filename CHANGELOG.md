# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Kanji support in the engine** (#62). The engine can now be taught kanji: a
  `KanjiItem` model with on'yomi/kun'yomi readings and a meaning, an optional
  `content/<pair>/kanji/*.yaml` table, `kanji_progress` persistence (migration 11)
  and accuracy-based grading. This is the engine half only — the learning UI and
  the ~100 N5 kanji themselves are follow-ups, and no pair teaches kanji yet, so
  nothing changes for learners today.

### Fixed
- Kanji were validated by neither half of the engine and rejected by the other:
  the content loader skipped every non-kana character, so a card written with
  kanji passed validation, while the decoder refused all Han characters
  outright — the card would have loaded cleanly and then never appeared in a
  single study session. The loader now requires every kanji in evaluable content
  to be in the kanji table, and the decoder accepts the ones a learner has
  mastered. The `is_han` check, previously duplicated with two different sets of
  Unicode ranges, is now defined once.

## [0.0.3] - 2026-08-01

### Changed
- **Rewritten in Rust; the Go implementation is retired.** The app is now a Cargo
  workspace: `core/` holds the language-agnostic engine and `term/` the ratatui
  terminal client. The engine is UI-free, which is what lets a future
  universal/offline client (mobile/desktop over FFI) reuse it — the motivation
  for the rewrite (see issue #58). Everything the Go version did, it still does:
  - **Core:** SM-2 scheduler (`srs`), domain model, study logic (kana
    automaticity, decoding gate, Rikai, challenge, N5 assessment sampling), the
    YAML/Markdown content loader and its validators (embedding the real `es-ja`
    course via `include_dir`), Spanish `i18n`, the third-party-asset `license`
    manifest — whose generated `NOTICE` matches the Go one byte for byte — SQLite
    storage via `rusqlite` (bundled, no system dependency), and the
    cross-curriculum spaced-repetition `review` queue.
  - **Client:** a ratatui event loop with idle-tick animation, screen-stack
    router, `NO_COLOR`/high-contrast theme, the fixed 64×23 frame, the animated
    braille globe and block wordmark, and all 13 screens over the real core with
    progress persistence.
  - **Your progress carries over.** The migration runner adopts databases created
    by the Go implementation's `goose` migrations: the resulting schema is
    identical (verified by applying both and diffing), and an existing profile
    opens in the Rust client with its XP, streak, card scheduling, kana mastery,
    grammar and story progress intact.
  - **Tooling:** CI and the release workflow now build, test, lint and package
    with Cargo (`.goreleaser.yaml` is gone). `POLYGLOT_DB` overrides the database
    path, so you can run against a scratch database instead of real progress.
  - The offline asset generators under `tools/` (`genfreq`, `genglobe`,
    `gennotice`) are still Go. They are not part of the binary and have no Rust
    equivalent yet.
- **Main menu layout.** The options now lead on the left and the rotating globe
  sits on the right, vertically centered against them; XP and streak moved below
  the options, where they read as context rather than as a header.

### Fixed
- The block wordmark lost the trailing padding on two of its four rows when it
  was ported, leaving it 53 columns wide instead of 55.
- A menu level long enough to overflow the frame's fixed content height now drops
  the wordmark instead of silently clipping its own rows off-screen, matching the
  Go behavior.
- `core/src/content/coverage.rs` was matched by an unanchored `coverage.*` rule in
  `.gitignore` — a source file of the engine was never committed, so a fresh
  clone did not compile.

## [0.0.2] - 2026-07-09

### Added
- Reverse kana trainer (romaji → kana recall): the kana trainer now offers a second drill direction alongside the existing recognition drill (glyph shown, pick the romaji). In reverse mode the prompt is the romaji reading and the four options are kana glyphs, so the learner must *produce* the character — the retrieval direction that recognition alone doesn't exercise (paired-associate learning). The direction is a toggle in the group picker (←/→), shown as a "Dirección:" line; it reuses all the existing session machinery — the same groups, the hiragana→katakana gate, SRS grading, and XP. Mastery tracks the character, not the presentation: a correct answer in either direction advances the same per-kana streak (no schema change). Options render with the accent kana styling so glyphs stay legible, and the layout is frame-fit-tested in both directions.
- N5 mock assessment (capstone level exam): a new "Examen N5" activity that certifies the level with a cross-curriculum retrieval test. It samples up to 15 questions round-robin across the three learnable strands — vocabulary (recall a word's Japanese form), kana (read a character), and grammar (fill a blanked slot in a Rikai pattern) — and grades against the same 80% mastery band as the end-of-chapter challenge (`study.ChallengePassed`). Like the challenge, every answer flows through the regular spaced-repetition, XP, and mastery paths (vocab/pattern fillers as SRS reviews, kana as an automaticity streak, pattern slots as a correctness streak), so a failed attempt is still learning and retries are immediate. It is unlocked only after mastering every N5 story chapter (the capstone gate) and its result is persisted (`assessment_result` table): the pass is sticky (Mastery Learning — never revoked) and the best score is kept, surfaced as a "✓ aprobado" badge on the menu entry. The exam screen wraps all prose to the frame and caps its review list by height, and a test renders every phase/question kind against the frame to guard against overflow.
- es-ja N5 story arc completed: two new Katsudoo chapters continue Yui's guided tour from where chapter 2 left off — "Direcciones en Sensō-ji" (asking and understanding directions at the temple grounds, presenting/practicing `question-words` and `positions`) and "La vida diaria" (a café conversation about daily routine, presenting/practicing `time-clock` and `verbs-daily`). Each chapter drills two vocabulary pools instead of one, exercising the end-of-chapter challenge's round-robin sampling across pools. The N5 story arc (arrival, greetings, shopping, directions, daily routine) is now complete at 4 chapters.
- es-ja N5 grammar via Rikai: nine new grammar patterns covering the particles は/が/を/に/で, the copula extended to demonstratives (これ/それ/あれ), i- and na-adjective predicates, the polite verb triad ~masu/~masen/~mashita (as a slot-varying contrast, not just vocabulary), and the て-form request ...てください. Two new vocabulary lessons back the tense/request patterns with the conjugated surface forms the frame-substitution engine needs (たべます/のみます present/negative/past, and the て-forms of たべる/のむ/みる/よむ/かく/いく), each tagged with a new communicative function (`polite-verb-forms`, `polite-requests`). All slot fillers are drawn from already-shipped N5 vocabulary, so every pattern is drillable as soon as its words are known.
- es-ja N5 vocabulary completed to JLPT scope: 40 new thematic lessons bring the course to **~800 self-authored N5 (CEFR A1–A2) cards across 47 lessons** — verbs (daily, movement, giving/using, more), i- and na-adjectives, adverbs, question words, demonstratives, positions, time (relative time, days, months, dates, the clock), large numbers and counters, family, people & occupations, body, places, the home, nature & weather, animals, clothing, transport, food & drink & restaurant, school, daily-life nouns, countries & languages, feelings, hobbies, conjunctions and courtesy expressions. Each lesson is tagged with a communicative function in the shared spine (29 new A1–A2 functions added). All words are self-authored (no proprietary JLPT list copied) and written in kana, so they pass the decodability gate and enter study ordered by frequency automatically.
- es-ja N5 core vocabulary bank: four new thematic lessons — Colores (10), Comida y bebida (12), Objetos cotidianos (12) and Adjetivos básicos (13) — adding ~47 N5 (CEFR A1) cards, each tagged with a new communicative function in the shared spine (`describe-colors`, `talk-food-drink`, `name-everyday-objects`, `describe-qualities`). All words are self-authored (selection scoped to N5/A1 topics and sanity-checked against the permissive frequency list; Spanish glosses and romaji written in-house — no proprietary JLPT list is copied) and written entirely in kana, so they pass the decodability gate and enter study by frequency automatically. Words that require the sokuon (っ) or the katakana chōonpu (ー) are deliberately deferred until those marks are teachable (see the follow-up ticket).
- Present-before-practice for Katsudoo: a new story beat kind, `present`, diegetically introduces a pool of material (a vocabulary lesson or a kana set) — rendering its items with readings and glosses — so the learner *meets* the material before a practice beat or the end-of-chapter challenge asks them to retrieve it (retrieval practice, the testing effect, operates on studied material — quizzing what was never taught is a content bug, not a design). The content loader now enforces this at validation time (`checkStoryPresentation`, language-agnostic like `checkStoryCoverage`): a chapter may only practice a pool it — or an earlier chapter on the mastery-gated linear path — has already presented, otherwise `Load` fails. Present beats carry an optional diegetic framing line (speaker + es/jp/romaji) above the presented list, which paginates within the fixed frame when the pool is longer than one screen (so large lessons never clip the frame — the learner turns pages before the practice that follows).
- Frequency-based card sequencing: the content loader now backfills each vocabulary card's frequency rank (`Card.Freq`) from the target language's embedded frequency list (`content/<lang>/frequency.tsv`, optional per language), matching a card's Japanese form against the list's surface word first and its reading second — so kana-only cards match kanji entries like 私/わたし. An explicit `freq:` in the lesson YAML always wins; unmatched words (including multi-token expressions like おねがいします) stay unranked and sort last. The rank drives the order in which *new* cards enter study sessions — most frequent words first (frequency-driven vocabulary acquisition, Nation) — while lesson and card authoring order stays curricular, and the rank is surfaced on the flashcard reveal ("Frecuencia: nº N"). Known limitation, documented: kana homographs may borrow a more frequent homograph's rank (e.g. に "dos" matches the particle に); the YAML override is the escape hatch.
- Mastery gates: each Katsudoo chapter now ends in a short retrieval-practice challenge (5 questions drawn from the chapter's practice pools; pass at 80% — Bloom's mastery band, stated up front on an intro screen) that must be passed to unlock the next chapter in the picker (Mastery Learning: advance on demonstrated mastery, not on having clicked through). Every challenge answer is graded through the same spaced-repetition paths as regular practice, so a failed attempt is still learning (testing effect, Roediger & Karpicke) and its missed cards come due immediately; retries are immediate with a fresh draw, mastery is never revoked, and a chapter with no practice beats auto-masters on completion. The picker distinguishes "visto · reto pendiente" from "✓ dominado" (new `mastered` column in `story_progress`), locked chapters show the kana-trainer lock treatment with a hint naming the chapter to master, and a standing note states the gating rule. Ships with a second es-ja chapter ("Compras en Nakamise": counting/shopping on Nakamise street, practicing the `numbers` lesson) so the gate is real.
- Adaptive new-vs-review pacing in the cross-curriculum review queue (`review.NewCardBudget`): due reviews always take priority; new (never-reviewed) cards only fill session seats reviews don't need, capped at 10 per session, and a lapse-heavy due set (half or more of due reviews have lapsed — finally consuming the stored `Lapses` signal) halves the intake. When new cards are held back, the Flashcards/Repaso screen says so and why ("%d tarjetas nuevas en espera: entran poco a poco para consolidar lo aprendido."), keeping the pacing rule visible rather than silent.
- Katsudoo, the communicative-activity story framework: a new content type, the story chapter (`content/<pair>/story/*.yaml`), models an ordered sequence of narration/dialogue/practice beats. Practice beats pause the story for one inline check that reuses the exact same grading logic as the real kana trainer and quiz screens (`study.GradeKana`, `srs.Review`, the same storage calls) rather than embedding those screens — the current navigation architecture has no "return to caller" mechanism, so this follows the same bespoke-inline-check precedent already used by first-run onboarding's practice step. Progress is tracked per chapter (`story_progress` table: beat reached, completed), so leaving mid-chapter and returning resumes where the learner left off. A new "Katsudoo" menu entry runs it, ungated. Ships with one small example chapter for es-ja — an arrival scene in Asakusa with a fictional, dignified local-guide character, and one practice beat reusing the existing `greetings` vocabulary lesson.
- Rikai, the knowledge-strand grammar drill: a new content type, the grammar pattern (`content/<pair>/grammar/*.yaml`), models a fixed sentence frame with one or more slots filled by vocabulary the learner already knows ("words before sentences", enforced at content-validation time). Each drill round varies exactly one slot while the rest stay fixed at a default filler (Cognitive Load Theory / Processing Instruction), cycling round-robin through a pattern's slots. Mastery is a correctness-only streak tracked per pattern slot (`internal/study.GradePatternSlot`, persisted in a new `pattern_progress` table), matching the accuracy-only precedent set for kana. A pattern is gated behind having at least one known filler (survived ≥1 spaced-repetition review) per slot; the new "Rikai" menu entry shows the same locked-icon treatment as the reading activities until then. Ships with the classic N5 copula pattern "X wa N desu" and a small self-introduction vocabulary lesson (watashi, anata, gakusei, sensei, nihonjin) to exercise it end-to-end.
- Block-letter "POLYGLOT" wordmark spanning the main menu header, echoing the README logo in the same solid-block style but sized to fill the shared fixed-width frame (`internal/art`). It stands in for the plain text app name and falls back to the text title on terminals too narrow to fit it.
- Kana trainer first-time onboarding: the first time a learner opens the kana trainer they see a short intro explaining the gated path (hiragana → katakana → reading) and what "dominar" (mastery) means — fast, accurate recognition. It is shown once per profile (persisted via a new `kana_onboarded` flag) and dismissed forever. The picker's locked-group hint now shows **live** progress toward the unlock (e.g. `23/46`), and a standing note explains what the per-group counts mean.
- Foundations decoding gate (Hiragana & Katakana to automaticity): the kana trainer now times each answer and tracks per-kana progress toward *automaticity* — a run of correct, fast answers (`internal/study.GradeKana`), persisted per profile in a new `kana_progress` table. Grounded in the Simple View of Reading (decoding must precede comprehension), the curriculum is gated two ways: katakana practice is locked until the hiragana base gojūon is fluent (`internal/study.Gate`), and reading is **progressive** — following the decodable-texts approach, the reading activities (Flashcards, Quiz, Repaso) present only the words and sentences built entirely from kana the learner has already mastered (`internal/study.Decoder`), so the readable set grows as they learn. Reading is locked on the main menu only while nothing is decodable yet. The kana picker shows per-group mastery counts and a fluency badge; locked items show a lock marker and an explanatory hint.
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
- Main menu reorganized as a grouped, drill-down navigation instead of one flat list. The top level now holds a handful of category rows — **Aprender** (Kana, Flashcards, Rikai), **Leer** (Katsudoo, Tabla de Kana), **Evaluar** (Repaso, Quiz, Examen N5), **Herramientas** (Estadísticas) — plus the standalone **Ajustes** and **Salir**; ENTER opens a category and ESC/← goes back. This stops the top-level screen from growing one row per new Core screen and blowing the frame's fixed content height, and — as a direct result — the block "POLYGLOT" wordmark fits the header again and reappears (its show-if-it-fits logic is unchanged). Locked activities (Flashcards/Quiz, Rikai, Examen N5) keep the exact same gating and lock-notice treatment, now shown inside their category submenu. The globe/info layout, wordmark, and fixed frame are untouched.
- Sokuon (small tsu っ/ッ) and chōonpu (long-vowel mark ー) are now treated as always-decodable kana marks by both the content kana-coverage check and the reading decoder (`study.Decoder`): they modify an adjacent, separately-teachable kana and have no reading of their own, so they never gate a word's decodability and are not taught as their own kana-trainer items. This unblocks the large set of common N5 vocabulary that uses them (がっこう, きって, ちょっと, コーヒー, ノート…).
- es-ja story chapters 1 (Asakusa, greetings) and 2 (Nakamise, numbers) now present their vocabulary before practicing it, via the new `present` beat — closing the gap where the numbers challenge in chapter 2 tested counting the story had never taught.
- Kana mastery now depends on accuracy only: a kana's streak advances on every correct answer and resets only on an incorrect one — response time no longer gates the streak (`internal/study.GradeKana`). Answer time is still recorded as each kana's best-time stat. The onboarding and picker copy no longer say mastery requires answering "rápido". Removes the opaque speed requirement that could leave a kana un-mastered with no visible reason even when answered correctly.
- The shared app frame is one row taller, giving every screen a blank line between its content and the bottom keyboard-help line (e.g. after the main menu's "Salir" entry).
- README landing rewritten to lead with a title banner and a dedicated "Supported languages" section instead of a feature list. Installation and everything below it are unchanged.
- The learned-progress figure now counts the whole curriculum: its label is "tarjetas aprendidas" (cards learned) and its total includes kana as well as vocabulary, keeping it coherent now that kana is scheduled by spaced repetition.
- Main menu header redesigned around the globe: the rotating globe and the app/menu content now render as two vertically centered columns, with the keyboard help pinned to the bottom of the frame.
- Screens now render inside a fixed-size frame whose dimensions depend only on the terminal size, so the border no longer grows or shrinks with its content (e.g. when a quiz reveals an answer) or when moving between sections.

### Fixed
- Text no longer overflows the frame's right border in the story, quiz, and Rikai screens. Two root causes: (1) several views rendered prose raw — story narration/dialogue, and the quiz/Rikai/story-practice question prompt (`¿Cómo se dice "…"?` with a long Spanish gloss such as "Cuál (de dos) / Dónde (cortés)") — so long lines were clipped; they now wrap to the frame width. (2) `ui.WrapText` only broke on spaces (`strings.Fields`), so space-less Japanese was kept as one unbreakable token that overflowed even present-beat framing lines that already called `WrapText`; it now hard-breaks an over-wide token by display width (honoring wide CJK runes). A new test renders every beat and present-beat page of every embedded chapter — checking every practice prompt against the widest option set — and fails if anything exceeds the frame's width or height.
- Main menu no longer clips its own bottom border. With the current number of menu items, the header block wordmark plus the globe/info columns exceeded the frame's fixed content height (an upper bound that does not grow with a taller terminal), which silently truncated the frame instead of overflowing visibly. The wordmark now also drops — falling back to the plain-text title, as it already did for narrow terminals — whenever it doesn't fit vertically, not just horizontally.
- Kana trainer "Todo" (All) group is now gated like the katakana groups: because it spans both syllabaries it stays locked until the hiragana base is fluent, closing a hole where a learner could practice katakana through "Todo" before the hiragana→katakana gate opened.
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

[Unreleased]: https://github.com/sebastiancaraballo/polyglot/compare/v0.0.2...HEAD
[0.0.2]: https://github.com/sebastiancaraballo/polyglot/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/sebastiancaraballo/polyglot/releases/tag/v0.0.1
