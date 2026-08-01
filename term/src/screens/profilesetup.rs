//! The profile creation flow: asks for a name, creates the profile, activates
//! it, and reloads the root.
//!
//! Port of the Go `internal/screens/profilesetup`. On first run this is shown
//! over the (empty) menu and cannot be escaped; from the profile switcher it is
//! a pushable "create new" flow.

use polyglot_core::i18n::Messages;
use polyglot_core::model::{self, NameError, MAX_NAME_LEN};
use ratatui::crossterm::event::{KeyCode, KeyModifiers};
use ratatui::layout::Rect;
use ratatui::text::Line;
use ratatui::widgets::{Paragraph, Wrap};
use ratatui::Frame;

use crate::app::{Ctx, Transition};
use crate::textfmt;
use crate::theme::Theme;

pub struct ProfileSetup {
    name: String,
    submitted: bool,
    tutorial: bool,
    error: bool,
}

impl ProfileSetup {
    pub fn new(tutorial: bool) -> ProfileSetup {
        ProfileSetup {
            name: String::new(),
            submitted: false,
            tutorial,
            error: false,
        }
    }

    pub fn handle(&mut self, code: KeyCode, mods: KeyModifiers, ctx: &Ctx<'_>) -> Transition {
        match code {
            KeyCode::Char('c') if mods.contains(KeyModifiers::CONTROL) => return Transition::Quit,
            KeyCode::Esc => {
                if !self.tutorial {
                    return Transition::Pop;
                }
            }
            KeyCode::Enter => return self.create_profile(ctx),
            KeyCode::Backspace => {
                self.name.pop();
                self.error = false;
            }
            KeyCode::Char(c) if !c.is_control() => {
                self.name.push(c);
                self.error = false;
            }
            _ => {}
        }
        Transition::Stay
    }

    fn create_profile(&mut self, ctx: &Ctx<'_>) -> Transition {
        self.submitted = true;
        let Ok(name) = model::normalize_name(&self.name) else {
            return Transition::Stay; // validation message shows why
        };
        self.name = name;

        match ctx.store.create_profile(&self.name) {
            Ok(p) => {
                let _ = ctx.store.set_active_profile_id(p.id);
                // First run runs the tutorial; the switcher's "create" does not.
                if self.tutorial {
                    Transition::StartOnboarding
                } else {
                    Transition::ReloadRoot
                }
            }
            Err(_) => {
                self.error = true;
                Transition::Stay
            }
        }
    }

    pub fn render(&self, f: &mut Frame, inner: Rect, theme: &Theme, msgs: &Messages) {
        let mut lines = vec![
            Line::styled(msgs.profile_name_title.clone(), theme.title),
            Line::raw(""),
            Line::styled(msgs.profile_name_prompt.clone(), theme.normal),
            Line::raw(""),
        ];

        if self.name.is_empty() {
            lines.push(Line::styled(
                format!("> {}", msgs.profile_name_placeholder),
                theme.subtle,
            ));
        } else {
            lines.push(Line::styled(format!("> {}", self.name), theme.normal));
        }

        if let Some(text) = self.validation_text(msgs) {
            lines.push(Line::raw(""));
            lines.push(Line::styled(text, theme.error));
        }
        if self.error {
            lines.push(Line::raw(""));
            lines.push(Line::styled(msgs.profile_create_error.clone(), theme.error));
        }

        lines.push(Line::raw(""));
        let help = if self.tutorial {
            msgs.profile_name_help_first.clone()
        } else {
            msgs.profile_name_help_cancel.clone()
        };
        lines.push(Line::styled(help, theme.help));

        f.render_widget(Paragraph::new(lines).wrap(Wrap { trim: false }), inner);
    }

    fn validation_text(&self, msgs: &Messages) -> Option<String> {
        if !self.submitted && self.name.trim().is_empty() {
            return None;
        }
        match model::normalize_name(&self.name) {
            Ok(_) => None,
            Err(NameError::Empty) => Some(msgs.profile_name_empty.clone()),
            Err(NameError::TooLong) => Some(textfmt::d(
                &msgs.profile_name_too_long_fmt,
                MAX_NAME_LEN as i64,
            )),
            Err(NameError::Invalid) => Some(msgs.profile_name_invalid.clone()),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use polyglot_core::storage::SqliteStore;

    fn empty_store() -> SqliteStore {
        SqliteStore::open_in_memory().unwrap()
    }

    fn ctx(store: &SqliteStore) -> Ctx<'_> {
        Ctx {
            store,
            profile_id: None,
        }
    }

    fn type_name(s: &mut ProfileSetup, name: &str, ctx: &Ctx<'_>) {
        for c in name.chars() {
            s.handle(KeyCode::Char(c), KeyModifiers::NONE, ctx);
        }
    }

    /// Submitting an invalid name stays on the step and creates nothing.
    #[test]
    fn submitting_an_invalid_name_stays_on_the_step() {
        let store = empty_store();
        let c = ctx(&store);
        let mut s = ProfileSetup::new(false);

        // An empty name is invalid.
        let t = s.handle(KeyCode::Enter, KeyModifiers::NONE, &c);
        assert!(matches!(t, Transition::Stay), "an empty name is rejected");
        assert!(s.submitted, "submitting marks the field as validated");
        assert!(
            store.list_profiles().unwrap().is_empty(),
            "no profile was created"
        );

        // Blank space is not a name either.
        type_name(&mut s, "   ", &c);
        let t = s.handle(KeyCode::Enter, KeyModifiers::NONE, &c);
        assert!(matches!(t, Transition::Stay), "a blank name is rejected");
        assert!(store.list_profiles().unwrap().is_empty());
    }

    /// A valid name creates the profile, makes it active, and — on first run —
    /// leads into the tutorial.
    #[test]
    fn a_valid_name_creates_the_profile() {
        let store = empty_store();
        let c = ctx(&store);
        let mut s = ProfileSetup::new(true); // first run
        type_name(&mut s, "Yui", &c);
        let t = s.handle(KeyCode::Enter, KeyModifiers::NONE, &c);

        assert!(
            matches!(t, Transition::StartOnboarding),
            "first run runs the tutorial, got {t:?}"
        );
        let profiles = store.list_profiles().unwrap();
        assert_eq!(profiles.len(), 1);
        assert_eq!(profiles[0].name, "Yui");
        assert_eq!(
            store.active_profile_id().unwrap(),
            Some(profiles[0].id),
            "the new profile became active"
        );
    }

    /// Creating from the profile switcher skips the tutorial.
    #[test]
    fn creating_from_the_switcher_skips_the_tutorial() {
        let store = empty_store();
        let c = ctx(&store);
        let mut s = ProfileSetup::new(false);
        type_name(&mut s, "Mei", &c);
        let t = s.handle(KeyCode::Enter, KeyModifiers::NONE, &c);

        assert!(matches!(t, Transition::ReloadRoot), "got {t:?}");
        assert_eq!(store.list_profiles().unwrap().len(), 1);
    }

    /// Backspace edits the name being typed.
    #[test]
    fn backspace_edits_the_name() {
        let store = empty_store();
        let c = ctx(&store);
        let mut s = ProfileSetup::new(false);
        type_name(&mut s, "Yuix", &c);
        s.handle(KeyCode::Backspace, KeyModifiers::NONE, &c);
        assert_eq!(s.name, "Yui");
    }
}
