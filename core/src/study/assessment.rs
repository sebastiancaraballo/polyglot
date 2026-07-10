use std::collections::{HashMap, HashSet};

use rand::seq::SliceRandom;
use rand::Rng;

use super::choice::options;
use crate::model::{Card, Jlpt, KanaItem, Lesson, Pattern};

/// How many questions a full assessment asks when the curriculum can supply
/// them: more than a 5-question chapter challenge, so an 80% bar is a
/// meaningful level check.
pub const ASSESSMENT_LENGTH: usize = 15;

/// The number of multiple-choice options per question (one correct answer plus
/// up to three distractors), matching the study screens.
const OPTION_COUNT: usize = 4;

/// Which strand an assessment question tests.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum AssessKind {
    /// Recall a word's Japanese form from its gloss.
    Vocab,
    /// Read a kana character.
    Kana,
    /// Fill a blanked slot in a grammar pattern.
    Pattern,
}

/// One multiple-choice question. Its `options` and `correct` index are built at
/// sample time (with the injected RNG) so the sampler is fully deterministic
/// and unit-testable; the screen only renders them.
#[derive(Clone, Debug, PartialEq)]
pub struct AssessQuestion {
    pub kind: AssessKind,
    /// `Vocab`: the prompt card. `Pattern`: the correct filler card.
    pub card: Option<Card>,
    /// `Kana`: the prompted character.
    pub kana: Option<KanaItem>,
    /// `Pattern`: the pattern being drilled.
    pub pattern: Option<Pattern>,
    /// `Pattern`: index of the blanked slot in `pattern.slots`.
    pub slot_idx: usize,
    /// `Pattern`: the non-blank slots' names -> default JP.
    pub fill: HashMap<String, String>,
    /// The choice strings.
    pub options: Vec<String>,
    /// Index of the correct option in `options`.
    pub correct: usize,
}

impl AssessQuestion {
    /// Identifies the item this question tests, for deduplication across
    /// strands.
    fn key(&self) -> String {
        match self.kind {
            AssessKind::Kana => {
                format!(
                    "kana:{}",
                    self.kana.as_ref().map_or("", |k| k.char.as_str())
                )
            }
            AssessKind::Pattern => {
                format!(
                    "pattern:{}",
                    self.pattern.as_ref().map_or("", |p| p.id.as_str())
                )
            }
            AssessKind::Vocab => {
                format!("card:{}", self.card.as_ref().map_or("", |c| c.id.as_str()))
            }
        }
    }
}

/// Draws up to [`ASSESSMENT_LENGTH`] questions for `level`, sampled without
/// replacement and round-robin across the three strands (vocab, kana, patterns)
/// so every strand the curriculum supplies contributes. `cards` resolves a
/// pattern slot's candidate/default card IDs back into full cards.
pub fn build_assessment<R: Rng + ?Sized>(
    rng: &mut R,
    level: Jlpt,
    lessons: &[Lesson],
    kana: &[KanaItem],
    patterns: &[Pattern],
    cards: &HashMap<String, Card>,
) -> Vec<AssessQuestion> {
    let mut pools = [
        vocab_questions(rng, level, lessons),
        kana_questions(rng, kana),
        pattern_questions(rng, level, patterns, cards),
    ];

    let mut seen: HashSet<String> = HashSet::new();
    let mut out: Vec<AssessQuestion> = Vec::new();
    while out.len() < ASSESSMENT_LENGTH {
        let mut progressed = false;
        for pool in pools.iter_mut() {
            if out.len() >= ASSESSMENT_LENGTH {
                break;
            }
            if pool.is_empty() {
                continue;
            }
            let q = pool.remove(0);
            progressed = true;
            if !seen.insert(q.key()) {
                continue;
            }
            out.push(q);
        }
        if !progressed {
            break;
        }
    }
    out
}

/// Builds a shuffled, capped pool of vocab questions from every card at the
/// given level, with distractors drawn from that same card set.
fn vocab_questions<R: Rng + ?Sized>(
    rng: &mut R,
    level: Jlpt,
    lessons: &[Lesson],
) -> Vec<AssessQuestion> {
    let mut cards: Vec<Card> = Vec::new();
    let mut pool: Vec<String> = Vec::new();
    for l in lessons {
        if l.jlpt != Some(level) {
            continue;
        }
        for c in &l.cards {
            cards.push(c.clone());
            pool.push(c.jp.clone());
        }
    }
    cards.shuffle(rng);
    cards.truncate(ASSESSMENT_LENGTH);

    let mut out = Vec::with_capacity(cards.len());
    for c in cards {
        let (opts, correct) = options(rng, &c.jp, &pool, OPTION_COUNT);
        out.push(AssessQuestion {
            kind: AssessKind::Vocab,
            card: Some(c),
            kana: None,
            pattern: None,
            slot_idx: 0,
            fill: HashMap::new(),
            options: opts,
            correct,
        });
    }
    out
}

