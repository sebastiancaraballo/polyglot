use std::collections::HashSet;

use rand::seq::SliceRandom;
use rand::Rng;

use crate::model::{BeatKind, Card, Chapter, KanaItem, Lesson, PracticeKind};

/// How many retrieval questions an end-of-chapter challenge asks, when the
/// chapter's pools are large enough to supply them.
pub const CHALLENGE_LENGTH: usize = 5;

/// One retrieval-practice question drawn from a chapter's practice-beat pools.
/// It mirrors a practice beat's shape so the story runner reuses the exact same
/// question-building and grading paths.
#[derive(Clone, Debug, PartialEq)]
pub struct ChallengeQuestion {
    pub practice: PracticeKind,
    /// The pool it was drawn from: a `Lesson::id` or kana type.
    pub ref_id: String,
    /// Set when `practice == Vocab`.
    pub card: Option<Card>,
    /// Set when `practice == Kana`.
    pub kana: Option<KanaItem>,
}

/// Draws up to [`CHALLENGE_LENGTH`] questions for the chapter, sampled without
/// replacement, round-robin across the chapter's distinct practice pools (in
/// beat order) so every referenced pool contributes. Returns an empty vec when
/// the chapter has no practice beats.
pub fn build_challenge<R: Rng + ?Sized>(
    rng: &mut R,
    chapter: &Chapter,
    lessons: &[Lesson],
    kana: &[KanaItem],
) -> Vec<ChallengeQuestion> {
    let mut pools = challenge_pools(chapter, lessons, kana);
    if pools.is_empty() {
        return Vec::new();
    }
    for p in pools.iter_mut() {
        p.shuffle(rng);
    }

    let mut out = Vec::new();
    while out.len() < CHALLENGE_LENGTH {
        let mut progressed = false;
        for p in pools.iter_mut() {
            if out.len() >= CHALLENGE_LENGTH {
                break;
            }
            if p.is_empty() {
                continue;
            }
            out.push(p.remove(0));
            progressed = true;
        }
        if !progressed {
            break;
        }
    }
    out
}

/// Builds one pool per distinct practice reference in beat order, deduplicating
/// individual items (by card ID / kana char) so the same question can never be
/// drawn twice even when pools overlap.
fn challenge_pools(
    chapter: &Chapter,
    lessons: &[Lesson],
    kana: &[KanaItem],
) -> Vec<Vec<ChallengeQuestion>> {
    let mut seen_ref: HashSet<&str> = HashSet::new();
    let mut seen_item: HashSet<String> = HashSet::new();
    let mut pools: Vec<Vec<ChallengeQuestion>> = Vec::new();

    for beat in &chapter.beats {
        if beat.kind != BeatKind::Practice || seen_ref.contains(beat.ref_id.as_str()) {
            continue;
        }
        seen_ref.insert(beat.ref_id.as_str());

        let mut questions: Vec<ChallengeQuestion> = Vec::new();
        match beat.practice {
            Some(PracticeKind::Vocab) => {
                for lesson in lessons {
                    if lesson.id != beat.ref_id {
                        continue;
                    }
                    for c in &lesson.cards {
                        let key = format!("card:{}", c.id);
                        if !seen_item.insert(key) {
                            continue;
                        }
                        questions.push(ChallengeQuestion {
                            practice: PracticeKind::Vocab,
                            ref_id: beat.ref_id.clone(),
                            card: Some(c.clone()),
                            kana: None,
                        });
                    }
                }
            }
            Some(PracticeKind::Kana) => {
                for k in kana {
                    if k.kana_type.as_str() != beat.ref_id {
                        continue;
                    }
                    let key = format!("kana:{}", k.char);
                    if !seen_item.insert(key) {
                        continue;
                    }
                    questions.push(ChallengeQuestion {
                        practice: PracticeKind::Kana,
                        ref_id: beat.ref_id.clone(),
                        card: None,
                        kana: Some(k.clone()),
                    });
                }
            }
            None => {}
        }
        if !questions.is_empty() {
            pools.push(questions);
        }
    }
    pools
}

/// Applies the mastery criterion: at least 80% correct — Bloom's mastery band,
/// expressed as a ratio so short challenges from small pools stay principled
/// (below 5 questions it effectively requires them all).
pub fn challenge_passed(correct: i64, total: i64) -> bool {
    total > 0 && correct * 5 >= total * 4
}

/// Returns the minimum correct answers that pass a challenge of `total`
/// questions: the smallest `n` with [`challenge_passed`].
pub fn challenge_needed(total: i64) -> i64 {
    (total * 4 + 4) / 5
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::model::{Beat, BeatKind};
    use rand::rngs::StdRng;
    use rand::SeedableRng;

    fn practice_beat(ref_id: &str) -> Beat {
        Beat {
            kind: BeatKind::Practice,
            speaker: String::new(),
            place: String::new(),
            source: String::new(),
            jp: String::new(),
            romaji: String::new(),
            practice: Some(PracticeKind::Vocab),
            ref_id: ref_id.to_string(),
        }
    }

    fn card(id: &str) -> Card {
        Card {
            id: id.to_string(),
            source: id.to_string(),
            jp: id.to_string(),
            romaji: String::new(),
            notes: String::new(),
            jlpt: None,
            functions: Vec::new(),
            freq: 0,
        }
    }

    #[test]
    fn passing_criterion() {
        assert!(challenge_passed(4, 5));
        assert!(!challenge_passed(3, 5));
        assert!(challenge_passed(5, 5));
        assert!(!challenge_passed(0, 0));
        assert_eq!(challenge_needed(5), 4);
        assert_eq!(challenge_needed(3), 3); // small pools effectively require all
    }

    #[test]
    fn builds_capped_deduped_questions() {
        let lesson = Lesson {
            id: "greetings".to_string(),
            title: String::new(),
            jlpt: None,
            functions: Vec::new(),
            cards: (0..8).map(|i| card(&format!("greetings:{i}"))).collect(),
        };
        let chapter = Chapter {
            id: "ch1".to_string(),
            title: String::new(),
            // Two beats referencing the same pool: dedup by ref must collapse them.
            beats: vec![practice_beat("greetings"), practice_beat("greetings")],
        };
        let mut rng = StdRng::seed_from_u64(7);
        let qs = build_challenge(&mut rng, &chapter, std::slice::from_ref(&lesson), &[]);
        assert_eq!(qs.len(), CHALLENGE_LENGTH);
        let ids: HashSet<_> = qs
            .iter()
            .map(|q| q.card.as_ref().unwrap().id.clone())
            .collect();
        assert_eq!(ids.len(), CHALLENGE_LENGTH, "questions are distinct");
    }

    #[test]
    fn no_practice_beats_yields_empty() {
        let chapter = Chapter {
            id: "ch".to_string(),
            title: String::new(),
            beats: vec![Beat {
                kind: BeatKind::Narration,
                speaker: String::new(),
                place: String::new(),
                source: "hola".to_string(),
                jp: String::new(),
                romaji: String::new(),
                practice: None,
                ref_id: String::new(),
            }],
        };
        let mut rng = StdRng::seed_from_u64(1);
        assert!(build_challenge(&mut rng, &chapter, &[], &[]).is_empty());
    }
}
