# 0001 — Japanese word-frequency list source

- **Status:** Accepted
- **Date:** 2026-06-25

## Context

Sequencing vocabulary by frequency is a core pedagogical lever (see
`MANIFESTO.md`), and the curriculum schema already carries an optional `Card.Freq`
slot. We need a Japanese word-frequency list whose license is compatible with
the project's MIT promise — which, per `CLAUDE.md` "Hard constraints", applies to
content and data, not just code. Allowed: Public Domain / CC0, the CC-BY family
(no `-NC`, no `-SA`), MIT / BSD / Apache. Rejected: NonCommercial, ShareAlike /
copyleft, and anything of unclear provenance.

## Options considered

- **`hermitdave/FrequencyWords`** — popular pre-computed lists. *Rejected:* the
  content is **CC BY-SA** (share-alike), and the underlying OpenSubtitles
  provenance is ambiguous for commercial redistribution.
- **`scriptin/kanji-frequency`** — CC BY 4.0. *Rejected:* partly derived from
  **Japanese Wikipedia (CC BY-SA)**, tainting the aggregate, and it is
  kanji-character frequency, not word frequency.
- **`MarvNC/JP-Resources` / JPDB-derived lists** — *Rejected:* scraped from a
  third-party site; provenance and redistribution rights are unclear.
- **Derive our own from a clean corpus.** *Chosen.* No ready-made permissive
  Japanese *word*-frequency list exists, so we generate one from a corpus whose
  license we can stand behind.

## Decision

Derive the list from the **Tatoeba Project** Japanese sentences corpus
(<https://tatoeba.org>), licensed **CC BY 2.0 FR** — attribution-only, with no
share-alike clause, and largely composed of the public-domain Tanaka Corpus. The
register (everyday sentences) suits a learner app.

Tokenization uses **kagome** (pure Go, MIT) with MeCab **IPADIC** (BSD-style),
in the offline tool `tools/genfreq`, which is a **separate Go module** so the
tokenizer and dictionary never enter the shipped binary's dependency graph. We
commit only the derived `content/ja/frequency.tsv`; we do **not** commit the raw
corpus.

## Consequences

- **Attribution is mandatory** (CC BY 2.0 FR) and is recorded in the repo-root
  `NOTICE`, generated from `internal/license/assets.yaml`.
- **Reproducibility:** output is pinned to the Tatoeba snapshot (**2026-06-25**)
  and the IPADIC version; the exact command lives in `tools/genfreq/README.md`.
- **Backfill deferred:** this work imports and validates the list but does not
  yet wire it into `Card.Freq`. Consuming it to sequence cards is the separate
  "frequency backfill" ticket.
