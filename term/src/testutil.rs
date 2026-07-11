//! Shared test helpers: render a screen into the app frame and flatten it to
//! text, for golden (insta) snapshots and content assertions — the analogue of
//! the Go screens' `golden.RequireEqual(View().Content)` tests.

use ratatui::backend::TestBackend;
use ratatui::layout::Rect;
use ratatui::Frame;
use ratatui::Terminal;

use crate::frame::draw_frame;
use crate::theme::Theme;

/// Renders `draw` into the centered app frame on an 80×30 test terminal and
/// returns the whole screen as text (trailing spaces trimmed for stable
/// snapshots). The plain theme keeps the output escape-free.
pub fn snapshot(draw: impl FnOnce(&mut Frame, Rect, &Theme)) -> String {
    let theme = Theme::plain();
    let mut terminal = Terminal::new(TestBackend::new(80, 30)).unwrap();
    terminal
        .draw(|f| {
            let inner = draw_frame(f, &theme);
            draw(f, inner, &theme);
        })
        .unwrap();

    let buf = terminal.backend().buffer();
    let mut lines = Vec::with_capacity(buf.area.height as usize);
    for y in 0..buf.area.height {
        let mut line = String::new();
        for x in 0..buf.area.width {
            line.push_str(buf[(x, y)].symbol());
        }
        lines.push(line.trim_end().to_string());
    }
    lines.join("\n")
}
