//! Content loading and validation for a single language pair.
//!
//! Port of the Go `internal/content` package. The Go `fs.FS` abstraction
//! becomes the [`ContentFs`] trait, implemented for the embedded bundle
//! ([`EmbeddedFs`]) and a real directory ([`DirFs`], for tests). YAML is parsed
//! with `serde`; content is embedded with `include_dir` (the `go:embed`
//! equivalent).

mod coverage;
mod freq_backfill;
mod frequency;
mod fsys;
mod functions;
mod grammar;
mod kana;
mod kanji;
mod lessons;
mod story;

use std::fmt;
use std::io;

pub use fsys::{ContentFs, DirFs, EmbeddedFs};

use crate::model::{Chapter, FreqEntry, KanaItem, KanjiItem, Lesson, Pattern};

/// The language pair shipped in v1.
pub const DEFAULT_PAIR: &str = "es-ja";

/// The content bundle embedded into the binary (the `go:embed` equivalent).
static CONTENT: include_dir::Dir<'static> =
    include_dir::include_dir!("$CARGO_MANIFEST_DIR/../content");

/// The fully loaded and validated content for a single language pair.
#[derive(Clone, Debug)]
pub struct Course {
    pub pair: String,
    pub lessons: Vec<Lesson>,
    pub kana: Vec<KanaItem>,
    /// Teachable kanji. Empty for a pair that teaches none.
    pub kanji: Vec<KanjiItem>,
    pub patterns: Vec<Pattern>,
    pub chapters: Vec<Chapter>,
}

/// An error loading or validating course content.
#[derive(Debug)]
pub struct LoadError(String);

impl LoadError {
    pub(crate) fn new(msg: impl Into<String>) -> Self {
        LoadError(msg.into())
    }
}

impl fmt::Display for LoadError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl std::error::Error for LoadError {}

/// Loads a course from the content bundled into the binary.
pub fn load_embedded(pair: &str) -> Result<Course, LoadError> {
    load(&EmbeddedFs::new(&CONTENT), pair)
}

/// Loads the word-frequency list for a target language from the content bundled
/// into the binary (e.g. lang `"ja"`).
pub fn load_embedded_frequency(lang: &str) -> Result<Vec<FreqEntry>, LoadError> {
    frequency::load_frequency(&EmbeddedFs::new(&CONTENT), lang)
}

/// Reads, parses, and validates the course for `pair` from `fsys`. Paths are
/// resolved as `<pair>/{lessons,kana,grammar,story}/*.yaml`. The
/// language-agnostic function catalog under `functions/*.yaml` is loaded once
/// and used to resolve the communicative functions each lesson references.
pub fn load(fsys: &dyn ContentFs, pair: &str) -> Result<Course, LoadError> {
    let catalog = functions::load_functions(fsys)?;
    let mut lessons = lessons::load_lessons(fsys, pair, &catalog)?;
    let kana = kana::load_kana(fsys, pair)?;
    let kanji = kanji::load_kanji(fsys, pair)?;
    let patterns = grammar::load_patterns(fsys, pair)?;

    // Every kana and kanji a card depends on must be teachable (present in the
    // tables). Without the kanji half, a card written with kanji would load
    // cleanly and then be undecodable forever — valid content nobody ever sees.
    let set = coverage::kana_set(&kana);
    let kanji_set = coverage::kanji_set(&kanji);
    for lesson in &lessons {
        for card in &lesson.cards {
            coverage::check_kana_coverage(&card.jp, &set)
                .map_err(|e| LoadError::new(format!("card {:?} {e}", card.id)))?;
            coverage::check_kanji_coverage(&card.jp, &kanji_set)
                .map_err(|e| LoadError::new(format!("card {:?} {e}", card.id)))?;
        }
    }

    // Every pattern's slot fillers must be vocab the learner is actually taught.
    let card_set = coverage::card_id_set(&lessons);
    for p in &patterns {
        coverage::check_vocab_coverage(p, &card_set)
            .map_err(|e| LoadError::new(format!("pattern {:?} {e}", p.id)))?;
    }

    let chapters = story::load_chapters(fsys, pair)?;

    // Every practice/present beat must reference content that actually exists.
    let lesson_ids = coverage::lesson_id_set(&lessons);
    let kana_types = coverage::kana_types_present(&kana);
    for c in &chapters {
        coverage::check_story_coverage(c, &lesson_ids, &kana_types)
            .map_err(|e| LoadError::new(format!("chapter {:?}: {e}", c.id)))?;
    }
    // A chapter may only practice material it (or an earlier chapter) presented.
    coverage::check_story_presentation(&chapters)?;

    // Backfill each card's frequency rank from the target language's list, when
    // one ships (`<lang>/frequency.tsv`). Like grammar and story content, the
    // list is optional: a pair without one simply has unranked cards.
    let lang = target_lang(pair);
    if !lang.is_empty() {
        let freq_file = format!("{lang}/frequency.tsv");
        match fsys.read(&freq_file) {
            Ok(data) => {
                let entries = frequency::parse_frequency(&freq_file, &data)?;
                freq_backfill::backfill_freq(&mut lessons, &freq_backfill::freq_index(&entries));
            }
            Err(e) if e.kind() == io::ErrorKind::NotFound => {} // leave unranked
            Err(e) => return Err(LoadError::new(format!("read {freq_file}: {e}"))),
        }
    }

    Ok(Course {
        pair: pair.to_string(),
        lessons,
        kana,
        kanji,
        patterns,
        chapters,
    })
}

