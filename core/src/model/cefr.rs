/// A Common European Framework of Reference for Languages level, from A1
/// (beginner) to C2 (mastery). It grades communicative functions in the
/// language-agnostic curriculum spine, independent of any single language's own
/// proficiency scale (e.g. JLPT for Japanese).
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum Cefr {
    A1,
    A2,
    B1,
    B2,
    C1,
    C2,
}

/// Every level ordered from beginner to mastery.
pub const CEFR_LEVELS: [Cefr; 6] = [Cefr::A1, Cefr::A2, Cefr::B1, Cefr::B2, Cefr::C1, Cefr::C2];

impl Cefr {
    /// The wire string for this level (as it appears in YAML content).
    pub fn as_str(&self) -> &'static str {
        match self {
            Cefr::A1 => "A1",
            Cefr::A2 => "A2",
            Cefr::B1 => "B1",
            Cefr::B2 => "B2",
            Cefr::C1 => "C1",
            Cefr::C2 => "C2",
        }
    }

    /// Parses a level from its wire string, or `None` if unrecognized (the
    /// analogue of the Go `CEFR.Valid` check).
    #[allow(clippy::should_implement_trait)]
    pub fn from_str(s: &str) -> Option<Cefr> {
        CEFR_LEVELS.into_iter().find(|c| c.as_str() == s)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn from_str_validity() {
        for (s, want) in [
            ("A1", true),
            ("A2", true),
            ("B1", true),
            ("B2", true),
            ("C1", true),
            ("C2", true),
            ("", false),
            ("A0", false),
            ("bogus", false),
        ] {
            assert_eq!(Cefr::from_str(s).is_some(), want, "{s:?}");
        }
    }
}
