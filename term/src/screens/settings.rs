//! The settings screen: the "show romaji" toggle plus destructive actions
//! (delete this profile, delete all data), each behind a Cancel-first confirm.
//!
//! Port of the Go `internal/screens/settings`. The Go screen emitted nav
//! messages for the router to carry out; here the writes go through the shared
//! store context directly, and destructive actions return `ReloadRoot`.

use polyglot_core::i18n::Messages;
use ratatui::crossterm::event::{KeyCode, KeyModifiers};
use ratatui::layout::Rect;
use ratatui::text::Line;
use ratatui::widgets::Paragraph;
use ratatui::Frame;

use crate::app::{Ctx, Transition};
use crate::theme::Theme;

#[derive(Clone, Copy)]
enum Action {
    DeleteProfile,
    WipeAll,
}

const ACTIONS: [Action; 2] = [Action::DeleteProfile, Action::WipeAll];

impl Action {
    fn label(self, msgs: &Messages) -> &str {
        match self {
            Action::DeleteProfile => &msgs.delete_profile,
            Action::WipeAll => &msgs.delete_all_data,
        }
    }
    fn warning(self, msgs: &Messages) -> &str {
        match self {
            Action::DeleteProfile => &msgs.delete_profile_warning,
            Action::WipeAll => &msgs.delete_all_warning,
        }
    }
    fn confirm(self, msgs: &Messages) -> &str {
        match self {
            Action::DeleteProfile => &msgs.confirm_delete_profile,
            Action::WipeAll => &msgs.confirm_delete,
        }
    }
}

pub struct Settings {
    show_romaji: bool,
    cursor: usize, // 0 = toggle, 1.. = actions
    confirming: bool,
    confirm_action: usize,
    confirm_yes: bool,
    error: Option<String>,
}

impl Settings {
    pub fn new(show_romaji: bool) -> Settings {
        Settings {
            show_romaji,
            cursor: 0,
            confirming: false,
            confirm_action: 0,
            confirm_yes: false,
            error: None,
        }
    }

    pub fn handle(&mut self, code: KeyCode, mods: KeyModifiers, ctx: &Ctx<'_>) -> Transition {
        if let KeyCode::Char('c') = code {
            if mods.contains(KeyModifiers::CONTROL) {
                return Transition::Quit;
            }
        }
        if self.confirming {
            self.handle_confirm(code, ctx)
        } else {
            self.handle_list(code, ctx)
        }
    }

    fn handle_list(&mut self, code: KeyCode, ctx: &Ctx<'_>) -> Transition {
        match code {
            KeyCode::Esc => return Transition::Pop,
            KeyCode::Up | KeyCode::Char('k') => self.cursor = self.cursor.saturating_sub(1),
            KeyCode::Down | KeyCode::Char('j') => {
                if self.cursor < ACTIONS.len() {
                    self.cursor += 1;
                }
            }
            KeyCode::Enter | KeyCode::Char(' ') => {
                if self.cursor == 0 {
                    self.show_romaji = !self.show_romaji;
                    if let Some(pid) = ctx.profile_id {
                        if let Err(e) = ctx.store.set_show_romaji(pid, self.show_romaji) {
                            self.error = Some(e.to_string());
                        }
                    }
                } else {
                    self.confirming = true;
                    self.confirm_action = self.cursor - 1;
                    self.confirm_yes = false; // default selection is "Cancel"
                }
            }
            _ => {}
        }
        Transition::Stay
    }

    fn handle_confirm(&mut self, code: KeyCode, ctx: &Ctx<'_>) -> Transition {
        match code {
            KeyCode::Esc => {
                self.confirming = false;
                Transition::Stay
            }
            KeyCode::Up
            | KeyCode::Down
            | KeyCode::Left
            | KeyCode::Right
            | KeyCode::Char('k' | 'j' | 'h' | 'l') => {
                self.confirm_yes = !self.confirm_yes;
                Transition::Stay
            }
            KeyCode::Enter | KeyCode::Char(' ') => {
                if self.confirm_yes {
                    self.perform(ACTIONS[self.confirm_action], ctx)
                } else {
                    self.confirming = false;
                    Transition::Stay
                }
            }
            _ => Transition::Stay,
        }
    }

    fn perform(&mut self, action: Action, ctx: &Ctx<'_>) -> Transition {
        let result: Result<(), String> = match action {
            Action::DeleteProfile => match ctx.profile_id {
                Some(pid) => ctx.store.delete_profile(pid).map_err(|e| e.to_string()),
                None => Ok(()),
            },
            Action::WipeAll => wipe_all(ctx),
        };
        match result {
            Ok(()) => Transition::ReloadRoot,
            Err(e) => {
                self.error = Some(e);
                self.confirming = false;
                Transition::Stay
            }
        }
    }

    pub fn render(&self, f: &mut Frame, inner: Rect, theme: &Theme, msgs: &Messages) {
        let lines = if self.confirming {
            self.confirm_lines(theme, msgs)
        } else {
            self.list_lines(theme, msgs)
        };
        f.render_widget(Paragraph::new(lines), inner);
    }

    fn list_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let mut lines = vec![
            Line::styled(msgs.settings_title.clone(), theme.title),
            Line::raw(""),
        ];
        lines.push(self.row(0, self.romaji_label(msgs), theme));
        for (i, a) in ACTIONS.iter().enumerate() {
            lines.push(self.row(i + 1, a.label(msgs).to_string(), theme));
        }
        lines.push(Line::raw(""));
        lines.push(Line::styled(msgs.settings_help.clone(), theme.help));
        lines
    }

    fn row<'a>(&self, index: usize, label: String, theme: &Theme) -> Line<'a> {
        if index == self.cursor {
            Line::styled(format!("▸ {label}"), theme.selected)
        } else {
            Line::styled(format!("  {label}"), theme.normal)
        }
    }

    fn romaji_label(&self, msgs: &Messages) -> String {
        let value = if self.show_romaji {
            &msgs.option_on
        } else {
            &msgs.option_off
        };
        format!("{}: {}", msgs.show_romaji_label, value)
    }

    fn confirm_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let action = ACTIONS[self.confirm_action];
        let mut lines = vec![
            Line::styled(action.label(msgs).to_string(), theme.title),
            Line::raw(""),
            Line::styled(action.warning(msgs).to_string(), theme.error),
            Line::raw(""),
        ];
        // Cancel (default) then the destructive confirm.
        let cancel_selected = !self.confirm_yes;
        lines.push(if cancel_selected {
            Line::styled(format!("▸ {}", msgs.cancel_label), theme.selected)
        } else {
            Line::styled(format!("  {}", msgs.cancel_label), theme.normal)
        });
        lines.push(if self.confirm_yes {
            Line::styled(format!("▸ {}", action.confirm(msgs)), theme.error)
        } else {
            Line::styled(format!("  {}", action.confirm(msgs)), theme.normal)
        });
        lines.push(Line::raw(""));
        lines.push(Line::styled(msgs.confirm_help.clone(), theme.help));
        lines
    }
}

/// Deletes every profile (cascading their progress), returning to a first-run
/// state.
fn wipe_all(ctx: &Ctx<'_>) -> Result<(), String> {
    let profiles = ctx.store.list_profiles().map_err(|e| e.to_string())?;
    for p in profiles {
        ctx.store.delete_profile(p.id).map_err(|e| e.to_string())?;
    }
    Ok(())
}
