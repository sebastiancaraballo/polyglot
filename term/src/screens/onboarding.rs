//! First-run onboarding: teaches the controls with a guided sample exercise and
//! marks the profile as onboarded on completion.
//!
//! Port of the Go `internal/screens/onboarding`.

use polyglot_core::i18n::Messages;
use polyglot_core::storage::SqliteStore;
use polyglot_core::study;
use ratatui::crossterm::event::{KeyCode, KeyModifiers};
use ratatui::layout::Rect;
use ratatui::text::Line;
use ratatui::widgets::{Paragraph, Wrap};
use ratatui::Frame;

use crate::app::{Ctx, Transition};
use crate::theme::Theme;

#[derive(Debug, PartialEq, Eq)]
enum Step {
    Welcome,
    Exercise,
    Done,
}

pub struct Onboarding {
    profile_id: Option<i64>,
    step: Step,
    selected: usize,
    answered: bool,
    correct: bool,
}

impl Onboarding {
    pub fn new(profile_id: Option<i64>) -> Onboarding {
        Onboarding {
            profile_id,
            step: Step::Welcome,
            selected: 0,
            answered: false,
            correct: false,
        }
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
        match self.step {
            Step::Welcome => {
                if is_confirm(code) {
                    self.step = Step::Exercise;
                }
            }
            Step::Exercise => self.handle_exercise(code, ctx.store),
            Step::Done => {
                if is_confirm(code) {
                    return self.finish(ctx);
                }
            }
        }
        Transition::Stay
    }

    fn handle_exercise(&mut self, code: KeyCode, store: &SqliteStore) {
        let _ = store;
        let options = option_count();
        // Once answered correctly, the next confirm advances to the final step.
        if self.answered && self.correct {
            if is_confirm(code) {
                self.step = Step::Done;
            }
            return;
        }
        match code {
            KeyCode::Up | KeyCode::Char('k') => self.selected = self.selected.saturating_sub(1),
            KeyCode::Down | KeyCode::Char('j') if self.selected + 1 < options => self.selected += 1,
            KeyCode::Char(c @ '1'..='4') => {
                let i = (c as u8 - b'1') as usize;
                if i < options {
                    self.selected = i;
                    self.answer();
                }
            }
            KeyCode::Enter | KeyCode::Char(' ') => self.answer(),
            _ => {}
        }
    }

    fn answer(&mut self) {
        self.answered = true;
        self.correct = self.selected == sample_correct();
    }

    fn finish(&mut self, ctx: &Ctx<'_>) -> Transition {
        if let Some(pid) = self.profile_id {
            let _ = ctx.store.set_onboarded(pid);
            let _ = ctx.store.add_xp(pid, study::ONBOARDING_XP);
        }
        Transition::Pop
    }

    pub fn render(&self, f: &mut Frame, inner: Rect, theme: &Theme, msgs: &Messages) {
        let lines = match self.step {
            Step::Exercise => self.exercise_lines(theme, msgs),
            Step::Done => self.done_lines(theme, msgs),
            Step::Welcome => self.welcome_lines(theme, msgs),
        };
        f.render_widget(Paragraph::new(lines).wrap(Wrap { trim: false }), inner);
    }

