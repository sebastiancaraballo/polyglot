use std::collections::HashMap;

use crate::model::{KanaCategory, KanaItem, KanaProgress, KanaType};

/// Summarizes how much of one syllabary's base set a learner has driven to
/// automaticity.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct Fluency {
    pub mastered: i64,
    pub total: i64,
}

impl Fluency {
    /// Reports whether every base kana in the syllabary is mastered.
    pub fn fluent(&self) -> bool {
        self.total > 0 && self.mastered >= self.total
    }
}

fn kana_fluency(
    items: &[KanaItem],
    progress: &HashMap<String, KanaProgress>,
    typ: KanaType,
) -> Fluency {
    let mut f = Fluency::default();
    for it in items {
        if it.kana_type != typ || it.category != KanaCategory::Base {
            continue;
        }
        f.total += 1;
        if progress.get(&it.char).is_some_and(|p| p.mastered) {
            f.mastered += 1;
        }
    }
    f
}

/// The Foundations decoding gate, derived from a learner's kana progress.
/// Katakana practice unlocks once hiragana is fluent; reading Japanese words
/// and sentences unlocks once both syllabaries are fluent.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct Gate {
    pub hiragana: Fluency,
    pub katakana: Fluency,
}

impl Gate {
    /// Reports whether katakana practice is available yet.
    pub fn katakana_unlocked(&self) -> bool {
        self.hiragana.fluent()
    }

    /// Reports whether reading Japanese words and sentences is available yet:
    /// both syllabaries must be fluent.
    pub fn reading_unlocked(&self) -> bool {
        self.hiragana.fluent() && self.katakana.fluent()
    }
}

/// Computes the gate from the full kana set and a learner's progress.
pub fn new_gate(items: &[KanaItem], progress: &HashMap<String, KanaProgress>) -> Gate {
    Gate {
        hiragana: kana_fluency(items, progress, KanaType::Hiragana),
        katakana: kana_fluency(items, progress, KanaType::Katakana),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn base(char: &str, typ: KanaType) -> KanaItem {
        KanaItem {
            char: char.to_string(),
            romaji: String::new(),
            kana_type: typ,
            category: KanaCategory::Base,
        }
    }

    fn mastered(chars: &[&str]) -> HashMap<String, KanaProgress> {
        chars
            .iter()
            .map(|c| {
                (
                    c.to_string(),
                    KanaProgress {
                        char: c.to_string(),
                        mastered: true,
                        ..Default::default()
                    },
                )
            })
            .collect()
    }

    #[test]
    fn gate_unlocks_progressively() {
        let items = vec![
            base("あ", KanaType::Hiragana),
            base("い", KanaType::Hiragana),
            base("ア", KanaType::Katakana),
        ];

        let g = new_gate(&items, &mastered(&[]));
        assert!(!g.katakana_unlocked());
        assert!(!g.reading_unlocked());

        let g = new_gate(&items, &mastered(&["あ", "い"]));
        assert!(g.katakana_unlocked(), "hiragana fluent unlocks katakana");
        assert!(!g.reading_unlocked());

        let g = new_gate(&items, &mastered(&["あ", "い", "ア"]));
        assert!(g.reading_unlocked(), "both fluent unlocks reading");
    }
}
