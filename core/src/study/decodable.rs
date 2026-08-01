use std::collections::{HashMap, HashSet};

use crate::model::{KanaItem, KanaProgress};

/// Reports which Japanese text a learner can already *read*, given the kana
/// they have driven to automaticity. It implements the progressive side of the
/// Foundations gate: instead of locking all reading until every kana is fluent,
/// the learner meets "decodable" words and sentences — those built only from
/// kana they have already mastered — and the readable set grows as they do.
/// This mirrors the decodable-texts approach to early reading.
pub struct Decoder {
    /// Valid two-scalar kana strings (yōon, e.g. `"きゃ"`).
    combos: HashSet<String>,
    /// Kana the learner has driven to automaticity.
    mastered: HashSet<String>,
}

impl Decoder {
    /// Builds a decoder from the course's kana set (so it can recognize
    /// multi-scalar combos) and a learner's progress.
    pub fn new(items: &[KanaItem], progress: &HashMap<String, KanaProgress>) -> Decoder {
        let mut combos = HashSet::new();
        for it in items {
            if it.char.chars().count() == 2 {
                combos.insert(it.char.clone());
            }
        }
        let mut mastered = HashSet::new();
        for (char, p) in progress {
            if p.mastered {
                mastered.insert(char.clone());
            }
        }
        Decoder { combos, mastered }
    }

    /// Reports whether `jp` can be read with only the learner's mastered kana.
    /// It tokenizes with longest match (a yōon combo before its parts), the
    /// same way the content loader validates kana coverage. A string with no
    /// kana, or containing kanji (whose decoding is a separate, later gate), is
    /// not decodable.
    pub fn decodable(&self, jp: &str) -> bool {
        let runes: Vec<char> = jp.chars().collect();
        let mut saw_kana = false;
        let mut i = 0;
        while i < runes.len() {
            let r = runes[i];
            if is_kana_mark(r) {
                // sokuon (っ/ッ) and chōonpu (ー) modify a neighbor; always
                // readable once you know kana.
                i += 1;
            } else if is_kana(r) {
                if i + 1 < runes.len() {
                    let pair: String = runes[i..i + 2].iter().collect();
                    if self.combos.contains(&pair) {
                        if !self.mastered.contains(&pair) {
                            return false;
                        }
                        saw_kana = true;
                        i += 2;
                        continue;
                    }
                }
                if !self.mastered.contains(&r.to_string()) {
                    return false;
                }
                saw_kana = true;
                i += 1;
            } else if is_han(r) {
                return false; // kanji is not yet decodable
            } else {
                i += 1; // skip punctuation, spaces, ASCII
            }
        }
        saw_kana
    }
}

/// Reports whether `c` belongs to either Japanese syllabary. Kana marks with no
/// standalone reading are handled before this by [`is_kana_mark`].
fn is_kana(c: char) -> bool {
    let u = c as u32;
    (0x3041..=0x309F).contains(&u) || (0x30A0..=0x30FF).contains(&u)
}

/// Reports whether `c` is a kana modifier with no standalone reading: the
/// sokuon (small tsu っ/ッ) or the chōonpu (ー). They modify an adjacent kana,
/// so a learner who knows kana can already read them.
fn is_kana_mark(c: char) -> bool {
    matches!(c, 'っ' | 'ッ' | 'ー')
}

/// Reports whether `c` is a Han (kanji) ideograph.
fn is_han(c: char) -> bool {
    let u = c as u32;
    (0x3400..=0x4DBF).contains(&u)      // CJK Extension A
        || (0x4E00..=0x9FFF).contains(&u)  // CJK Unified Ideographs
        || (0xF900..=0xFAFF).contains(&u)  // CJK Compatibility Ideographs
        || (0x20000..=0x2FA1F).contains(&u) // Extensions B–F + compat supplement
}

#[cfg(test)]
mod tests {
    use super::*;

    fn item(char: &str) -> KanaItem {
        use crate::model::{KanaCategory, KanaType};
        KanaItem {
            char: char.to_string(),
            romaji: String::new(),
            kana_type: KanaType::Hiragana,
            category: if char.chars().count() == 2 {
                KanaCategory::Combo
            } else {
                KanaCategory::Base
            },
        }
    }

    fn decoder_for(mastered: &[&str]) -> Decoder {
        let items: Vec<KanaItem> = [
            "こ", "ん", "に", "ち", "は", "き", "ゅ", "う", "きゅ", "が", "コ", "ヒ",
        ]
        .iter()
        .map(|c| item(c))
        .collect();
        let progress: HashMap<String, KanaProgress> = mastered
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
            .collect();
        Decoder::new(&items, &progress)
    }

    #[test]
    fn decodable_cases() {
        let cases: &[(&[&str], &str, bool)] = &[
            (&["こ", "ん", "に", "ち", "は"], "こんにちは", true),
            (&["こ", "ん", "に", "ち"], "こんにちは", false),
            (&["き", "ゅ", "う"], "きゅう", false),
            (&["きゅ", "う"], "きゅう", true),
            (&["こ"], "", false),
            (&["こ", "ん", "に", "ち", "は"], "日本は", false),
            (&["が", "こ", "う"], "がっこう", true),
            (&["が", "こ"], "がっこう", false),
            (&["コ", "ヒ"], "コーヒー", true),
        ];
        for (mastered, jp, want) in cases {
            let d = decoder_for(mastered);
            assert_eq!(d.decodable(jp), *want, "decodable({jp:?})");
        }
    }
}
