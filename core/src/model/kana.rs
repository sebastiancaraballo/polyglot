/// Distinguishes the two Japanese syllabaries.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum KanaType {
    Hiragana,
    Katakana,
}

impl KanaType {
    /// The wire string (`"hiragana"`/`"katakana"`), also used as the kana
    /// practice `RefID` in story beats.
    pub fn as_str(&self) -> &'static str {
        match self {
            KanaType::Hiragana => "hiragana",
            KanaType::Katakana => "katakana",
        }
    }

    /// Parses a kana type from its wire string, or `None` if unrecognized.
    #[allow(clippy::should_implement_trait)]
    pub fn from_str(s: &str) -> Option<KanaType> {
        match s {
            "hiragana" => Some(KanaType::Hiragana),
            "katakana" => Some(KanaType::Katakana),
            _ => None,
        }
    }
}

/// Groups kana by how they are formed: the base gojūon, voiced (dakuten) and
/// semi-voiced (handakuten) variants, and palatalized combinations (yōon).
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum KanaCategory {
    Base,
    Dakuten,
    Handakuten,
    Combo,
}

impl KanaCategory {
    /// The wire string for this category.
    pub fn as_str(&self) -> &'static str {
        match self {
            KanaCategory::Base => "base",
            KanaCategory::Dakuten => "dakuten",
            KanaCategory::Handakuten => "handakuten",
            KanaCategory::Combo => "combo",
        }
    }

    /// Parses a category from its wire string, or `None` if unrecognized.
    #[allow(clippy::should_implement_trait)]
    pub fn from_str(s: &str) -> Option<KanaCategory> {
        match s {
            "base" => Some(KanaCategory::Base),
            "dakuten" => Some(KanaCategory::Dakuten),
            "handakuten" => Some(KanaCategory::Handakuten),
            "combo" => Some(KanaCategory::Combo),
            _ => None,
        }
    }
}

/// A single kana character paired with its romaji reading.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct KanaItem {
    pub char: String,
    pub romaji: String,
    pub kana_type: KanaType,
    pub category: KanaCategory,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn category_from_str_validity() {
        for (s, want) in [
            ("base", true),
            ("dakuten", true),
            ("handakuten", true),
            ("combo", true),
            ("", false),
            ("bogus", false),
        ] {
            assert_eq!(KanaCategory::from_str(s).is_some(), want, "{s:?}");
        }
    }
}
