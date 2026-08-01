//! Shared study-mode logic used by multiple screens: multiple-choice option
//! generation, study-streak bookkeeping, kana automaticity grading, the
//! Foundations decoding gate (both the syllabary-ordering gate and the
//! progressive, decodable-texts reading filter), Rikai grammar drills, the
//! end-of-chapter challenge, and the level mock assessment.
//!
//! Port of the Go `internal/study` package.

mod assessment;
mod automaticity;
mod challenge;
mod choice;
mod decodable;
mod gate;
mod rikai;
mod streak;
mod xp;

pub use assessment::{build_assessment, AssessKind, AssessQuestion, ASSESSMENT_LENGTH};
pub use automaticity::{grade_kana, grade_kanji, MASTERY_STREAK};
pub use challenge::{
    build_challenge, challenge_needed, challenge_passed, ChallengeQuestion, CHALLENGE_LENGTH,
};
pub use choice::options;
pub use decodable::Decoder;
pub use gate::{new_gate, Fluency, Gate};
pub use rikai::{card_known, grade_pattern_slot, pattern_ready, render_frame, variable_slot_index};
pub use streak::update_streak;
pub use xp::{xp_for_answer, xp_for_grade, ONBOARDING_XP};
