//! The app's fixed-size, centered border frame.
//!
//! Port of the Go `internal/ui` layout Frame. The outer size depends only on
//! the terminal size — never on the content — so the border stays put as the
//! user moves between screens. ratatui reflows on resize natively, so there is
//! no `WindowSizeMsg` to track.

use ratatui::layout::Rect;
use ratatui::widgets::{Block, BorderType, Borders, Padding};
use ratatui::Frame;

use crate::theme::Theme;

/// Frame outer dimensions, sized to fit the tallest screen. Upper bounds: the
/// frame shrinks to fit a smaller terminal.
pub const FRAME_WIDTH: u16 = 64;
pub const FRAME_HEIGHT: u16 = 23;

/// Draws the centered border frame and returns the inner content `Rect`.
pub fn draw_frame(f: &mut Frame, theme: &Theme) -> Rect {
    let area = f.area();
    let w = FRAME_WIDTH.min(area.width);
    let h = FRAME_HEIGHT.min(area.height);
    let x = area.x + area.width.saturating_sub(w) / 2;
    let y = area.y + area.height.saturating_sub(h) / 2;
    let rect = Rect {
        x,
        y,
        width: w,
        height: h,
    };

    let block = Block::default()
        .borders(Borders::ALL)
        .border_type(BorderType::Rounded)
        .border_style(theme.border)
        .padding(Padding::new(3, 3, 1, 1)); // matches lipgloss Padding(1, 3)
    let inner = block.inner(rect);
    f.render_widget(block, rect);
    inner
}

#[cfg(test)]
mod tests {
    use super::*;
    use ratatui::backend::TestBackend;
    use ratatui::text::Text;
    use ratatui::widgets::{Paragraph, Wrap};
    use ratatui::Terminal;

    /// Draws the frame on a `w`×`h` terminal and returns its inner content area.
    fn inner_on(w: u16, h: u16) -> Rect {
        let theme = Theme::plain();
        let mut terminal = Terminal::new(TestBackend::new(w, h)).unwrap();
        let mut inner = Rect::default();
        terminal
            .draw(|f| {
                inner = draw_frame(f, &theme);
            })
            .unwrap();
        inner
    }

    /// The content area is the frame minus its border and padding — the budget
    /// every screen lays out against.
    #[test]
    fn content_area_matches_the_frame_interior() {
        let inner = inner_on(80, 40);
        // 64 wide - 2 border - 6 padding = 56; 23 high - 2 border - 2 padding = 19.
        assert_eq!(inner.width, 56, "content width");
        assert_eq!(inner.height, 19, "content height");
    }

    /// The frame is centered and never exceeds the terminal.
    #[test]
    fn frame_is_centered_and_bounded() {
        let inner = inner_on(80, 40);
        assert_eq!(
            inner.x,
            (80 - FRAME_WIDTH) / 2 + 1 + 3,
            "centered + border + padding"
        );

        // On a terminal smaller than the frame, it shrinks instead of clipping.
        let small = inner_on(40, 12);
        assert!(small.width <= 40 && small.height <= 12);
    }

    /// Japanese prose has no spaces, and its characters occupy two cells. It
    /// must still be broken by display width so it wraps inside the frame
    /// instead of overflowing it.
    #[test]
    fn spaceless_japanese_prose_wraps_inside_the_frame() {
        let theme = Theme::plain();
        let jp = "これが基本の疑問詞です。どこ、どう、どちら。よく聞いてくださいね。";
        let mut terminal = Terminal::new(TestBackend::new(80, 30)).unwrap();
        terminal
            .draw(|f| {
                let inner = draw_frame(f, &theme);
                f.render_widget(
                    Paragraph::new(Text::raw(jp)).wrap(Wrap { trim: false }),
                    inner,
                );
            })
            .unwrap();

        // Nothing may be drawn outside the frame's content columns.
        let buf = terminal.backend().buffer();
        let inner = inner_on(80, 30);
        let mut rendered_rows = 0;
        for y in 0..buf.area.height {
            let mut row = String::new();
            for x in 0..buf.area.width {
                let sym = buf[(x, y)].symbol();
                // Border glyphs aside, only the content columns may hold text.
                if !sym.trim().is_empty() && !"╭╮╰╯│─".contains(sym) {
                    assert!(
                        x >= inner.x && x < inner.x + inner.width,
                        "text at column {x} escapes the content area [{}, {})",
                        inner.x,
                        inner.x + inner.width
                    );
                    row.push_str(sym);
                }
            }
            if !row.is_empty() {
                rendered_rows += 1;
            }
        }
        assert!(
            rendered_rows >= 2,
            "the sentence must wrap onto multiple lines, got {rendered_rows}"
        );
    }
}
