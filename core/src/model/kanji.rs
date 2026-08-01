use super::Jlpt;

/// A single teachable kanji.
///
/// Unlike a [`KanaItem`](super::KanaItem) — one character, one reading — a kanji
/// carries *several* readings, and which one applies depends on the word it
/// appears in: on'yomi (the reading borrowed from Chinese, used mostly in
/// compounds) and kun'yomi (the native Japanese reading, used mostly standalone).
/// The engine stores both and does not try to pick one; a card's own `romaji`
/// remains the authority for how that particular word is read.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct KanjiItem {
    /// The character itself, e.g. `"日"`.
    pub char: String,
    /// On'yomi readings, in kana (e.g. `["ニチ", "ジツ"]`). May be empty.
    pub on: Vec<String>,
    /// Kun'yomi readings, in kana (e.g. `["ひ", "か"]`). May be empty.
    pub kun: Vec<String>,
    /// Meaning in the learner's source language (Spanish for v1).
    pub meaning: String,
    /// Proficiency level this kanji belongs to.
    pub jlpt: Option<Jlpt>,
}

impl KanjiItem {
    /// Every reading, on'yomi first — for display and for building answer pools.
    pub fn readings(&self) -> Vec<&str> {
        self.on
            .iter()
            .chain(self.kun.iter())
            .map(String::as_str)
            .collect()
    }
}

/// Returns whether `c` is a Han character (a kanji): the CJK Unified Ideographs
/// blocks plus the compatibility block. Shared by the content validator, the
/// decoder and the study logic so they always agree on what counts as a kanji.
pub fn is_han(c: char) -> bool {
    let u = c as u32;
    (0x3400..=0x4DBF).contains(&u)          // CJK Extension A
        || (0x4E00..=0x9FFF).contains(&u)   // CJK Unified Ideographs
        || (0xF900..=0xFAFF).contains(&u)   // CJK Compatibility Ideographs
        || (0x20000..=0x2FA1F).contains(&u) // Extensions B-F + compat supplement
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn han_recognizes_kanji_and_rejects_kana() {
        for c in ['日', '本', '語', '学', '一'] {
            assert!(is_han(c), "{c:?} is a kanji");
        }
        for c in ['あ', 'ア', 'ー', 'っ', 'a', '1', '、'] {
            assert!(!is_han(c), "{c:?} is not a kanji");
        }
    }

    #[test]
    fn readings_lists_on_before_kun() {
        let k = KanjiItem {
            char: "日".to_string(),
            on: vec!["ニチ".to_string(), "ジツ".to_string()],
            kun: vec!["ひ".to_string()],
            meaning: "día, sol".to_string(),
            jlpt: Some(Jlpt::N5),
        };
        assert_eq!(k.readings(), vec!["ニチ", "ジツ", "ひ"]);
    }

    #[test]
    fn a_kanji_may_have_only_one_kind_of_reading() {
        let k = KanjiItem {
            char: "私".to_string(),
            on: Vec::new(),
            kun: vec!["わたし".to_string()],
            meaning: "yo".to_string(),
            jlpt: Some(Jlpt::N5),
        };
        assert_eq!(k.readings(), vec!["わたし"]);
    }
}
