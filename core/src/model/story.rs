/// Distinguishes what kind of moment a [`Beat`] is. `Narration` sets a scene or
/// advances the plot with no specific speaker; `Dialogue` is a line spoken by a
/// character; `Present` diegetically introduces a pool of material (a vocabulary
/// lesson or a kana set) so the learner meets it before being asked to retrieve
/// it; `Practice` pauses the story for one diegetic check that reuses an
/// existing trainer's grading logic.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum BeatKind {
    Narration,
    Dialogue,
    Present,
    Practice,
}

impl BeatKind {
    /// The wire string for this beat kind.
    pub fn as_str(&self) -> &'static str {
        match self {
            BeatKind::Narration => "narration",
            BeatKind::Dialogue => "dialogue",
            BeatKind::Present => "present",
            BeatKind::Practice => "practice",
        }
    }

    /// Parses a beat kind from its wire string, or `None` if unrecognized.
    #[allow(clippy::should_implement_trait)]
    pub fn from_str(s: &str) -> Option<BeatKind> {
        match s {
            "narration" => Some(BeatKind::Narration),
            "dialogue" => Some(BeatKind::Dialogue),
            "present" => Some(BeatKind::Present),
            "practice" => Some(BeatKind::Practice),
            _ => None,
        }
    }
}

/// Identifies which existing trainer's pool a practice beat draws its question
/// from.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum PracticeKind {
    Vocab,
    Kana,
}

impl PracticeKind {
    /// The wire string for this practice kind.
    pub fn as_str(&self) -> &'static str {
        match self {
            PracticeKind::Vocab => "vocab",
            PracticeKind::Kana => "kana",
        }
    }

    /// Parses a practice kind from its wire string, or `None` if unrecognized.
    #[allow(clippy::should_implement_trait)]
    pub fn from_str(s: &str) -> Option<PracticeKind> {
        match s {
            "vocab" => Some(PracticeKind::Vocab),
            "kana" => Some(PracticeKind::Kana),
            _ => None,
        }
    }
}

/// A single moment in a chapter. Only the fields relevant to `kind` are
/// populated; the rest stay empty/`None`, the same "optional, zero means unset"
/// idiom used by `Card::freq` and `Pattern::notes`.
#[derive(Clone, Debug, PartialEq)]
pub struct Beat {
    pub kind: BeatKind,

    /// Dialogue only: the character's name.
    pub speaker: String,
    /// Optional: a real-world place this beat evokes.
    pub place: String,

    /// Narration/dialogue/present: the line in the learner's source language
    /// (Spanish).
    pub source: String,
    /// Narration/dialogue/present: the line in Japanese.
    pub jp: String,
    /// Narration/dialogue/present: optional romanized reading.
    pub romaji: String,

    /// Present/practice only.
    pub practice: Option<PracticeKind>,
    /// Resolves the pool a present or practice beat draws on: a `Lesson::id`
    /// when `practice` is `Vocab`, or `"hiragana"`/`"katakana"` when `practice`
    /// is `Kana`.
    pub ref_id: String,
}

/// An ordered sequence of beats: Katsudoo's communicative-activity unit.
#[derive(Clone, Debug, PartialEq)]
pub struct Chapter {
    pub id: String,
    pub title: String,
    pub beats: Vec<Beat>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn beat_kind_from_str_validity() {
        for (s, want) in [
            ("narration", true),
            ("dialogue", true),
            ("practice", true),
            ("present", true),
            ("", false),
            ("bogus", false),
        ] {
            assert_eq!(BeatKind::from_str(s).is_some(), want, "{s:?}");
        }
    }

    #[test]
    fn practice_kind_from_str_validity() {
        for (s, want) in [
            ("vocab", true),
            ("kana", true),
            ("", false),
            ("bogus", false),
        ] {
            assert_eq!(PracticeKind::from_str(s).is_some(), want, "{s:?}");
        }
    }
}
