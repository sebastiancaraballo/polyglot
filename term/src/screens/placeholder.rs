//! A placeholder for destinations not yet ported to ratatui, so navigation is
//! exercisable end-to-end during the TUI port. Each ported screen replaces its
//! placeholder.

use ratatui::crossterm::event::{KeyCode, KeyModifiers};
use ratatui::layout::Rect;
use ratatui::text::Line;
use ratatui::widgets::Paragraph;
use ratatui::Frame;

use crate::app::{Dest, Transition};
use crate::theme::Theme;

pub struct Placeholder {
    title: &'static str,
}

impl Placeholder {
    pub fn new(dest: Dest) -> Placeholder {
        Placeholder {
            title: dest.title(),
        }
    }

    pub fn handle(
        &mut self,
        code: KeyCode,
        mods: KeyModifiers,
        _ctx: &crate::app::Ctx<'_>,
    ) -> Transition {
        match code {
            KeyCode::Char('c') if mods.contains(KeyModifiers::CONTROL) => Transition::Quit,
            KeyCode::Esc | KeyCode::Left | KeyCode::Backspace | KeyCode::Char('q') => {
                Transition::Pop
            }
            _ => Transition::Stay,
        }
    }

    pub fn render(&self, f: &mut Frame, inner: Rect, theme: &Theme) {
        let lines = vec![
            Line::styled(self.title.to_string(), theme.title),
            Line::raw(""),
            Line::styled("Pantalla aún no portada a Rust.", theme.normal),
            Line::styled("Próximamente.", theme.subtle),
        ];
        let body = Rect {
            height: inner.height.saturating_sub(1),
            ..inner
        };
        let help_area = Rect {
            y: inner.y + inner.height.saturating_sub(1),
            height: 1,
            ..inner
        };
        f.render_widget(Paragraph::new(lines), body);
        f.render_widget(
            Paragraph::new(Line::styled("ESC volver", theme.help)),
            help_area,
        );
    }
}
