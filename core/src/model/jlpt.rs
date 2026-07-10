/// A Japanese-Language Proficiency Test level, from N5 (easiest) to N1
/// (hardest). Levels are used as a motivational progress indicator.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum Jlpt {
    N5,
    N4,
    N3,
    N2,
    N1,
}

/// Every level ordered from easiest to hardest.
pub const JLPT_LEVELS: [Jlpt; 5] = [Jlpt::N5, Jlpt::N4, Jlpt::N3, Jlpt::N2, Jlpt::N1];

impl Jlpt {
    /// The wire string for this level (as it appears in YAML content).
    pub fn as_str(&self) -> &'static str {
        match self {
            Jlpt::N5 => "N5",
            Jlpt::N4 => "N4",
            Jlpt::N3 => "N3",
            Jlpt::N2 => "N2",
            Jlpt::N1 => "N1",
        }
    }

    /// Parses a level from its wire string, or `None` if unrecognized.
    #[allow(clippy::should_implement_trait)]
    pub fn from_str(s: &str) -> Option<Jlpt> {
        JLPT_LEVELS.into_iter().find(|j| j.as_str() == s)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn from_str_validity() {
        for (s, want) in [("N5", true), ("N1", true), ("", false), ("N6", false)] {
            assert_eq!(Jlpt::from_str(s).is_some(), want, "{s:?}");
        }
    }
}
