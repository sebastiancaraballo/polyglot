//! Terminal theme: the styles used across screens.
//!
//! Port of the Go `internal/ui` theme. A high-contrast variant avoids color and
//! relies on bold/reverse so the UI stays legible without color (honoring
//! `NO_COLOR`).

use ratatui::style::{Color, Modifier, Style};

/// Reports whether color output should be disabled, following the `NO_COLOR`
/// convention (<https://no-color.org>): the variable is set and non-empty.
pub fn no_color() -> bool {
    std::env::var("NO_COLOR")
        .map(|v| !v.is_empty())
        .unwrap_or(false)
}

/// The set of styles used across screens. Some styles are consumed only by
/// screens still being ported, so the whole palette is retained.
#[allow(dead_code)]
#[derive(Clone, Copy)]
pub struct Theme {
    pub title: Style,
    pub subtle: Style,
    pub normal: Style,
    pub selected: Style,
    pub accent: Style,
    pub success: Style,
    pub error: Style,
    pub help: Style,
    pub border: Style,
}

impl Theme {
    /// The theme appropriate for the current environment, switching to
    /// high-contrast when `NO_COLOR` is set.
    pub fn default_theme() -> Theme {
        Theme::new(no_color())
    }

    /// A theme with no color or text attributes, for deterministic,
    /// escape-free snapshot tests.
    #[allow(dead_code)]
    pub fn plain() -> Theme {
        let p = Style::new();
        Theme {
            title: p,
            subtle: p,
            normal: p,
            selected: p,
            accent: p,
            success: p,
            error: p,
            help: p,
            border: p,
        }
    }

    /// Builds a theme. When `high_contrast` is true, colors are dropped in favor
    /// of bold and reverse styling.
    pub fn new(high_contrast: bool) -> Theme {
        if high_contrast {
            return Theme {
                title: Style::new().add_modifier(Modifier::BOLD),
                subtle: Style::new().add_modifier(Modifier::DIM),
                normal: Style::new(),
                selected: Style::new().add_modifier(Modifier::BOLD | Modifier::REVERSED),
                accent: Style::new().add_modifier(Modifier::BOLD),
                success: Style::new().add_modifier(Modifier::BOLD),
                error: Style::new().add_modifier(Modifier::BOLD),
                help: Style::new().add_modifier(Modifier::DIM),
                border: Style::new(),
            };
        }

        let accent = Color::Indexed(63); // indigo
        let subtle = Color::Indexed(245); // grey
        let success = Color::Indexed(42); // green
        let danger = Color::Indexed(203); // red
        Theme {
            title: Style::new().fg(accent).add_modifier(Modifier::BOLD),
            subtle: Style::new().fg(subtle),
            normal: Style::new(),
            selected: Style::new().fg(accent).add_modifier(Modifier::BOLD),
            accent: Style::new().fg(accent),
            success: Style::new().fg(success),
            error: Style::new().fg(danger),
            help: Style::new().fg(subtle),
            border: Style::new().fg(accent),
        }
    }
}
