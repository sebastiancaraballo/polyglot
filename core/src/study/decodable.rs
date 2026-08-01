use std::collections::{HashMap, HashSet};

use crate::model::{is_han, KanaItem, KanaProgress, KanjiItem, KanjiProgress};

/// Reports which Japanese text a learner can already *read*, given the kana and
/// kanji they have driven to automaticity. It implements the progressive side of
/// the Foundations gate: instead of locking all reading until every character is
/// fluent, the learner meets "decodable" words and sentences — those built only
/// from characters they have already mastered — and the readable set grows as
/// they do. This mirrors the decodable-texts approach to early reading.
pub struct Decoder {
    /// Valid two-scalar kana strings (yōon, e.g. `"きゃ"`).
    combos: HashSet<String>,
    /// Kana the learner has driven to automaticity.
    mastered: HashSet<String>,
    /// Kanji the learner has mastered. Empty until a pair teaches kanji, which
    /// keeps the pre-kanji behavior — no kanji is readable — exactly as it was.
    mastered_kanji: HashSet<String>,
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
        Decoder {
            combos,
            mastered,
            mastered_kanji: HashSet::new(),
        }
    }

    /// Adds the learner's kanji progress. Separate from [`Decoder::new`] so a
    /// caller that has no kanji to offer keeps working unchanged.
    pub fn with_kanji(
        mut self,
        _items: &[KanjiItem],
        progress: &HashMap<String, KanjiProgress>,
    ) -> Decoder {
        self.mastered_kanji = progress
            .iter()
            .filter(|(_, p)| p.mastered)
            .map(|(char, _)| char.clone())
            .collect();
        self
    }

    /// Reports whether `jp` can be read with only the characters the learner has
    /// mastered. It tokenizes with longest match (a yōon combo before its
    /// parts), the same way the content loader validates kana coverage. A
    /// kanji counts as readable once mastered; a string with no kana at all is
    /// not decodable, since the gate is about reading Japanese script.
    pub fn decodable(&self, jp: &str) -> bool {
        let runes: Vec<char> = jp.chars().collect();
        let mut saw_script = false;
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
                        saw_script = true;
                        i += 2;
                        continue;
                    }
                }
                if !self.mastered.contains(&r.to_string()) {
                    return false;
                }
                saw_script = true;
                i += 1;
            } else if is_han(r) {
                if !self.mastered_kanji.contains(&r.to_string()) {
                    return false;
                }
                saw_script = true;
                i += 1;
            } else {
                i += 1; // skip punctuation, spaces, ASCII
            }
        }
        saw_script
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

    fn kanji_decoder(kana_mastered: &[&str], kanji_mastered: &[&str]) -> Decoder {
        let progress: HashMap<String, KanjiProgress> = kanji_mastered
            .iter()
            .map(|c| {
                (
                    c.to_string(),
                    KanjiProgress {
                        char: c.to_string(),
                        mastered: true,
                        ..Default::default()
                    },
                )
            })
            .collect();
        decoder_for(kana_mastered).with_kanji(&[], &progress)
    }

    /// A kanji is unreadable until mastered, and a mixed word needs every one of
    /// its characters — the same all-or-nothing rule kana already follow.
    #[test]
    fn kanji_decodable_once_mastered() {
        // No kanji progress at all: the pre-kanji behavior is unchanged.
        assert!(!decoder_for(&["は"]).decodable("日本は"));

        // Knowing one of the two kanji is not enough.
        assert!(!kanji_decoder(&["は"], &["日"]).decodable("日本は"));

        // Every character known: readable.
        assert!(kanji_decoder(&["は"], &["日", "本"]).decodable("日本は"));

        // A kanji-only word is readable too — the gate is about script, not kana.
        assert!(kanji_decoder(&[], &["日", "本"]).decodable("日本"));

        // A mastered kanji does not rescue an unmastered kana.
        assert!(!kanji_decoder(&[], &["日"]).decodable("日は"));
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
