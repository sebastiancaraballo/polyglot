//! The profile switcher: lists known profiles plus a "create new" row.
//!
//! Port of the Go `internal/screens/profiles`. Switching sets the active profile
//! and reloads the root; creating navigates to the profile-setup flow.

use polyglot_core::i18n::Messages;
use polyglot_core::model::Profile;
use polyglot_core::storage::SqliteStore;
use ratatui::crossterm::event::{KeyCode, KeyModifiers};
use ratatui::layout::Rect;
use ratatui::text::Line;
use ratatui::widgets::{Paragraph, Wrap};
use ratatui::Frame;

use crate::app::{Ctx, Dest, Transition};
use crate::theme::Theme;

pub struct Profiles {
    profiles: Vec<Profile>,
    active_id: Option<i64>,
    cursor: usize,
    error: Option<String>,
}

impl Profiles {
    pub fn new(store: &SqliteStore, active_id: Option<i64>) -> Profiles {
        let (profiles, error) = match store.list_profiles() {
            Ok(p) => (p, None),
            Err(e) => (Vec::new(), Some(e.to_string())),
        };
        Profiles {
            profiles,
            active_id,
            cursor: 0,
            error,
        }
    }

    /// The last cursor index: the "create new profile" row sits after the list.
    fn last_cursor(&self) -> usize {
        self.profiles.len()
    }

    pub fn handle(&mut self, code: KeyCode, mods: KeyModifiers, ctx: &Ctx<'_>) -> Transition {
        match code {
            KeyCode::Char('c') if mods.contains(KeyModifiers::CONTROL) => return Transition::Quit,
            KeyCode::Esc => return Transition::Pop,
            KeyCode::Up | KeyCode::Char('k') => self.cursor = self.cursor.saturating_sub(1),
            KeyCode::Down | KeyCode::Char('j') => {
                if self.cursor < self.last_cursor() {
                    self.cursor += 1;
                }
            }
            KeyCode::Enter | KeyCode::Char(' ') => return self.choose(ctx),
            _ => {}
        }
        Transition::Stay
    }

    fn choose(&mut self, ctx: &Ctx<'_>) -> Transition {
        if self.cursor == self.profiles.len() {
            return Transition::Push(Dest::ProfileSetup);
        }
        let id = self.profiles[self.cursor].id;
        match ctx.store.set_active_profile_id(id) {
            Ok(()) => Transition::ReloadRoot,
            Err(e) => {
                self.error = Some(e.to_string());
                Transition::Stay
            }
        }
    }

    pub fn render(&self, f: &mut Frame, inner: Rect, theme: &Theme, msgs: &Messages) {
        let mut lines = vec![
            Line::styled(msgs.profiles_title.clone(), theme.title),
            Line::raw(""),
        ];

        if let Some(err) = &self.error {
            lines.push(Line::styled(err.clone(), theme.error));
            lines.push(Line::raw(""));
        } else if self.profiles.is_empty() {
            lines.push(Line::styled(msgs.no_profiles.clone(), theme.subtle));
        } else {
            for (i, p) in self.profiles.iter().enumerate() {
                let line = self.profile_line(p, msgs);
                lines.push(if i == self.cursor {
                    Line::styled(format!("▸ {line}"), theme.selected)
                } else {
                    Line::styled(format!("  {line}"), theme.normal)
                });
            }
        }

        let create_selected = self.cursor == self.profiles.len();
        lines.push(if create_selected {
            Line::styled(format!("▸ {}", msgs.profile_create_new), theme.selected)
        } else {
            Line::styled(format!("  {}", msgs.profile_create_new), theme.normal)
        });
        lines.push(Line::raw(""));
        lines.push(Line::styled(msgs.profiles_help.clone(), theme.help));

        f.render_widget(Paragraph::new(lines).wrap(Wrap { trim: false }), inner);
    }

    fn profile_line(&self, p: &Profile, msgs: &Messages) -> String {
        if Some(p.id) == self.active_id {
            format!("{}  ● {}", p.name, msgs.active_profile_label)
        } else {
            p.name.clone()
        }
    }
}
