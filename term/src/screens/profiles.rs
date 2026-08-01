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

#[cfg(test)]
mod tests {
    use super::*;

    fn store_with_two() -> (SqliteStore, i64, i64) {
        let s = SqliteStore::open_in_memory().unwrap();
        let a = s.create_profile("Yui").unwrap().id;
        let b = s.create_profile("Mei").unwrap().id;
        (s, a, b)
    }

    /// Choosing a profile makes it active and reloads the app root.
    #[test]
    fn choosing_a_profile_switches_to_it() {
        let (store, a, b) = store_with_two();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(a),
        };
        let mut p = Profiles::new(&store, Some(a));
        assert_eq!(p.profiles.len(), 2);

        p.handle(KeyCode::Down, KeyModifiers::NONE, &ctx); // move to the second
        let t = p.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);

        assert!(
            matches!(t, Transition::ReloadRoot),
            "switching profile reloads the root"
        );
        assert_eq!(
            store.active_profile_id().unwrap(),
            Some(b),
            "the chosen profile became active"
        );
    }

    /// The row after the list opens profile creation.
    #[test]
    fn choosing_create_opens_profile_setup() {
        let (store, a, _) = store_with_two();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(a),
        };
        let mut p = Profiles::new(&store, Some(a));
        for _ in 0..p.profiles.len() {
            p.handle(KeyCode::Down, KeyModifiers::NONE, &ctx);
        }
        let t = p.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert!(
            matches!(t, Transition::Push(Dest::ProfileSetup)),
            "got {t:?}"
        );
    }

    /// The active profile is marked with a symbol and a label, never by color
    /// alone.
    #[test]
    fn view_marks_the_active_profile() {
        let (store, a, _) = store_with_two();
        let msgs = polyglot_core::i18n::default();
        let p = Profiles::new(&store, Some(a));
        let view = crate::testutil::snapshot(|f, inner, theme| p.render(f, inner, theme, msgs));

        assert!(view.contains("Yui") && view.contains("Mei"), "lists both");
        assert!(view.contains('●'), "marks the active profile with a symbol");
        assert!(
            view.contains(&msgs.active_profile_label),
            "and names it in text"
        );
    }

    /// The cursor never leaves the list plus its trailing "create" row.
    #[test]
    fn cursor_is_clamped() {
        let (store, a, _) = store_with_two();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(a),
        };
        let mut p = Profiles::new(&store, Some(a));
        p.handle(KeyCode::Up, KeyModifiers::NONE, &ctx);
        assert_eq!(p.cursor, 0, "up at the top stays");

        for _ in 0..10 {
            p.handle(KeyCode::Down, KeyModifiers::NONE, &ctx);
        }
        assert_eq!(p.cursor, p.profiles.len(), "down stops on the create row");
    }
}
