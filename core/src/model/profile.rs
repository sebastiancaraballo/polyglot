use chrono::{DateTime, Utc};

/// A local learner. Multiple profiles can exist on the same machine, each with
/// its own progress and statistics.
#[derive(Clone, Debug, PartialEq)]
pub struct Profile {
    pub id: i64,
    pub name: String,
    pub onboarded: bool,
    /// Controls whether romaji is displayed alongside Japanese in the study
    /// screens. New profiles default to `true`.
    pub show_romaji: bool,
    /// Records whether the learner has seen the kana trainer's first-time
    /// intro. New profiles default to `false`.
    pub kana_onboarded: bool,
    pub created_at: Option<DateTime<Utc>>,
}
