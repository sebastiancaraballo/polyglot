# genfreq — Japanese word-frequency list generator

`genfreq` derives `content/ja/frequency.tsv` (a ranked Japanese word-frequency
list) from the **Tatoeba** Japanese sentences corpus.

It is an **offline** build-time tool and lives in its **own Go module** so that
its tokenizer dependency never enters the shipped binary's module graph:

- [`kagome`](https://github.com/ikawaha/kagome) — pure-Go morphological
  analyzer, **MIT**.
- MeCab **IPADIC** (bundled by `kagome-dict/ipa`) — BSD-style dictionary.

The repository ships only the derived list, not the tokenizer and not the raw
corpus.

## Reproduce

```sh
# 1. Download the Tatoeba Japanese sentences export (CC BY 2.0 FR).
curl -fsSLO https://downloads.tatoeba.org/exports/per_language/jpn/jpn_sentences.tsv.bz2
bunzip2 jpn_sentences.tsv.bz2

# 2. Regenerate the list (run from this directory).
cd tools/genfreq
go run . -in /path/to/jpn_sentences.tsv -out ../../content/ja/frequency.tsv -n 10000
```

Flags: `-in` (required, Tatoeba export path), `-out` (default
`../../content/ja/frequency.tsv`), `-n` (keep the N most frequent words, default
`10000`).

## Provenance

- **Corpus:** Tatoeba Project Japanese sentences, snapshot **2026-06-25**.
- **License:** CC BY 2.0 FR (attribution-only). Attribution is recorded in the
  repo-root `NOTICE`; the rationale and rejected alternatives are in
  [`docs/adr/0001-japanese-frequency-list.md`](../../docs/adr/0001-japanese-frequency-list.md).

Output is deterministic for a fixed corpus snapshot and IPADIC version (ties are
broken by surface form).