/// Builds a shuffled, capped pool of kana-reading questions, with romaji
/// distractors drawn from the whole syllabary set.
fn kana_questions<R: Rng + ?Sized>(rng: &mut R, kana: &[KanaItem]) -> Vec<AssessQuestion> {
    let mut items: Vec<KanaItem> = kana.to_vec();
    let pool: Vec<String> = kana.iter().map(|k| k.romaji.clone()).collect();
    items.shuffle(rng);
    items.truncate(ASSESSMENT_LENGTH);

    let mut out = Vec::with_capacity(items.len());
    for k in items {
        let (opts, correct) = options(rng, &k.romaji, &pool, OPTION_COUNT);
        out.push(AssessQuestion {
            kind: AssessKind::Kana,
            card: None,
            kana: Some(k),
            pattern: None,
            slot_idx: 0,
            fill: HashMap::new(),
            options: opts,
            correct,
        });
    }
    out
}

/// Builds a shuffled, capped pool of grammar questions: one per pattern at the
/// given level, blanking a random slot and offering that slot's candidate
/// fillers as options.
fn pattern_questions<R: Rng + ?Sized>(
    rng: &mut R,
    level: Jlpt,
    patterns: &[Pattern],
    cards: &HashMap<String, Card>,
) -> Vec<AssessQuestion> {
    let mut pats: Vec<Pattern> = patterns
        .iter()
        .filter(|p| p.jlpt == Some(level) && !p.slots.is_empty())
        .cloned()
        .collect();
    pats.shuffle(rng);
    pats.truncate(ASSESSMENT_LENGTH);

    let mut out = Vec::new();
    for p in pats {
        let slot_idx = rng.gen_range(0..p.slots.len());
        let slot = p.slots[slot_idx].clone();
        let correct_card = match cards.get(&slot.default) {
            Some(c) => c.clone(),
            None => continue,
        };
        let mut fill: HashMap<String, String> = HashMap::new();
        for (i, s) in p.slots.iter().enumerate() {
            if i == slot_idx {
                continue;
            }
            if let Some(c) = cards.get(&s.default) {
                fill.insert(s.name.clone(), c.jp.clone());
            }
        }
        let mut cand_pool: Vec<String> = Vec::new();
        for id in &slot.card_ids {
            if let Some(c) = cards.get(id) {
                cand_pool.push(c.jp.clone());
            }
        }
        let (opts, correct) = options(rng, &correct_card.jp, &cand_pool, OPTION_COUNT);
        out.push(AssessQuestion {
            kind: AssessKind::Pattern,
            card: Some(correct_card),
            kana: None,
            pattern: Some(p),
            slot_idx,
            fill,
            options: opts,
            correct,
        });
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;
    use rand::rngs::StdRng;
    use rand::SeedableRng;

    fn card(id: &str, jp: &str) -> Card {
        Card {
            id: id.to_string(),
            source: id.to_string(),
            jp: jp.to_string(),
            romaji: String::new(),
            notes: String::new(),
            jlpt: Some(Jlpt::N5),
            functions: Vec::new(),
            freq: 0,
        }
    }

    #[test]
    fn samples_across_strands_capped_and_deduped() {
        let lesson = Lesson {
            id: "l".to_string(),
            title: String::new(),
            jlpt: Some(Jlpt::N5),
            functions: Vec::new(),
            cards: (0..20)
                .map(|i| card(&format!("l:{i}"), &format!("w{i}")))
                .collect(),
        };
        let kana: Vec<KanaItem> = ["あ", "い", "う", "え", "お"]
            .iter()
            .map(|c| {
                use crate::model::{KanaCategory, KanaType};
                KanaItem {
                    char: c.to_string(),
                    romaji: c.to_string(),
                    kana_type: KanaType::Hiragana,
                    category: KanaCategory::Base,
                }
            })
            .collect();
        let mut rng = StdRng::seed_from_u64(3);
        let qs = build_assessment(
            &mut rng,
            Jlpt::N5,
            std::slice::from_ref(&lesson),
            &kana,
            &[],
            &HashMap::new(),
        );

        assert_eq!(qs.len(), ASSESSMENT_LENGTH);
        let keys: HashSet<_> = qs.iter().map(|q| q.key()).collect();
        assert_eq!(keys.len(), qs.len(), "no duplicate items");
        assert!(
            qs.iter().any(|q| q.kind == AssessKind::Kana),
            "kana strand contributes"
        );
    }
}
