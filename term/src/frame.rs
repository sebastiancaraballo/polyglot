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
