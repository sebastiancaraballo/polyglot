//! ASCII/braille art for the terminal UI: the block wordmark and the rotating
//! globe. Port of the Go `internal/art` package.

mod globe;

pub use globe::GLOBE_FRAMES;

/// The block-letter app name shown in the main-menu header, sized to span the
/// shared fixed-width frame (4 rows, 55 columns). Full-block glyphs (█) and
/// spaces only, so it renders consistently across terminals and fonts.
pub const WORDMARK: &str = "\
██████ ██████ ██     ██  ██ ██████ ██     ██████ ██████
██  ██ ██  ██ ██      ████  ██     ██     ██  ██   ██  
██████ ██  ██ ██       ██   ██ ███ ██     ██  ██   ██  
██     ██████ ██████   ██   ██████ ██████ ██████   ██  ";

#[cfg(test)]
mod tests {
    use super::*;

    /// Guards the generated frames: every frame must be a rectangle of
    /// identical, non-empty dimensions so the header layout stays aligned, and
    /// every cell must be a braille glyph (U+2800–U+28FF).
    #[test]
    fn globe_frames_are_uniform_braille() {
        assert!(
            GLOBE_FRAMES.len() >= 2,
            "an animation needs at least two frames"
        );

        let rows = GLOBE_FRAMES[0].lines().count();
        let cols = GLOBE_FRAMES[0].lines().next().unwrap().chars().count();
        assert!(rows > 0 && cols > 0, "frame 0 is empty");

        for (i, frame) in GLOBE_FRAMES.iter().enumerate() {
            let lines: Vec<&str> = frame.lines().collect();
            assert_eq!(lines.len(), rows, "frame {i} row count");
            for (j, line) in lines.iter().enumerate() {
                assert_eq!(line.chars().count(), cols, "frame {i} line {j} width");
                for c in line.chars() {
                    assert!(
                        ('\u{2800}'..='\u{28FF}').contains(&c),
                        "frame {i} line {j} has non-braille char {c:?}"
                    );
                }
            }
        }
    }

    /// The wordmark is a rectangle of full blocks and spaces only, so it renders
    /// identically across terminals and fonts.
    #[test]
    fn wordmark_is_a_uniform_block_rectangle() {
        let lines: Vec<&str> = WORDMARK.lines().collect();
        assert_eq!(lines.len(), 4, "wordmark row count");
        let width = lines[0].chars().count();
        assert_eq!(width, 55, "wordmark width");
        for (i, line) in lines.iter().enumerate() {
            assert_eq!(line.chars().count(), width, "wordmark line {i} width");
            for c in line.chars() {
                assert!(
                    c == '█' || c == ' ',
                    "wordmark line {i} has unexpected char {c:?}"
                );
            }
        }
    }
}
