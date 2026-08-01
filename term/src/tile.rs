//! The focal tile: one character, boxed and centered, used as the prompt of the
//! character trainers.
//!
//! Shared by the kana and kanji trainers so the two read as the same activity —
//! and so the character stays the visual subject of the question rather than a
//! line of text among others.

/// Draws `stimulus` inside a centered, padded box `width` columns wide. A
/// terminal cannot change the font size, so prominence comes from a wide border
/// and generous padding (2 rows, 6 columns) rather than from a larger glyph.
pub fn big_tile(stimulus: &str, width: u16) -> Vec<String> {
    const PAD_H: usize = 6;
    const PAD_V: usize = 2;
    let sw = display_width(stimulus);
    let inner_w = sw + PAD_H * 2;
    let lead = " ".repeat((width as usize).saturating_sub(inner_w + 2) / 2);

    let border = |l: &str, r: &str| format!("{lead}{l}{}{r}", "─".repeat(inner_w));
    let blank = format!("{lead}│{}│", " ".repeat(inner_w));
    let glyph = format!(
        "{lead}│{}{stimulus}{}│",
        " ".repeat(PAD_H),
        " ".repeat(inner_w - sw - PAD_H)
    );

    let mut rows = vec![border("╭", "╮")];
    rows.extend(std::iter::repeat_n(blank.clone(), PAD_V));
    rows.push(glyph);
    rows.extend(std::iter::repeat_n(blank, PAD_V));
    rows.push(border("╰", "╯"));
    rows
}

/// Terminal cells a string occupies: kana and kanji are full-width (two cells).
pub fn display_width(s: &str) -> usize {
    s.chars()
        .map(|c| {
            let u = c as u32;
            if (0x3040..=0x30FF).contains(&u) || (0x4E00..=0x9FFF).contains(&u) {
                2
            } else {
                1
            }
        })
        .sum()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tile_is_a_padded_centered_box() {
        let rows = big_tile("あ", 40);
        assert_eq!(rows.len(), 7, "top + 2 pad + glyph + 2 pad + bottom");
        assert!(rows[0].contains('╭') && rows[0].contains('╮'));
        assert!(rows[3].contains('あ'), "the glyph sits in the middle row");
        assert!(rows[6].contains('╰') && rows[6].contains('╯'));
        assert!(rows[0].starts_with(' '), "centered, so indented");
    }

    /// A kanji is full-width like a kana, so the box comes out the same size —
    /// the two trainers look like one activity.
    #[test]
    fn a_kanji_tile_matches_a_kana_tile() {
        let kana = big_tile("あ", 40);
        let kanji = big_tile("日", 40);
        assert_eq!(kana[0], kanji[0], "same box for the same display width");
        assert!(kanji[3].contains('日'));
    }

    #[test]
    fn full_width_characters_count_as_two_cells() {
        assert_eq!(display_width("あ"), 2);
        assert_eq!(display_width("日"), 2);
        assert_eq!(display_width("a"), 1);
        assert_eq!(display_width("きゃ"), 4);
    }
}
