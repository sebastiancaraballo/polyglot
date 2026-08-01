use chrono::{DateTime, Utc};

use super::Jlpt;

/// Records a profile's outcome on a level's mock assessment: one row per
/// (profile, level). `passed` is sticky — once a level is passed it is never
/// revoked (Mastery Learning), the same idiom as `StoryProgress::mastered`.
/// `best_correct`/`total` keep the learner's best score for the level.
#[derive(Clone, Debug, PartialEq)]
pub struct AssessmentResult {
    pub level: Jlpt,
    pub passed: bool,
    pub best_correct: i64,
    pub total: i64,
    /// When the assessment was last taken; `None` if never.
    pub taken_at: Option<DateTime<Utc>>,
}
