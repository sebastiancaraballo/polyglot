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
use ratatui::widgets::Paragraph;
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

        f.render_widget(Paragraph::new(lines), inner);
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
