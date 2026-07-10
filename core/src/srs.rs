//! Spaced-repetition scheduler (SM-2 style).
//!
//! Port of the Go `internal/srs` package. [`review`] is a pure function: the
//! result depends only on its inputs.

use chrono::{DateTime, Duration, Utc};

use crate::model::{CardState, DEFAULT_EASE};

/// The learner's self-assessment of a review, from `Again` (forgot) to `Easy`.
///
/// The discriminants match the Go original (1..=4) so they persist and cross
/// FFI boundaries identically.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(i64)]
pub enum Grade {
    /// Forgot the card.
    Again = 1,
    /// Recalled with difficulty.
    Hard = 2,
    /// Recalled correctly.
    Good = 3,
    /// Recalled effortlessly.
    Easy = 4,
}

impl Grade {
    /// Returns the grade for a persisted/wire discriminant, or `None` if out of
    /// range (mirrors the Go `Grade.Valid` check on an `int`).
    pub fn from_i64(v: i64) -> Option<Grade> {
        match v {
            1 => Some(Grade::Again),
            2 => Some(Grade::Hard),
            3 => Some(Grade::Good),
            4 => Some(Grade::Easy),
            _ => None,
        }
    }

    /// The persisted/wire discriminant for this grade.
    pub fn as_i64(self) -> i64 {
        self as i64
    }
}

const MIN_EASE: f64 = 1.3;
const EASY_BONUS: f64 = 1.3;

/// Returns the initial scheduling state for a card that has never been
/// reviewed. Its `None` `due_at` makes it immediately due.
pub fn new_card(card_id: impl Into<String>) -> CardState {
    CardState::new(card_id)
}

/// Reports whether the card is due for review at `now`.
pub fn is_due(state: &CardState, now: DateTime<Utc>) -> bool {
    match state.due_at {
        None => true,
        Some(due) => due <= now,
    }
}

/// Applies a grade to a card's state and returns the updated state, including
/// the next interval (in days) and due date.
pub fn review(state: &CardState, grade: Grade, now: DateTime<Utc>) -> CardState {
    let mut s = state.clone();
    if s.ease == 0.0 {
        s.ease = DEFAULT_EASE;
    }
    s.last_reviewed_at = Some(now);

    if grade == Grade::Again {
        s.reps = 0;
        s.lapses += 1;
        s.ease = clamp_ease(s.ease - 0.20);
        s.interval = 0;
        s.due_at = Some(now); // review again in the same session
        return s;
    }

    match grade {
        Grade::Hard => s.ease = clamp_ease(s.ease - 0.15),
        Grade::Easy => s.ease = clamp_ease(s.ease + 0.15),
        _ => {}
    }

    if s.reps == 0 {
        s.interval = first_interval(grade);
    } else {
        s.interval = grown_interval(s.interval, s.ease, grade);
    }

    s.reps += 1;
    s.due_at = Some(now + Duration::days(s.interval));
    s
}

/// Returns the interval (in days) that [`review`] would assign for the given
/// grade, without mutating state.
pub fn preview_interval(state: &CardState, grade: Grade, now: DateTime<Utc>) -> i64 {
    review(state, grade, now).interval
}

fn first_interval(grade: Grade) -> i64 {
    match grade {
        Grade::Easy => 4,
        _ => 1, // Hard, Good
    }
}

fn grown_interval(prev: i64, ease: f64, grade: Grade) -> i64 {
    let prev = prev.max(1);
    let factor = match grade {
        Grade::Hard => 1.2,
        Grade::Easy => ease * EASY_BONUS,
        _ => ease, // Good
    };
    let next = (prev as f64 * factor).round() as i64;
    if next <= prev {
        prev + 1
    } else {
        next
    }
}

fn clamp_ease(ease: f64) -> f64 {
    if ease < MIN_EASE {
        MIN_EASE
    } else {
        ease
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::model::DEFAULT_EASE;
    use chrono::TimeZone;

    fn now() -> DateTime<Utc> {
        Utc.with_ymd_and_hms(2026, 6, 19, 12, 0, 0).unwrap()
    }

    #[test]
    fn new_card_is_due() {
        let card = new_card("c1");
        assert_eq!(card.ease, DEFAULT_EASE);
        assert!(is_due(&card, now()), "a new card should be due immediately");
    }

    #[test]
    fn grade_round_trips() {
        for g in [Grade::Again, Grade::Hard, Grade::Good, Grade::Easy] {
            assert_eq!(Grade::from_i64(g.as_i64()), Some(g));
        }
        assert_eq!(Grade::from_i64(0), None);
        assert_eq!(Grade::from_i64(5), None);
    }

    #[test]
    fn review_first_success() {
        for (grade, want_interval) in [(Grade::Hard, 1), (Grade::Good, 1), (Grade::Easy, 4)] {
            let got = review(&new_card("c1"), grade, now());
            assert_eq!(got.interval, want_interval, "grade {grade:?}");
            assert_eq!(got.reps, 1, "grade {grade:?}");
            assert_eq!(got.due_at, Some(now() + Duration::days(want_interval)));
        }
    }

    #[test]
    fn review_again_resets() {
        let card = review(
            &review(&new_card("c1"), Grade::Good, now()),
            Grade::Good,
            now(),
        );
        assert!(card.reps > 0);

        let got = review(&card, Grade::Again, now());
        assert_eq!(got.reps, 0);
        assert_eq!(got.lapses, 1);
        assert_eq!(got.interval, 0);
        assert!(is_due(&got, now()));
        assert!(got.ease < card.ease, "ease should decrease after Again");
    }

    #[test]
    fn intervals_grow() {
        let mut card = new_card("c1");
        let mut prev = 0;
        for i in 0..5 {
            card = review(&card, Grade::Good, now());
            assert!(card.interval > prev, "interval did not grow at step {i}");
            prev = card.interval;
        }
    }

    #[test]
    fn easy_grows_faster_than_good() {
        let base = review(
            &review(&new_card("c1"), Grade::Good, now()),
            Grade::Good,
            now(),
        );
        let good = review(&base, Grade::Good, now());
        let easy = review(&base, Grade::Easy, now());
        assert!(easy.interval > good.interval);
    }

    #[test]
    fn ease_never_below_minimum() {
        let mut card = new_card("c1");
        for _ in 0..20 {
            card = review(&card, Grade::Again, now());
        }
        assert!(card.ease >= MIN_EASE);
    }

    #[test]
    fn preview_matches_review() {
        let card = review(&new_card("c1"), Grade::Good, now());
        for g in [Grade::Again, Grade::Hard, Grade::Good, Grade::Easy] {
            assert_eq!(
                preview_interval(&card, g, now()),
                review(&card, g, now()).interval
            );
        }
    }
}
