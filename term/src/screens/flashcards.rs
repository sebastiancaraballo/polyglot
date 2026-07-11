//! The flashcard-style review screen (also the cross-curriculum "Repaso").
//!
//! Port of the Go `internal/screens/flashcards`. Presents the items currently
//! due — built by the cross-curriculum review queue — one at a time: prompt,
//! then the revealed answer, then a spaced-repetition grade.

use chrono::Utc;
use polyglot_core::i18n::Messages;
use polyglot_core::review::{self, Scheduled};
use polyglot_core::srs::{self, Grade};
use polyglot_core::storage::SqliteStore;
use polyglot_core::study;
use ratatui::crossterm::event::{KeyCode, KeyModifiers};
use ratatui::layout::Rect;
use ratatui::text::Line;
use ratatui::widgets::{Paragraph, Wrap};
use ratatui::Frame;

use crate::app::{Ctx, Transition};
use crate::textfmt;
use crate::theme::Theme;

const SESSION_LIMIT: i64 = 20;

pub struct Flashcards {
    queue: Vec<Scheduled>,
    held_back_new: usize,
    index: usize,
    revealed: bool,
    reviewed: usize,
    streak_applied: bool,
    title: String,
    show_romaji: bool,
    error: Option<String>,
}

impl Flashcards {
    /// Builds a review session containing the items currently due.
    pub fn new(
        store: &SqliteStore,
        profile_id: Option<i64>,
        items: &[review::Item],
        title: String,
        show_romaji: bool,
    ) -> Flashcards {
        let mut queue = Vec::new();
        let mut held_back_new = 0;
        let mut error = None;
        if let Some(pid) = profile_id {
            match review::build_queue(store, pid, items, Utc::now(), SESSION_LIMIT) {
                Ok(q) => {
                    queue = q.items;
                    held_back_new = q.held_back_new;
                }
                Err(e) => error = Some(e.to_string()),
            }
        }
        Flashcards {
            queue,
            held_back_new,
            index: 0,
            revealed: false,
            reviewed: 0,
            streak_applied: false,
            title,
            show_romaji,
            error,
        }
    }

    fn finished(&self) -> bool {
        self.index >= self.queue.len()
    }

    pub fn handle(&mut self, code: KeyCode, mods: KeyModifiers, ctx: &Ctx<'_>) -> Transition {
        if let KeyCode::Char('c') = code {
            if mods.contains(KeyModifiers::CONTROL) {
                return Transition::Quit;
            }
        }
        if code == KeyCode::Esc {
            return Transition::Pop;
        }
        if self.finished() {
            return Transition::Stay;
        }
        if !self.revealed {
            if matches!(code, KeyCode::Enter | KeyCode::Char(' ')) {
                self.revealed = true;
            }
            return Transition::Stay;
        }
        let grade = match code {
            KeyCode::Char('1') => Some(Grade::Again),
            KeyCode::Char('2') => Some(Grade::Hard),
            KeyCode::Char('3') => Some(Grade::Good),
            KeyCode::Char('4') => Some(Grade::Easy),
            _ => None,
        };
        if let Some(grade) = grade {
            self.grade(grade, ctx);
        }
        Transition::Stay
    }

    fn grade(&mut self, grade: Grade, ctx: &Ctx<'_>) {
        let sched = &self.queue[self.index];
        let state = srs::review(&sched.state, grade, Utc::now());
        if let Some(pid) = ctx.profile_id {
            if let Err(e) = self.persist(ctx.store, pid, &state, grade) {
                self.error = Some(e);
            }
        }
        self.reviewed += 1;
        self.index += 1;
        self.revealed = false;
    }

    fn persist(
        &mut self,
        store: &SqliteStore,
        pid: i64,
        state: &polyglot_core::model::CardState,
        grade: Grade,
    ) -> Result<(), String> {
        store
            .save_card_state(pid, state)
            .map_err(|e| e.to_string())?;
        store
            .add_xp(pid, study::xp_for_grade(grade))
            .map_err(|e| e.to_string())?;
        if !self.streak_applied {
            let stats = store.get_stats(pid).map_err(|e| e.to_string())?;
            store
                .save_stats(pid, &study::update_streak(stats, Utc::now()))
                .map_err(|e| e.to_string())?;
            self.streak_applied = true;
        }
        Ok(())
    }

    pub fn render(&self, f: &mut Frame, inner: Rect, theme: &Theme, msgs: &Messages) {
        let lines = if self.queue.is_empty() {
            vec![Line::styled(msgs.nothing_due.clone(), theme.title)]
        } else if self.finished() {
            self.summary_lines(theme, msgs)
        } else {
            self.card_lines(theme, msgs)
        };
        f.render_widget(Paragraph::new(lines).wrap(Wrap { trim: false }), inner);
    }

    fn card_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let item = &self.queue[self.index].item;
        let mut lines = vec![Line::styled(
            format!("{}  {}/{}", self.title, self.index + 1, self.queue.len()),
            theme.title,
        )];
        if self.held_back_new > 0 {
            lines.push(Line::styled(
                textfmt::d(&msgs.flash_new_held_fmt, self.held_back_new as i64),
                theme.subtle,
            ));
        }
        lines.push(Line::raw(""));
        lines.push(Line::styled(item.prompt.clone(), theme.accent));
        lines.push(Line::raw(""));

        if !self.revealed {
            lines.push(Line::styled(msgs.reveal_help.clone(), theme.help));
            return lines;
        }

        lines.push(Line::styled(item.answer.clone(), theme.title));
        if self.show_romaji && !item.secondary.is_empty() {
            lines.push(Line::styled(item.secondary.clone(), theme.subtle));
        }
        if !item.notes.is_empty() {
            lines.push(Line::styled(item.notes.clone(), theme.subtle));
        }
        if item.freq > 0 {
            lines.push(Line::styled(
                textfmt::d(&msgs.freq_rank_fmt, item.freq),
                theme.subtle,
            ));
        }
        lines.push(Line::raw(""));
        lines.push(Line::styled(msgs.grade_prompt.clone(), theme.normal));
        for line in self.grade_options(theme, msgs) {
            lines.push(line);
        }
        lines.push(Line::styled(msgs.back_help.clone(), theme.help));
        lines
    }

    fn grade_options<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let state = &self.queue[self.index].state;
        let now = Utc::now();
        [
            ("1", &msgs.grade_again, Grade::Again),
            ("2", &msgs.grade_hard, Grade::Hard),
            ("3", &msgs.grade_good, Grade::Good),
            ("4", &msgs.grade_easy, Grade::Easy),
        ]
        .into_iter()
        .map(|(key, label, grade)| {
            let days = srs::preview_interval(state, grade, now);
            Line::styled(
                format!("[{key}] {label} ({})", self.format_interval(days, msgs)),
                theme.normal,
            )
        })
        .collect()
    }

    fn format_interval(&self, days: i64, msgs: &Messages) -> String {
        if days <= 0 {
            msgs.today.clone()
        } else {
            format!("{days}{}", msgs.day_short)
        }
    }

    fn summary_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let mut lines = vec![
            Line::styled(msgs.session_done.clone(), theme.title),
            Line::raw(""),
            Line::styled(
                format!("{}: {}", msgs.reviewed_label, self.reviewed),
                theme.normal,
            ),
        ];
        if self.held_back_new > 0 {
            lines.push(Line::styled(
                textfmt::d(&msgs.flash_new_held_fmt, self.held_back_new as i64),
                theme.subtle,
            ));
        }
        lines.push(Line::raw(""));
        lines.push(Line::styled(msgs.back_help.clone(), theme.help));
        lines
    }
}
