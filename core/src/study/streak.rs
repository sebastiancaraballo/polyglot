use chrono::{DateTime, Duration, TimeZone, Utc};

use crate::model::Stats;

/// Advances the study streak for a session occurring at `now`. It is a no-op if
/// the profile already studied today; it increments the streak if the last
/// study day was yesterday, and otherwise resets it to 1. `best_streak` and
/// `last_studied_at` are updated accordingly.
pub fn update_streak(mut stats: Stats, now: DateTime<Utc>) -> Stats {
    let today = truncate_day(now);
    match stats.last_studied_at {
        None => stats.streak = 1,
        Some(last) => {
            let last_day = truncate_day(last);
            if last_day == today {
                return stats; // already counted today
            } else if last_day == today - Duration::days(1) {
                stats.streak += 1;
            } else {
                stats.streak = 1;
            }
        }
    }

    if stats.streak > stats.best_streak {
        stats.best_streak = stats.streak;
    }
    stats.last_studied_at = Some(now);
    stats
}

fn truncate_day(t: DateTime<Utc>) -> DateTime<Utc> {
    let day = t.date_naive();
    Utc.from_utc_datetime(&day.and_hms_opt(0, 0, 0).unwrap())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn at(y: i32, m: u32, d: u32) -> DateTime<Utc> {
        Utc.with_ymd_and_hms(y, m, d, 12, 0, 0).unwrap()
    }

    #[test]
    fn first_session_starts_streak() {
        let s = update_streak(Stats::default(), at(2026, 7, 9));
        assert_eq!(s.streak, 1);
        assert_eq!(s.best_streak, 1);
        assert!(s.last_studied_at.is_some());
    }

    #[test]
    fn same_day_is_noop() {
        let s = update_streak(Stats::default(), at(2026, 7, 9));
        let s2 = update_streak(s.clone(), at(2026, 7, 9));
        assert_eq!(s2.streak, s.streak);
    }

    #[test]
    fn consecutive_day_increments_and_gap_resets() {
        let mut s = update_streak(Stats::default(), at(2026, 7, 9));
        s = update_streak(s, at(2026, 7, 10));
        assert_eq!(s.streak, 2);
        assert_eq!(s.best_streak, 2);
        s = update_streak(s, at(2026, 7, 13)); // gap
        assert_eq!(s.streak, 1);
        assert_eq!(s.best_streak, 2); // best is preserved
    }
}
