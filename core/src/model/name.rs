use std::fmt;

use unicode_general_category::{get_general_category, GeneralCategory};

/// The maximum length, in Unicode scalar values, of a profile name.
pub const MAX_NAME_LEN: usize = 24;

/// Why a profile name was rejected by [`normalize_name`].
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum NameError {
    /// The name was empty (or only whitespace).
    Empty,
    /// The name exceeded [`MAX_NAME_LEN`].
    TooLong,
    /// The name had a control character, or no letters.
    Invalid,
}

impl fmt::Display for NameError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let msg = match self {
            NameError::Empty => "name is empty",
            NameError::TooLong => "name is too long",
            NameError::Invalid => "name is invalid",
        };
        write!(f, "model: {msg}")
    }
}

impl std::error::Error for NameError {}

/// Trims a raw profile name and validates it. The check is simple but works for
/// names of any nationality: it accepts any string of Unicode letters (so every
/// script qualifies) plus marks, spaces, and common name punctuation, as long
/// as it has at least one letter, no control characters, and fits
/// [`MAX_NAME_LEN`] scalar values.
pub fn normalize_name(raw: &str) -> Result<String, NameError> {
    let name = raw.trim();
    if name.is_empty() {
        return Err(NameError::Empty);
    }
    if name.chars().count() > MAX_NAME_LEN {
        return Err(NameError::TooLong);
    }

    let mut has_letter = false;
    for c in name.chars() {
        if c.is_control() {
            return Err(NameError::Invalid);
        } else if is_letter(c) {
            has_letter = true;
        } else if is_mark(c) || c.is_whitespace() || is_name_punct(c) {
            // allowed
        } else {
            return Err(NameError::Invalid);
        }
    }
    if !has_letter {
        return Err(NameError::Invalid);
    }
    Ok(name.to_string())
}

fn is_letter(c: char) -> bool {
    matches!(
        get_general_category(c),
        GeneralCategory::UppercaseLetter
            | GeneralCategory::LowercaseLetter
            | GeneralCategory::TitlecaseLetter
            | GeneralCategory::ModifierLetter
            | GeneralCategory::OtherLetter
    )
}

fn is_mark(c: char) -> bool {
    matches!(
        get_general_category(c),
        GeneralCategory::NonspacingMark
            | GeneralCategory::SpacingMark
            | GeneralCategory::EnclosingMark
    )
}

/// Reports whether `c` is punctuation that legitimately appears in names across
/// languages (e.g. O'Brien, Jean-Luc, Nuñez·).
fn is_name_punct(c: char) -> bool {
    matches!(c, '-' | '\'' | '\u{2019}' | '.' | '\u{00B7}')
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn valid_names() {
        let a = "a".repeat(MAX_NAME_LEN);
        let cases = [
            ("Sebastián", "Sebastián"),
            ("  José Niño  ", "José Niño"),
            ("李", "李"),
            ("Анна", "Анна"),
            ("O'Brien", "O'Brien"),
            ("Jean-Luc", "Jean-Luc"),
            (a.as_str(), a.as_str()),
        ];
        for (input, want) in cases {
            assert_eq!(normalize_name(input), Ok(want.to_string()), "{input:?}");
        }
    }

    #[test]
    fn invalid_names() {
        let too_long = "a".repeat(MAX_NAME_LEN + 1);
        let cases = [
            ("", NameError::Empty),
            ("   ", NameError::Empty),
            (too_long.as_str(), NameError::TooLong),
            ("123", NameError::Invalid),
            ("ab\ncd", NameError::Invalid),
            ("🎉", NameError::Invalid),
        ];
        for (input, want) in cases {
            assert_eq!(normalize_name(input), Err(want), "{input:?}");
        }
    }
}
