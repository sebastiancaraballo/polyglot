use std::time::Duration;

use crate::model::{KanaProgress, KanjiProgress};

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

/// Folds one answer into a kanji's progress and returns the updated value.
/// Same accuracy-only rule as [`grade_kana`]: a run of correct answers marks it
/// mastered, and mastery is never revoked by a later lapse. Kanji answers are
/// not timed — a reading recalled slowly is still recalled — so there is no
/// `elapsed` and no best time. Pure function.
pub fn grade_kanji(mut p: KanjiProgress, correct: bool) -> KanjiProgress {
    p.attempts += 1;
    if correct {
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

    /// Mastery lands exactly on the streak threshold, not before.
    #[test]
    fn mastery_requires_an_accurate_run() {
        let mut p = KanaProgress::default();
        for i in 1..=MASTERY_STREAK {
            p = grade_kana(p, true, ms(1000));
            assert_eq!(p.mastered, i >= MASTERY_STREAK, "after {i} correct answers");
        }
        assert_eq!(p.streak, MASTERY_STREAK);
        assert_eq!(p.attempts, MASTERY_STREAK);
    }

    /// Response time does not gate the streak: a slow but correct answer counts,
    /// and still records a best time.
    #[test]
    fn slow_answer_still_advances_streak() {
        let p = grade_kana(KanaProgress::default(), true, Duration::from_secs(30));
        assert_eq!(p.streak, 1);
        assert!(p.best_ms > 0, "a correct answer records a best time");
    }

    /// An untimed answer counts toward mastery but records no best time.
    #[test]
    fn untimed_correct_answer_advances_streak() {
        let p = grade_kana(KanaProgress::default(), true, Duration::ZERO);
        assert_eq!(p.streak, 1);
        assert_eq!(p.best_ms, 0);
    }

    /// Kanji grade on accuracy alone and their mastery is sticky, like kana —
    /// but they carry no timing, since a reading recalled slowly is recalled.
    #[test]
    fn kanji_master_on_accuracy_and_stay_mastered() {
        let mut p = KanjiProgress::default();
        for i in 1..=MASTERY_STREAK {
            p = grade_kanji(p, true);
            assert_eq!(p.mastered, i >= MASTERY_STREAK, "after {i} correct answers");
        }
        assert_eq!(p.streak, MASTERY_STREAK);
        assert_eq!(p.attempts, MASTERY_STREAK);

        p = grade_kanji(p, false);
        assert_eq!(p.streak, 0, "a wrong answer resets the streak");
        assert!(p.mastered, "mastery survives a later lapse");
    }

    /// Mastery is sticky: a later lapse zeroes the streak but never revokes it.
    #[test]
    fn mastery_survives_a_later_lapse() {
        let mut p = KanaProgress::default();
        for _ in 0..MASTERY_STREAK {
            p = grade_kana(p, true, ms(1000));
        }
        p = grade_kana(p, false, ms(1000));
        assert_eq!(p.streak, 0);
        assert!(p.mastered, "mastery should remain sticky after a lapse");
    }
}
