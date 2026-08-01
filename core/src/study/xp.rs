use crate::srs::Grade;

/// The one-time bonus granted for completing onboarding.
pub const ONBOARDING_XP: i64 = 20;

/// Returns the experience points earned for an answer graded by the
/// spaced-repetition scheduler. Every answer earns something so all interaction
/// is rewarded, but more accurate recall earns more.
pub fn xp_for_grade(grade: Grade) -> i64 {
    match grade {
        Grade::Again => 2,
        Grade::Hard => 6,
        Grade::Good => 10,
        Grade::Easy => 14,
    }
}

/// The correct/incorrect shorthand used by screens that only distinguish right
/// from wrong (quiz, kana trainer).
pub fn xp_for_answer(correct: bool) -> i64 {
    if correct {
        xp_for_grade(Grade::Good)
    } else {
        xp_for_grade(Grade::Again)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn grades_are_ordered_and_answer_maps_to_good_again() {
        assert!(
            xp_for_grade(Grade::Again) < xp_for_grade(Grade::Hard)
                && xp_for_grade(Grade::Hard) < xp_for_grade(Grade::Good)
                && xp_for_grade(Grade::Good) < xp_for_grade(Grade::Easy)
        );
        assert_eq!(xp_for_answer(true), xp_for_grade(Grade::Good));
        assert_eq!(xp_for_answer(false), xp_for_grade(Grade::Again));
    }

    /// The award per grade is pinned: changing it changes every learner's
    /// progression, so it should never drift silently.
    #[test]
    fn xp_per_grade_is_pinned() {
        for (grade, want) in [
            (Grade::Again, 2),
            (Grade::Hard, 6),
            (Grade::Good, 10),
            (Grade::Easy, 14),
        ] {
            assert_eq!(xp_for_grade(grade), want, "{grade:?}");
        }
    }

    /// A correct answer always outearns an incorrect one.
    #[test]
    fn correct_answers_outearn_incorrect() {
        assert!(xp_for_answer(false) < xp_for_answer(true));
    }
}
