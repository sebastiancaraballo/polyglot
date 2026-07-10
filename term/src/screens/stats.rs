//! The statistics screen: XP, streak, words learned, and kana totals.
//!
//! Port of the Go `internal/screens/stats`. Read-only: it loads aggregate
//! progress once at build time.

use polyglot_core::content::Course;
use polyglot_core::i18n::Messages;
use polyglot_core::model::KanaType;
use polyglot_core::storage::SqliteStore;
use ratatui::crossterm::event::{KeyCode, KeyModifiers};
use ratatui::layout::Rect;
use ratatui::text::Line;
use ratatui::widgets::Paragraph;
use ratatui::Frame;

use crate::app::Transition;
use crate::theme::Theme;

pub struct Stats {
    xp: i64,
    streak: i64,
    best: i64,
    learned: i64,
    total: i64,
    hira: i64,
    kata: i64,
    error: Option<String>,
}

impl Stats {
    /// Builds the stats screen, reading aggregate progress from storage.
    pub fn new(store: &SqliteStore, course: &Course, profile_id: Option<i64>) -> Stats {
        let total = (course.lessons.iter().map(|l| l.cards.len()).sum::<usize>()
            + course.kana.len()) as i64;
        let (mut hira, mut kata) = (0i64, 0i64);
        for k in &course.kana {
            match k.kana_type {
                KanaType::Hiragana => hira += 1,
                KanaType::Katakana => kata += 1,
            }
        }

        let base = Stats {
            xp: 0,
            streak: 0,
            best: 0,
            learned: 0,
            total,
            hira,
            kata,
            error: None,
        };

        let Some(pid) = profile_id else {
            return base;
        };
        match (store.get_stats(pid), store.count_learned_cards(pid)) {
            (Ok(s), Ok(learned)) => Stats {
                xp: s.xp,
                streak: s.streak,
                best: s.best_streak,
                learned,
                ..base
            },
            (Err(e), _) | (_, Err(e)) => Stats {
                error: Some(e.to_string()),
                ..base
            },
        }
    }

    pub fn handle(&mut self, code: KeyCode, mods: KeyModifiers) -> Transition {
        match code {
            KeyCode::Char('c') if mods.contains(KeyModifiers::CONTROL) => Transition::Quit,
            KeyCode::Esc | KeyCode::Char('q') | KeyCode::Enter => Transition::Pop,
            _ => Transition::Stay,
        }
    }

    pub fn render(&self, f: &mut Frame, inner: Rect, theme: &Theme, msgs: &Messages) {
        let mut lines: Vec<Line> = vec![
            Line::styled(msgs.stats_title.clone(), theme.title),
            Line::raw(""),
        ];
        if let Some(err) = &self.error {
            lines.push(Line::styled(err.clone(), theme.error));
        } else {
            lines.push(Line::styled(
                format!("★ {}: {}", msgs.xp_label, self.xp),
                theme.normal,
            ));
            lines.push(Line::styled(
                format!("{}/{} {}", self.learned, self.total, msgs.learned_suffix),
                theme.normal,
            ));
            lines.push(Line::styled(
                format!(
                    "▲ {}: {} {}  ({}: {})",
                    msgs.streak_label, self.streak, msgs.days_suffix, msgs.best_label, self.best
                ),
                theme.normal,
            ));
            lines.push(Line::styled(
                format!(
                    "{}: {}   {}: {}",
                    msgs.hiragana_label, self.hira, msgs.katakana_label, self.kata
                ),
                theme.normal,
            ));
        }

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
            Paragraph::new(Line::styled(msgs.back_help.clone(), theme.help)),
            help_area,
        );
    }
}
