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
mod lessons;
mod story;

use std::fmt;
use std::io;

pub use fsys::{ContentFs, DirFs, EmbeddedFs};

use crate::model::{Chapter, FreqEntry, KanaItem, Lesson, Pattern};

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
    let patterns = grammar::load_patterns(fsys, pair)?;

    // Every kana a card depends on must be teachable (present in the tables).
    let set = coverage::kana_set(&kana);
    for lesson in &lessons {
        for card in &lesson.cards {
            coverage::check_kana_coverage(&card.jp, &set)
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
