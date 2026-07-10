use std::time::Duration;

use crate::model::KanaProgress;

/// The run of correct answers in a row that marks a kana mastered. Mastery
/// depends on accuracy only: answering correctly several times running shows
/// the reading is learned. Response time is recorded as a stat (`best_ms`) but
/// does not gate the streak.
pub const MASTERY_STREAK: i64 = 3;

/// Folds one answer into a kana's progress and returns the updated value.
/// `elapsed` is the time the learner took to answer; it is recorded as the
/// kana's best time but does not affect the mastery streak. Pure function.
pub fn grade_kana(mut p: KanaProgress, correct: bool, elapsed: Duration) -> KanaProgress {
    p.attempts += 1;

    if correct {
        let ms = elapsed.as_millis() as i64;
        if ms > 0 && (p.best_ms == 0 || ms < p.best_ms) {
            p.best_ms = ms;
        }
        p.streak += 1;
        if p.streak >= MASTERY_STREAK {
            p.mastered = true;
        }
    } else {
        p.streak = 0;
    }
    p
}

#[cfg(test)]
mod tests {
    use super::*;

    fn ms(n: u64) -> Duration {
        Duration::from_millis(n)
    }

    #[test]
    fn masters_after_streak_and_tracks_best_time() {
        let mut p = KanaProgress {
            char: "あ".to_string(),
            ..Default::default()
        };
        p = grade_kana(p, true, ms(900));
        p = grade_kana(p, true, ms(500));
        assert!(!p.mastered);
        p = grade_kana(p, true, ms(700));
        assert!(p.mastered);
        assert_eq!(p.streak, 3);
        assert_eq!(p.attempts, 3);
        assert_eq!(p.best_ms, 500);
    }

    #[test]
    fn wrong_answer_resets_streak_not_mastery() {
        let mut p = KanaProgress::default();
        p = grade_kana(p, true, ms(400));
        p = grade_kana(p, false, ms(400));
        assert_eq!(p.streak, 0);
        assert_eq!(p.attempts, 2);
    }
}
