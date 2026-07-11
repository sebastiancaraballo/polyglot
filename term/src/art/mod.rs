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
