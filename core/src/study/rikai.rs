use std::collections::{HashMap, HashSet};

use super::automaticity::MASTERY_STREAK;
use crate::model::{CardState, Pattern, PatternProgress, Slot};

/// The "words before sentences" gating signal: a vocab card counts as known
/// once it has survived at least one spaced-repetition review. This is
/// deliberately the simplest viable signal.
pub fn card_known(state: &CardState) -> bool {
    state.reps > 0
}

/// Reports whether every slot of the pattern has at least one filler the
/// learner already knows, so the pattern can be drilled without ever
/// introducing new vocabulary through the grammar drill itself.
pub fn pattern_ready(p: &Pattern, known: &HashSet<String>) -> bool {
    p.slots.iter().all(|slot| slot_ready(slot, known))
}

fn slot_ready(slot: &Slot, known: &HashSet<String>) -> bool {
    slot.card_ids.iter().any(|id| known.contains(id))
}

/// Returns which slot index varies on drill round `round` (0-based), cycling
/// through the pattern's slots one at a time. Cognitive Load Theory: change
/// only one variable per round.
pub fn variable_slot_index(slot_count: usize, round: usize) -> usize {
    if slot_count == 0 {
        0
    } else {
        round % slot_count
    }
}

/// Substitutes each `"{name}"` placeholder in `frame` with `fill[name]`. An
/// unmatched or unknown placeholder is emitted literally.
pub fn render_frame(frame: &str, fill: &HashMap<String, String>) -> String {
    let mut out = String::with_capacity(frame.len());
    let mut rest = frame;
    while let Some(open) = rest.find('{') {
        out.push_str(&rest[..open]);
        let after = &rest[open..];
        if let Some(close) = after.find('}') {
            let name = &after[1..close];
            if let Some(v) = fill.get(name) {
                out.push_str(v);
                rest = &after[close + 1..];
                continue;
            }
        }
        out.push('{');
        rest = &after[1..];
    }
    out.push_str(rest);
    out
}

/// Folds one substitution-drill answer into a pattern slot's progress. Mastery
/// is correctness-only (a streak of [`MASTERY_STREAK`] in a row), matching the
/// precedent set for kana.
pub fn grade_pattern_slot(mut p: PatternProgress, correct: bool) -> PatternProgress {
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

    fn fill(pairs: &[(&str, &str)]) -> HashMap<String, String> {
        pairs
            .iter()
            .map(|(k, v)| (k.to_string(), v.to_string()))
            .collect()
    }

    #[test]
    fn render_substitutes_known_and_keeps_unknown() {
        let f = fill(&[("X", "わたし"), ("N", "がくせい")]);
        assert_eq!(render_frame("{X}は{N}です", &f), "わたしはがくせいです");
        assert_eq!(render_frame("{Z}です", &f), "{Z}です"); // unknown kept literal
    }

    #[test]
    fn variable_slot_cycles() {
        assert_eq!(variable_slot_index(0, 5), 0);
        assert_eq!(variable_slot_index(2, 0), 0);
        assert_eq!(variable_slot_index(2, 1), 1);
        assert_eq!(variable_slot_index(2, 2), 0);
    }

    #[test]
    fn pattern_ready_needs_a_known_filler_per_slot() {
        let p = Pattern {
            id: "p".to_string(),
            title: String::new(),
            jlpt: None,
            frame: "{X}は{N}です".to_string(),
            slots: vec![
                Slot {
                    name: "X".to_string(),
                    card_ids: vec!["a".to_string(), "b".to_string()],
                    default: "a".to_string(),
                },
                Slot {
                    name: "N".to_string(),
                    card_ids: vec!["c".to_string()],
                    default: "c".to_string(),
                },
            ],
            notes: String::new(),
        };
        let known: HashSet<String> = ["a"].iter().map(|s| s.to_string()).collect();
        assert!(!pattern_ready(&p, &known)); // slot N has no known filler
        let known: HashSet<String> = ["b", "c"].iter().map(|s| s.to_string()).collect();
        assert!(pattern_ready(&p, &known));
    }

    #[test]
    fn grade_slot_masters_after_streak() {
        let mut p = PatternProgress::default();
        for _ in 0..MASTERY_STREAK {
            p = grade_pattern_slot(p, true);
        }
        assert!(p.mastered);
        p = grade_pattern_slot(p, false);
        assert_eq!(p.streak, 0);
        assert!(p.mastered); // never revoked
    }

    /// A slot masters exactly on the streak threshold, not before, and every
    /// answer counts as an attempt.
    #[test]
    fn grade_slot_mastery_requires_an_accurate_run() {
        let mut p = PatternProgress::default();
        for i in 1..=MASTERY_STREAK {
            p = grade_pattern_slot(p, true);
            assert_eq!(p.mastered, i >= MASTERY_STREAK, "after {i} correct answers");
        }
        assert_eq!(p.streak, MASTERY_STREAK);
        assert_eq!(p.attempts, MASTERY_STREAK);
    }

    /// A card counts as known once it has been reviewed at least once.
    #[test]
    fn card_known_after_one_review() {
        for (name, reps, want) in [
            ("never reviewed", 0, false),
            ("reviewed once", 1, true),
            ("reviewed many times", 5, true),
        ] {
            let state = CardState {
                reps,
                ..CardState::new("c")
            };
            assert_eq!(card_known(&state), want, "{name}");
        }
    }
}