/// Extracts the target language from a pair like `"es-ja"` (`""` when the pair
/// has no source-target form).
fn target_lang(pair: &str) -> &str {
    match pair.rfind('-') {
        Some(i) => &pair[i + 1..],
        None => "",
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::model::{Jlpt, KanaCategory, KanaType};
    use std::collections::{HashMap, HashSet};
    use std::path::PathBuf;
    use std::sync::atomic::{AtomicU32, Ordering};

    /// A throwaway on-disk content tree, the `fstest.MapFS` equivalent: the Rust
    /// loader reads through [`ContentFs`], whose test backend is a real
    /// directory. Removed on drop so a failing assertion leaks nothing.
    struct Fixture {
        root: PathBuf,
    }

    impl Fixture {
        /// Writes `files` (content-root-relative path, YAML body) into a fresh
        /// scratch directory.
        fn new(files: &[(&str, &str)]) -> Fixture {
            static N: AtomicU32 = AtomicU32::new(0);
            let n = N.fetch_add(1, Ordering::Relaxed);
            let root =
                std::env::temp_dir().join(format!("polyglot-content-{}-{n}", std::process::id()));
            std::fs::create_dir_all(&root).unwrap();
            for (path, body) in files {
                let full = root.join(path);
                std::fs::create_dir_all(full.parent().unwrap()).unwrap();
                std::fs::write(&full, body).unwrap();
            }
            Fixture { root }
        }

        fn load(&self, pair: &str) -> Result<Course, LoadError> {
            load(&DirFs::new(&self.root), pair)
        }
    }

    impl Drop for Fixture {
        fn drop(&mut self) {
            std::fs::remove_dir_all(&self.root).ok();
        }
    }

    const VALID_KANA: &str = "type: hiragana\nitems:\n  - char: あ\n    romaji: a\n";

    /// A single vocab card `a:1` for pattern and story tests to reference.
    const VALID_PATTERN_LESSON: &str =
        "id: a\ntitle: t\njlpt: N5\ncards:\n  - es: Hola\n    jp: あ\n    romaji: a\n";

    /// The minimal well-formed pair used by the happy-path tests.
    fn valid_files() -> Vec<(&'static str, &'static str)> {
        vec![
            (
                "functions/core.yaml",
                "functions:\n  - id: greet-daytime\n    cefr: A1\n    description: Saludar.\n",
            ),
            (
                "xx/lessons/01.yaml",
                "id: greetings\ntitle: Saludos\njlpt: N5\nfunctions: [greet-daytime]\ncards:\n  - es: Hola\n    jp: こんにちは\n    romaji: konnichiwa\n",
            ),
            (
                "xx/kana/h.yaml",
                "type: hiragana\nitems:\n  - char: こ\n    romaji: ko\n  - char: ん\n    romaji: n\n  - char: に\n    romaji: ni\n  - char: ち\n    romaji: chi\n  - char: は\n    romaji: wa\n",
            ),
        ]
    }

    /// A pair with a target-language frequency list: cards matched by word, by
    /// reading, not at all, and one with a manual override.
    fn freq_files() -> Vec<(&'static str, &'static str)> {
        vec![
            (
                "es-xx/lessons/01.yaml",
                concat!(
                    "id: a\ntitle: t\njlpt: N5\ncards:\n",
                    "  - es: Palabra\n    jp: ことば\n    romaji: kotoba\n",
                    "  - es: Yo\n    jp: わたし\n    romaji: watashi\n",
                    "  - es: Por favor\n    jp: おねがいします\n    romaji: onegaishimasu\n",
                    "  - es: Manual\n    jp: はい\n    romaji: hai\n    freq: 7\n",
                ),
            ),
            (
                "es-xx/kana/h.yaml",
                concat!(
                    "type: hiragana\nitems:\n  - char: こ\n    romaji: ko\n  - char: と\n    romaji: to\n",
                    "  - char: ば\n    romaji: ba\n  - char: わ\n    romaji: wa\n  - char: た\n    romaji: ta\n",
                    "  - char: し\n    romaji: shi\n  - char: お\n    romaji: o\n  - char: ね\n    romaji: ne\n",
                    "  - char: が\n    romaji: ga\n  - char: い\n    romaji: i\n  - char: ま\n    romaji: ma\n",
                    "  - char: す\n    romaji: su\n  - char: は\n    romaji: wa\n",
                ),
            ),
            (
                "xx/frequency.tsv",
                "# comment\n1\tことば\tことば\t100\n2\t私\tわたし\t90\n3\tはい\tはい\t80\n",
            ),
        ]
    }

    #[test]
    fn load_valid_pair() {
        let f = Fixture::new(&valid_files());
        let course = f.load("xx").expect("valid content loads");
        assert_eq!(course.lessons.len(), 1);
        assert_eq!(course.lessons[0].cards.len(), 1);
        assert_eq!(course.lessons[0].cards[0].id, "greetings:1");
    }

    /// A card's frequency rank is optional and defaults to unranked.
    #[test]
    fn freq_is_optional() {
        let f = Fixture::new(&valid_files());
        let course = f.load("xx").unwrap();
        assert_eq!(course.lessons[0].cards[0].freq, 0);
    }

    #[test]
    fn kana_category_defaults_to_base() {
        let f = Fixture::new(&valid_files());
        let course = f.load("xx").unwrap();
        assert_eq!(course.kana[0].category, KanaCategory::Base);
    }

    #[test]
    fn kana_invalid_category_is_rejected() {
        let f = Fixture::new(&[
            (
                "xx/lessons/01.yaml",
                "id: a\ntitle: t\njlpt: N5\ncards:\n  - es: Hola\n    jp: こんにちは\n    romaji: konnichiwa\n",
            ),
            (
                "xx/kana/h.yaml",
                "type: hiragana\nitems:\n  - char: が\n    romaji: ga\n    category: bogus\n",
            ),
        ]);
        assert!(f.load("xx").is_err(), "expected invalid category error");
    }

    #[test]
    fn load_rejects_invalid_lessons() {
        let cases = [
            (
                "missing jp",
                "id: a\ntitle: t\njlpt: N5\ncards:\n  - es: Hola\n    romaji: hola\n",
            ),
            (
                "invalid jlpt",
                "id: a\ntitle: t\njlpt: N9\ncards:\n  - es: Hola\n    jp: こんにちは\n    romaji: konnichiwa\n",
            ),
            (
                "missing id",
                "title: t\njlpt: N5\ncards:\n  - es: Hola\n    jp: こんにちは\n    romaji: konnichiwa\n",
            ),
            ("no cards", "id: a\ntitle: t\njlpt: N5\ncards: []\n"),
        ];
        for (name, lesson) in cases {
            let f = Fixture::new(&[
                ("xx/lessons/01.yaml", lesson),
                (
                    "xx/kana/h.yaml",
                    "type: hiragana\nitems:\n  - char: あ\n    romaji: a\n",
                ),
            ]);
            assert!(f.load("xx").is_err(), "expected a validation error: {name}");
        }
    }

    #[test]
    fn load_rejects_duplicate_lesson_id() {
        let lesson =
            "id: dup\ntitle: t\njlpt: N5\ncards:\n  - es: Hola\n    jp: こんにちは\n    romaji: konnichiwa\n";
        let f = Fixture::new(&[
            ("xx/lessons/01.yaml", lesson),
            ("xx/lessons/02.yaml", lesson),
            (
                "xx/kana/h.yaml",
                "type: hiragana\nitems:\n  - char: あ\n    romaji: a\n",
            ),
        ]);
        assert!(f.load("xx").is_err(), "expected duplicate lesson id error");
    }

    #[test]
    fn load_missing_directories_is_an_error() {
        let f = Fixture::new(&[]);
        assert!(
            f.load("xx").is_err(),
            "expected error when no lessons exist"
        );
    }

    /// The curriculum-level validation table: every way content can be
    /// self-inconsistent must fail the load, not ship broken.
    #[test]
    fn load_rejects_invalid_curriculum() {
        let cases: [(&str, Vec<(&str, &str)>); 30] = [
            (
                "unknown function ref",
                vec![
                    ("functions/core.yaml", "functions:\n  - id: greet\n    cefr: A1\n    description: d\n"),
                    ("xx/lessons/01.yaml", "id: a\ntitle: t\njlpt: N5\nfunctions: [nope]\ncards:\n  - es: Hola\n    jp: あ\n    romaji: a\n"),
                    ("xx/kana/h.yaml", VALID_KANA),
                ],
            ),
            (
                "invalid cefr",
                vec![
                    ("functions/core.yaml", "functions:\n  - id: greet\n    cefr: X9\n    description: d\n"),
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                ],
            ),
            (
                "duplicate function id",
                vec![
                    ("functions/core.yaml", "functions:\n  - id: greet\n    cefr: A1\n    description: d\n  - id: greet\n    cefr: A2\n    description: e\n"),
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                ],
            ),
            (
                "function missing description",
                vec![
                    ("functions/core.yaml", "functions:\n  - id: greet\n    cefr: A1\n"),
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                ],
            ),
            (
                "negative freq",
                vec![
                    ("xx/lessons/01.yaml", "id: a\ntitle: t\njlpt: N5\ncards:\n  - es: Hola\n    jp: あ\n    romaji: a\n    freq: -1\n"),
                    ("xx/kana/h.yaml", VALID_KANA),
                ],
            ),
            (
                "card uses unteachable kana",
                vec![
                    ("xx/lessons/01.yaml", "id: a\ntitle: t\njlpt: N5\ncards:\n  - es: Hola\n    jp: そ\n    romaji: so\n"),
                    ("xx/kana/h.yaml", VALID_KANA),
                ],
            ),
            (
                "pattern missing id",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/grammar/01.yaml", "title: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [a:1]\n"),
                ],
            ),
            (
                "pattern missing frame",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/grammar/01.yaml", "id: p\ntitle: t\njlpt: N5\nslots:\n  - name: X\n    cards: [a:1]\n"),
                ],
            ),
            (
                "pattern invalid jlpt",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/grammar/01.yaml", "id: p\ntitle: t\njlpt: N9\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [a:1]\n"),
                ],
            ),
            (
                "duplicate pattern id",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/grammar/01.yaml", "id: p\ntitle: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [a:1]\n"),
                    ("xx/grammar/02.yaml", "id: p\ntitle: t2\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [a:1]\n"),
                ],
            ),
            (
                "pattern slot missing name",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/grammar/01.yaml", "id: p\ntitle: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - cards: [a:1]\n"),
                ],
            ),
            (
                "pattern slot with no candidate cards",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/grammar/01.yaml", "id: p\ntitle: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: []\n"),
                ],
            ),
            (
                "frame placeholder not declared as a slot",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/grammar/01.yaml", "id: p\ntitle: t\njlpt: N5\nframe: \"{Y}\"\nslots:\n  - name: X\n    cards: [a:1]\n"),
                ],
            ),
            (
                "declared slot not used in frame",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/grammar/01.yaml", "id: p\ntitle: t\njlpt: N5\nframe: \"hola\"\nslots:\n  - name: X\n    cards: [a:1]\n"),
                ],
            ),
            (
                "slot default not among its candidate cards",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/grammar/01.yaml", "id: p\ntitle: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [a:1]\n    default: a:2\n"),
                ],
            ),
            (
                "pattern slot references unknown card id",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/grammar/01.yaml", "id: p\ntitle: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [nope:1]\n"),
                ],
            ),
            (
                "chapter missing id",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/story/01.yaml", "title: t\nbeats:\n  - kind: narration\n    es: Hola\n    jp: あ\n"),
                ],
            ),
            (
                "chapter missing title",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/story/01.yaml", "id: c\nbeats:\n  - kind: narration\n    es: Hola\n    jp: あ\n"),
                ],
            ),
            (
                "chapter with no beats",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/story/01.yaml", "id: c\ntitle: t\nbeats: []\n"),
                ],
            ),
            (
                "beat invalid kind",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/story/01.yaml", "id: c\ntitle: t\nbeats:\n  - kind: bogus\n"),
                ],
            ),
            (
                "dialogue beat missing speaker",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/story/01.yaml", "id: c\ntitle: t\nbeats:\n  - kind: dialogue\n    es: Hola\n    jp: あ\n"),
                ],
            ),
            (
                "narration beat missing jp",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/story/01.yaml", "id: c\ntitle: t\nbeats:\n  - kind: narration\n    es: Hola\n"),
                ],
            ),
            (
                "narration beat missing es",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/story/01.yaml", "id: c\ntitle: t\nbeats:\n  - kind: narration\n    jp: あ\n"),
                ],
            ),
            (
                "practice beat missing ref_id",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/story/01.yaml", "id: c\ntitle: t\nbeats:\n  - kind: practice\n    practice: vocab\n"),
                ],
            ),
            (
                "practice beat invalid practice kind",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/story/01.yaml", "id: c\ntitle: t\nbeats:\n  - kind: practice\n    practice: bogus\n    ref_id: a\n"),
                ],
            ),
            (
                "practice beat carries stray dialogue fields",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/story/01.yaml", "id: c\ntitle: t\nbeats:\n  - kind: practice\n    practice: vocab\n    ref_id: a\n    jp: あ\n"),
                ],
            ),
            (
                "duplicate chapter id",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/story/01.yaml", "id: c\ntitle: t\nbeats:\n  - kind: narration\n    es: Hola\n    jp: あ\n"),
                    ("xx/story/02.yaml", "id: c\ntitle: t2\nbeats:\n  - kind: narration\n    es: Hola\n    jp: あ\n"),
                ],
            ),
            (
                "practice beat references unknown lesson id",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/story/01.yaml", "id: c\ntitle: t\nbeats:\n  - kind: practice\n    practice: vocab\n    ref_id: nope\n"),
                ],
            ),
            (
                "present beat with partial framing (jp without es)",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/story/01.yaml", "id: c\ntitle: t\nbeats:\n  - kind: present\n    practice: vocab\n    ref_id: a\n    jp: あ\n"),
                ],
            ),
            (
                "practice before its pool is presented",
                vec![
                    ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                    ("xx/kana/h.yaml", VALID_KANA),
                    ("xx/story/01.yaml", "id: c\ntitle: t\nbeats:\n  - kind: practice\n    practice: vocab\n    ref_id: a\n  - kind: present\n    practice: vocab\n    ref_id: a\n"),
                ],
            ),
        ];

        for (name, files) in cases {
            let f = Fixture::new(&files);
            assert!(f.load("xx").is_err(), "expected a validation error: {name}");
        }
    }

    /// A practice beat naming a kana type with no items must fail, as must one
    /// naming a type that does not exist at all.
    #[test]
    fn load_rejects_unknown_kana_pools() {
        for ref_id in ["bogus", "katakana"] {
            let f = Fixture::new(&[
                ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                ("xx/kana/h.yaml", VALID_KANA),
                (
                    "xx/story/01.yaml",
                    &format!("id: c\ntitle: t\nbeats:\n  - kind: practice\n    practice: kana\n    ref_id: {ref_id}\n"),
                ),
            ]);
            assert!(
                f.load("xx").is_err(),
                "expected an error for kana pool {ref_id:?}"
            );
        }
    }

    /// A present beat must name the pool it introduces.
    #[test]
    fn load_rejects_present_beat_without_pool() {
        for story in [
            "id: c\ntitle: t\nbeats:\n  - kind: present\n    practice: vocab\n",
            "id: c\ntitle: t\nbeats:\n  - kind: present\n    practice: vocab\n    ref_id: nope\n",
        ] {
            let f = Fixture::new(&[
                ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
                ("xx/kana/h.yaml", VALID_KANA),
                ("xx/story/01.yaml", story),
            ]);
            assert!(f.load("xx").is_err(), "expected an error for {story:?}");
        }
    }

    #[test]
    fn load_pattern_valid() {
        let f = Fixture::new(&[
            ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
            ("xx/kana/h.yaml", VALID_KANA),
            (
                "xx/grammar/01.yaml",
                "id: p\ntitle: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [a:1]\n    default: a:1\n",
            ),
        ]);
        let course = f.load("xx").expect("valid pattern loads");
        assert_eq!(course.patterns.len(), 1);
        assert_eq!(course.patterns[0].slots.len(), 1);
        assert_eq!(course.patterns[0].slots[0].default, "a:1");
    }

    /// An omitted slot default falls back to the slot's first candidate card.
    #[test]
    fn pattern_slot_default_defaults_to_first_card() {
        let f = Fixture::new(&[
            ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
            ("xx/kana/h.yaml", VALID_KANA),
            (
                "xx/grammar/01.yaml",
                "id: p\ntitle: t\njlpt: N5\nframe: \"{X}\"\nslots:\n  - name: X\n    cards: [a:1]\n",
            ),
        ]);
        let course = f.load("xx").unwrap();
        assert_eq!(course.patterns[0].slots[0].default, "a:1");
    }

    /// Grammar and story content are optional: a pair without them still loads.
    #[test]
    fn no_grammar_or_story_is_not_an_error() {
        let f = Fixture::new(&valid_files());
        let course = f
            .load("xx")
            .expect("a pair with neither grammar nor story loads");
        assert!(course.patterns.is_empty());
        assert!(course.chapters.is_empty());
    }

    #[test]
    fn load_story_valid() {
        let f = Fixture::new(&[
            ("xx/lessons/01.yaml", VALID_PATTERN_LESSON),
            ("xx/kana/h.yaml", VALID_KANA),
            (
                "xx/story/01.yaml",
                concat!(
                    "id: c\ntitle: t\nbeats:\n",
                    "  - kind: narration\n    es: Hola\n    jp: あ\n",
                    "  - kind: dialogue\n    speaker: Yui\n    es: Hola\n    jp: あ\n",
                    "  - kind: present\n    practice: vocab\n    ref_id: a\n",
                    "  - kind: practice\n    practice: vocab\n    ref_id: a\n",
                ),
            ),
        ]);
        let course = f.load("xx").expect("valid story loads");
        assert_eq!(course.chapters.len(), 1);
        let beats = &course.chapters[0].beats;
        assert_eq!(beats.len(), 4);
        assert_eq!(beats[1].speaker, "Yui");
        assert_eq!(beats[2].kind, crate::model::BeatKind::Present);
        assert_eq!(beats[2].practice, Some(crate::model::PracticeKind::Vocab));
        assert_eq!(beats[2].ref_id, "a");
        assert_eq!(beats[3].practice, Some(crate::model::PracticeKind::Vocab));
        assert_eq!(beats[3].ref_id, "a");
    }

    #[test]
    fn freq_backfill_by_word_reading_and_override() {
        let f = Fixture::new(&freq_files());
        let course = f.load("es-xx").expect("freq fixture loads");
        let cards = &course.lessons[0].cards;
        for (name, card, want) in [
            ("matched by surface word", &cards[0], 1),
            ("matched by reading (kanji surface form)", &cards[1], 2),
            ("unmatched stays unranked", &cards[2], 0),
            ("explicit freq wins over backfill", &cards[3], 7),
        ] {
            assert_eq!(card.freq, want, "{name} ({})", card.jp);
        }
    }

    /// The frequency list is optional: without one, cards simply stay unranked.
    #[test]
    fn freq_backfill_missing_list_is_not_an_error() {
        let files: Vec<_> = freq_files()
            .into_iter()
            .filter(|(p, _)| *p != "xx/frequency.tsv")
            .collect();
        let f = Fixture::new(&files);
        let course = f.load("es-xx").expect("load without a frequency list");
        assert_eq!(course.lessons[0].cards[0].freq, 0);
    }

    /// Every embedded card has a unique non-empty id and a valid level, and both
    /// syllabaries ship.
    #[test]
    fn embedded_cards_and_kana_are_well_formed() {
        let course = load_embedded(DEFAULT_PAIR).unwrap();
        assert!(course.lessons.len() >= 2);
        assert!(!course.kana.is_empty());

        let mut seen = HashSet::new();
        for lesson in &course.lessons {
            for card in &lesson.cards {
                assert!(
                    !card.id.is_empty(),
                    "lesson {:?} has an empty card id",
                    lesson.id
                );
                assert!(
                    seen.insert(card.id.clone()),
                    "duplicate card id {:?}",
                    card.id
                );
                assert!(card.jlpt.is_some(), "card {:?} has no JLPT level", card.id);
            }
        }

        let mut types: HashMap<KanaType, usize> = HashMap::new();
        for k in &course.kana {
            *types.entry(k.kana_type).or_default() += 1;
        }
        assert!(types.get(&KanaType::Hiragana).copied().unwrap_or(0) > 0);
        assert!(types.get(&KanaType::Katakana).copied().unwrap_or(0) > 0);
    }

    /// Long vowels are written with the macron in romaji, and the typeable input
    /// form is kept in the card's notes.
    #[test]
    fn embedded_course_uses_pronunciation_romaji_for_long_vowels() {
        let course = load_embedded(DEFAULT_PAIR).unwrap();
        let want = [
            ("おはよう", "ohayō", "ohayou"),
            ("ありがとう", "arigatō", "arigatou"),
            ("さようなら", "sayōnara", "sayounara"),
            ("きゅう", "kyū", "kyuu"),
            ("じゅう", "jū", "juu"),
        ];

        for (jp, romaji, input) in want {
            let card = course
                .lessons
                .iter()
                .flat_map(|l| &l.cards)
                .find(|c| c.jp == jp)
                .unwrap_or_else(|| panic!("missing embedded card {jp:?}"));
            assert_eq!(card.romaji, romaji, "{jp} romaji");
            assert!(
                card.notes.contains(input),
                "{jp} notes {:?} must carry the input form {input:?}",
                card.notes
            );
        }
    }

    /// Both syllabaries ship the full set: 46 base, 20 dakuten, 5 handakuten and
    /// 33 combination kana.
    #[test]
    fn embedded_kana_categories_are_complete() {
        let course = load_embedded(DEFAULT_PAIR).unwrap();
        let mut counts: HashMap<(KanaType, KanaCategory), usize> = HashMap::new();
        for k in &course.kana {
            *counts.entry((k.kana_type, k.category)).or_default() += 1;
        }

        for typ in [KanaType::Hiragana, KanaType::Katakana] {
            for (cat, want) in [
                (KanaCategory::Base, 46),
                (KanaCategory::Dakuten, 20),
                (KanaCategory::Handakuten, 5),
                (KanaCategory::Combo, 33),
            ] {
                let got = counts.get(&(typ, cat)).copied().unwrap_or(0);
                assert_eq!(got, want, "{typ:?} {cat:?} count");
            }
        }
    }

    /// Every embedded lesson declares at least one communicative function, and
    /// its cards inherit them.
    #[test]
    fn embedded_lessons_reference_known_functions() {
        let course = load_embedded(DEFAULT_PAIR).unwrap();
        for lesson in &course.lessons {
            assert!(
                !lesson.functions.is_empty(),
                "lesson {:?} references no functions",
                lesson.id
            );
            for card in &lesson.cards {
                assert_eq!(
                    card.functions.len(),
                    lesson.functions.len(),
                    "card {:?} must inherit its lesson's functions",
                    card.id
                );
            }
        }
    }

    /// The core N5 vocabulary bank ships, tagged N5 and non-empty.
    #[test]
    fn embedded_n5_vocabulary_lessons() {
        let course = load_embedded(DEFAULT_PAIR).unwrap();
        for id in ["colors", "food-drink", "everyday-objects", "adjectives"] {
            let lesson = course
                .lessons
                .iter()
                .find(|l| l.id == id)
                .unwrap_or_else(|| panic!("missing N5 vocabulary lesson {id:?}"));
            assert_eq!(lesson.jlpt, Some(Jlpt::N5), "lesson {id:?} level");
            assert!(!lesson.cards.is_empty(), "lesson {id:?} has no cards");
        }
    }

    /// The N5 grammar slice: particles, copula, adjective predicates, the polite
    /// verb triad and the te-form request.
    #[test]
    fn embedded_n5_grammar_patterns() {
        let course = load_embedded(DEFAULT_PAIR).unwrap();
        for id in [
            "x-wa-n-desu",
            "kore-sore-are-wa-n-desu",
            "kore-wa-i-adj-desu",
            "x-wa-na-adj-desu",
            "x-ga-suki-desu",
            "x-wo-tabemasu",
            "x-wo-nomimasu",
            "place-ni-ikimasu",
            "place-de-tabemasu",
            "verb-te-kudasai",
        ] {
            let pattern = course
                .patterns
                .iter()
                .find(|p| p.id == id)
                .unwrap_or_else(|| panic!("missing N5 grammar pattern {id:?}"));
            assert_eq!(pattern.jlpt, Some(Jlpt::N5), "pattern {id:?} level");
        }
    }

    /// The N5 story arc, in mastery-gate order.
    #[test]
    fn embedded_n5_story_chapters() {
        let course = load_embedded(DEFAULT_PAIR).unwrap();
        let want = [
            "capitulo-1-asakusa",
            "capitulo-2-nakamise",
            "capitulo-3-sensoji",
            "capitulo-4-rutina",
        ];
        for id in want {
            assert!(
                course.chapters.iter().any(|c| c.id == id),
                "missing N5 story chapter {id:?}"
            );
        }
        assert_eq!(course.chapters.len(), want.len(), "embedded chapter count");
    }

    /// Every embedded pattern carries a valid level and only fills its slots
    /// with vocabulary the learner is actually taught.
    #[test]
    fn embedded_grammar_patterns_are_covered_by_vocabulary() {
        let course = load_embedded(DEFAULT_PAIR).unwrap();
        assert!(!course.patterns.is_empty());
        let card_set = coverage::card_id_set(&course.lessons);
        for p in &course.patterns {
            assert!(p.jlpt.is_some(), "pattern {:?} has no JLPT level", p.id);
            coverage::check_vocab_coverage(p, &card_set)
                .unwrap_or_else(|e| panic!("pattern {:?}: {e}", p.id));
        }
    }

    /// Every embedded chapter references pools that exist and presents material
    /// before practicing it.
    #[test]
    fn embedded_story_chapters_are_covered() {
        let course = load_embedded(DEFAULT_PAIR).unwrap();
        assert!(!course.chapters.is_empty());
        let lesson_ids = coverage::lesson_id_set(&course.lessons);
        let kana_types = coverage::kana_types_present(&course.kana);
        for c in &course.chapters {
            coverage::check_story_coverage(c, &lesson_ids, &kana_types)
                .unwrap_or_else(|e| panic!("chapter {:?}: {e}", c.id));
        }
        coverage::check_story_presentation(&course.chapters)
            .expect("embedded chapters must present before they practice");
    }

    /// Every embedded card is spelled with kana the learner can already decode.
    #[test]
    fn embedded_kana_coverage() {
        let course = load_embedded(DEFAULT_PAIR).unwrap();
        let set = coverage::kana_set(&course.kana);
        for lesson in &course.lessons {
            for card in &lesson.cards {
                coverage::check_kana_coverage(&card.jp, &set)
                    .unwrap_or_else(|e| panic!("card {:?} ({}): {e}", card.id, card.jp));
            }
        }
    }

    const VALID_KANJI: &str =
        "items:\n  - char: 日\n    on: [ニチ]\n    kun: [ひ]\n    meaning: día\n    jlpt: N5\n";

    /// A pair with no kanji directory loads exactly as before — every pair
    /// shipped today is in that state.
    #[test]
    fn kanji_table_is_optional() {
        let f = Fixture::new(&valid_files());
        let course = f.load("xx").expect("a pair without kanji still loads");
        assert!(course.kanji.is_empty());
    }

    #[test]
    fn loads_a_kanji_table() {
        let mut files = valid_files();
        files.push(("xx/kanji/n5.yaml", VALID_KANJI));
        let f = Fixture::new(&files);
        let course = f.load("xx").expect("kanji table loads");
        assert_eq!(course.kanji.len(), 1);
        let k = &course.kanji[0];
        assert_eq!(k.char, "日");
        assert_eq!(k.readings(), vec!["ニチ", "ひ"]);
        assert_eq!(k.meaning, "día");
        assert_eq!(k.jlpt, Some(Jlpt::N5));
    }

    /// A card may only use kanji the course actually teaches — otherwise it
    /// would load fine and then be undecodable forever.
    #[test]
    fn card_using_an_untaught_kanji_is_rejected() {
        let mut files = valid_files();
        files.push((
            "xx/lessons/02.yaml",
            "id: k\ntitle: t\njlpt: N5\nfunctions: [greet-daytime]\ncards:\n  - es: Japón\n    jp: 日本\n    romaji: nihon\n",
        ));
        let f = Fixture::new(&files);
        assert!(
            f.load("xx").is_err(),
            "a card with untaught kanji must fail the load"
        );

        // Teaching both kanji makes the same card valid.
        let mut files = valid_files();
        files.push((
            "xx/lessons/02.yaml",
            "id: k\ntitle: t\njlpt: N5\nfunctions: [greet-daytime]\ncards:\n  - es: Japón\n    jp: 日本\n    romaji: nihon\n",
        ));
        files.push((
            "xx/kanji/n5.yaml",
            "items:\n  - char: 日\n    on: [ニチ]\n    meaning: día\n  - char: 本\n    kun: [もと]\n    meaning: libro\n",
        ));
        let f = Fixture::new(&files);
        f.load("xx").expect("taught kanji make the card valid");
    }

    /// The kanji table validates its own entries.
    #[test]
    fn rejects_invalid_kanji_entries() {
        let cases = [
            (
                "not a kanji",
                "items:\n  - char: あ\n    on: [ア]\n    meaning: a\n",
            ),
            (
                "multi-character",
                "items:\n  - char: 日本\n    on: [ニ]\n    meaning: x\n",
            ),
            ("no readings", "items:\n  - char: 日\n    meaning: día\n"),
            ("no meaning", "items:\n  - char: 日\n    on: [ニチ]\n"),
            (
                "invalid jlpt",
                "items:\n  - char: 日\n    on: [ニチ]\n    meaning: día\n    jlpt: N9\n",
            ),
            ("empty table", "items: []\n"),
        ];
        for (name, body) in cases {
            let mut files = valid_files();
            files.push(("xx/kanji/n5.yaml", body));
            let f = Fixture::new(&files);
            assert!(f.load("xx").is_err(), "expected a validation error: {name}");
        }
    }

    /// The same kanji may not be declared twice across tables.
    #[test]
    fn rejects_duplicate_kanji() {
        let mut files = valid_files();
        files.push(("xx/kanji/a.yaml", VALID_KANJI));
        files.push(("xx/kanji/b.yaml", VALID_KANJI));
        let f = Fixture::new(&files);
        assert!(f.load("xx").is_err(), "duplicate kanji must fail the load");
    }

    /// Every kanji in *evaluable* content must be in the kanji table — the same
    /// "teach it before you test it" rule kana obey. This replaces the old
    /// kanji-free tripwire, which only held because no kanji existed anywhere.
    #[test]
    fn embedded_evaluable_content_only_uses_taught_kanji() {
        let course = load_embedded(DEFAULT_PAIR).unwrap();
        let set = coverage::kanji_set(&course.kanji);
        for lesson in &course.lessons {
            for card in &lesson.cards {
                coverage::check_kanji_coverage(&card.jp, &set)
                    .unwrap_or_else(|e| panic!("card {:?} ({}): {e}", card.id, card.jp));
            }
        }
    }

    /// The shipped kanji table is well-formed: every entry has readings, a
    /// meaning and a level.
    #[test]
    fn embedded_kanji_table_is_well_formed() {
        let course = load_embedded(DEFAULT_PAIR).unwrap();
        assert!(!course.kanji.is_empty(), "the course ships kanji");
        let mut seen = HashSet::new();
        for k in &course.kanji {
            assert!(seen.insert(&k.char), "duplicate kanji {:?}", k.char);
            assert_eq!(k.char.chars().count(), 1, "{:?} is one character", k.char);
            assert!(
                k.char.chars().all(crate::model::is_han),
                "{:?} is a kanji",
                k.char
            );
            assert!(!k.readings().is_empty(), "{:?} has readings", k.char);
            assert!(!k.meaning.trim().is_empty(), "{:?} has a meaning", k.char);
            assert_eq!(k.jlpt, Some(Jlpt::N5), "{:?} is tagged N5", k.char);
            // Readings are kana plus the okurigana parentheses of the
            // convention the loader enforces (e.g. た(べる)).
            for r in k.readings() {
                assert!(
                    r.chars().all(|c| {
                        let u = c as u32;
                        c == '('
                            || c == ')'
                            || (0x3040..=0x309F).contains(&u)
                            || (0x30A0..=0x30FF).contains(&u)
                    }),
                    "reading {r:?} of {:?} must be kana with okurigana parens",
                    k.char
                );
            }
        }
    }

    /// Vocabulary is still written in kana only: the cards will start using
    /// kanji in a later slice, once these are actually taught.
    #[test]
    fn embedded_vocabulary_is_still_kana_only() {
        let course = load_embedded(DEFAULT_PAIR).unwrap();
        for lesson in &course.lessons {
            for card in &lesson.cards {
                assert!(
                    !card.jp.chars().any(crate::model::is_han),
                    "card {:?} uses kanji",
                    card.id
                );
            }
        }
    }

    /// The embedded frequency list is contiguous, descending, deduplicated and
    /// Japanese throughout.
    #[test]
    fn embedded_frequency_list_is_well_formed() {
        let entries = load_embedded_frequency("ja").unwrap();
        assert!(!entries.is_empty());

        let mut seen = HashSet::new();
        let mut prev_count = entries[0].count;
        for (i, e) in entries.iter().enumerate() {
            assert_eq!(
                e.rank as usize,
                i + 1,
                "ranks must be contiguous and ascending"
            );
            assert!(e.count > 0, "entry {:?} has a non-positive count", e.word);
            assert!(
                e.count <= prev_count,
                "entry {:?} count {} exceeds previous {prev_count} (must be non-increasing)",
                e.word,
                e.count
            );
            prev_count = e.count;
            assert!(!e.word.is_empty(), "entry rank {} has a blank word", e.rank);
            assert!(seen.insert(&e.word), "duplicate word {:?}", e.word);
            assert!(
                e.word.chars().any(is_japanese),
                "word {:?} is not Japanese (kana/kanji)",
                e.word
            );
        }
    }

    fn is_japanese(c: char) -> bool {
        let u = c as u32;
        (0x3040..=0x309F).contains(&u) || (0x30A0..=0x30FF).contains(&u) || crate::model::is_han(c)
    }

    #[test]
    fn embedded_course_loads_and_validates() {
        let course = load_embedded(DEFAULT_PAIR).expect("embedded es-ja course must load");
        assert!(!course.lessons.is_empty(), "has lessons");
        assert!(!course.kana.is_empty(), "has kana");
        assert!(!course.patterns.is_empty(), "has grammar patterns");
        assert!(!course.chapters.is_empty(), "has story chapters");

        // Frequency backfill ran: at least one card picked up a rank.
        let ranked = course
            .lessons
            .iter()
            .flat_map(|l| &l.cards)
            .filter(|c| c.freq > 0)
            .count();
        assert!(ranked > 0, "frequency backfill assigned some ranks");
    }

    #[test]
    fn embedded_frequency_loads() {
        let entries = load_embedded_frequency("ja").expect("ja frequency list must load");
        assert!(!entries.is_empty());
        assert_eq!(entries[0].rank, 1);
    }

    #[test]
    fn target_lang_splits_pair() {
        assert_eq!(target_lang("es-ja"), "ja");
        assert_eq!(target_lang("solo"), "");
    }
}
