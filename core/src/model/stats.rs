use chrono::{DateTime, Utc};

/// Per-profile aggregate progress shown on the menu and stats screens.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct Stats {
    pub streak: i64,
    pub best_streak: i64,
    /// `None` means the profile has never studied.
    pub last_studied_at: Option<DateTime<Utc>>,
    /// Cumulative experience points earned across all activity.
    pub xp: i64,
}