    fn welcome_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let mut lines = vec![
            Line::styled(msgs.welcome_title.clone(), theme.title),
            Line::raw(""),
            Line::styled(msgs.welcome_intro.clone(), theme.normal),
            Line::raw(""),
            Line::styled(msgs.controls_title.clone(), theme.accent),
        ];
        for k in &msgs.controls_keys {
            lines.push(Line::styled(format!("  {k}"), theme.normal));
        }
        lines.push(Line::raw(""));
        lines.push(Line::styled(msgs.welcome_next.clone(), theme.help));
        lines
    }

    fn exercise_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let mut lines = vec![
            Line::styled(msgs.practice_title.clone(), theme.title),
            Line::raw(""),
            Line::styled(
                format!("{}  ({})", msgs.sample_word, msgs.sample_romaji),
                theme.accent,
            ),
            Line::styled(msgs.sample_prompt.clone(), theme.normal),
            Line::raw(""),
        ];

        let correct = sample_correct();
        for (i, opt) in msgs.sample_options.iter().enumerate() {
            let line = format!(" {}) {opt}", i + 1);
            if i == correct && (self.answered || i == self.selected) {
                lines.push(Line::styled(
                    format!("✓{line}  {}", msgs.sample_hint),
                    theme.success,
                ));
            } else if i == self.selected {
                lines.push(Line::styled(format!("▸{line}"), theme.selected));
            } else {
                lines.push(Line::styled(format!(" {line}"), theme.normal));
            }
        }
        lines.push(Line::raw(""));
        if self.answered && self.correct {
            lines.push(Line::styled(msgs.practice_correct.clone(), theme.success));
            lines.push(Line::styled(msgs.practice_next.clone(), theme.help));
        } else if self.answered {
            lines.push(Line::styled(msgs.practice_retry.clone(), theme.error));
        } else {
            lines.push(Line::styled(msgs.choice_help.clone(), theme.help));
        }
        lines
    }

    fn done_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        vec![
            Line::styled(msgs.done_title.clone(), theme.title),
            Line::raw(""),
            Line::styled(msgs.done_recommend.clone(), theme.normal),
            Line::raw(""),
            Line::styled(msgs.done_next.clone(), theme.help),
        ]
    }
}

fn sample_correct() -> usize {
    polyglot_core::i18n::default().sample_correct as usize
}

fn option_count() -> usize {
    polyglot_core::i18n::default().sample_options.len()
}

fn is_confirm(code: KeyCode) -> bool {
    matches!(code, KeyCode::Enter | KeyCode::Char(' '))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn store_with_profile() -> (SqliteStore, i64) {
        let s = SqliteStore::open_in_memory().unwrap();
        let pid = s.create_profile("Yui").unwrap().id;
        (s, pid)
    }

    /// The happy path: welcome → exercise (answered correctly) → done, which
    /// marks the profile onboarded and awards the welcome XP.
    #[test]
    fn the_flow_completes_and_persists() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut o = Onboarding::new(Some(pid));
        assert_eq!(o.step, Step::Welcome);

        o.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert_eq!(o.step, Step::Exercise);

        // Answer correctly, then confirm through to the end.
        o.selected = sample_correct();
        o.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert!(o.answered && o.correct);
        o.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert_eq!(o.step, Step::Done);

        let t = o.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert!(matches!(t, Transition::Pop), "finishing leaves onboarding");
        assert!(
            store.get_profile(pid).unwrap().onboarded,
            "the profile is marked onboarded"
        );
        assert_eq!(
            store.get_stats(pid).unwrap().xp,
            study::ONBOARDING_XP,
            "the welcome XP was awarded"
        );
    }

    /// A wrong answer keeps the learner on the exercise: the first success has
    /// to be a real one.
    #[test]
    fn a_wrong_answer_stays_on_the_exercise() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut o = Onboarding::new(Some(pid));
        o.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // -> exercise

        o.selected = (sample_correct() + 1) % option_count();
        o.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // answer wrongly
        assert!(o.answered && !o.correct);
        o.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert_eq!(o.step, Step::Exercise, "a wrong answer does not advance");

        // Correcting it does advance.
        o.selected = sample_correct();
        o.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        o.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert_eq!(o.step, Step::Done);
    }

    /// Space works everywhere enter does: advancing and answering.
    #[test]
    fn space_advances_and_answers() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut o = Onboarding::new(Some(pid));

        o.handle(KeyCode::Char(' '), KeyModifiers::NONE, &ctx);
        assert_eq!(o.step, Step::Exercise, "space leaves the welcome step");

        o.selected = sample_correct();
        o.handle(KeyCode::Char(' '), KeyModifiers::NONE, &ctx);
        assert!(o.answered, "space answers the exercise");
    }

    /// The number keys pick an option directly.
    #[test]
    fn number_keys_pick_an_option() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut o = Onboarding::new(Some(pid));
        o.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // -> exercise

        let key = (b'1' + sample_correct() as u8) as char;
        o.handle(KeyCode::Char(key), KeyModifiers::NONE, &ctx);
        assert!(o.answered && o.correct, "the number key answered directly");
    }
}
