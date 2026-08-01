/// Tracks a learner's progress toward *automaticity* on a single kana: fast,
/// accurate recognition rather than effortful decoding. Mastery is earned by a
/// run of correct, fast answers; a wrong or slow answer breaks the run. Once
/// reached, `mastered` stays set — long-term retention is the spaced-repetition
/// system's job, not the decoding gate's.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct KanaProgress {
    pub char: String,
    /// Current run of correct, fast answers.
    pub streak: i64,
    /// Total answers seen.
    pub attempts: i64,
    /// Reached the automaticity threshold at least once.
    pub mastered: bool,
    /// Fastest correct answer, in milliseconds (`0` = none yet).
    pub best_ms: i64,
}

/// Tracks a learner's progress toward reading one kanji. Mirrors
/// [`KanaProgress`] — a correctness streak driving a sticky `mastered` flag —
/// because the decoding gate asks the same question of both: can the learner
/// read this character yet? Readings are not tracked individually; a kanji is
/// mastered as a unit.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct KanjiProgress {
    pub char: String,
    /// Current run of correct answers.
    pub streak: i64,
    /// Total answers seen.
    pub attempts: i64,
    /// Reached the mastery threshold at least once; never revoked.
    pub mastered: bool,
}

/// Tracks a learner's progress drilling one slot of one grammar pattern to a
/// correctness-based mastery streak — the same automaticity idiom used for
/// [`KanaProgress`]. Slots are tracked independently so words-before-sentences
/// sequencing can tell which slot still needs practice.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct PatternProgress {
    pub pattern_id: String,
    pub slot: String,
    pub streak: i64,
    pub attempts: i64,
    pub mastered: bool,
}

/// Tracks a learner's position within one Katsudoo chapter: many rows per
/// profile (one per chapter), the same idiom as [`KanaProgress`] and
/// [`PatternProgress`], so progress on multiple chapters can coexist as the
/// story content grows.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct StoryProgress {
    pub chapter_id: String,
    /// Index of the next beat to show; `0` = not yet started.
    pub beat_index: i64,
    /// Positional: every beat has been seen.
    pub completed: bool,
    /// The end-of-chapter challenge was passed; never revoked.
    pub mastered: bool,
}
