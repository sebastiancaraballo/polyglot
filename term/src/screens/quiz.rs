//! The multiple-choice vocabulary quiz screen.
//!
//! Port of the Go `internal/screens/quiz`. Asks how to say a Spanish word in
//! Japanese, four options, and records each answer as a spaced-repetition grade.

use std::collections::HashMap;

use chrono::Utc;
use polyglot_core::i18n::Messages;
use polyglot_core::model::Card;
use polyglot_core::srs::{self, Grade};
use polyglot_core::storage::SqliteStore;
use polyglot_core::study;
use rand::rngs::StdRng;
use rand::seq::SliceRandom;
use rand::SeedableRng;
use ratatui::crossterm::event::{KeyCode, KeyModifiers};
use ratatui::layout::Rect;
use ratatui::text::Line;
use ratatui::widgets::{Paragraph, Wrap};
use ratatui::Frame;

use crate::app::{Ctx, Transition};
use crate::textfmt;
use crate::theme::Theme;

const OPTION_COUNT: usize = 4;
const QUESTION_LIMIT: usize = 10;

pub struct Quiz {
    cards: Vec<Card>,
    rng: StdRng,
    pool: Vec<String>,
    romaji: HashMap<String, String>,
    show_romaji: bool,

    deck: Vec<Card>,
    index: usize,
    options: Vec<String>,
    correct: usize,
    selected: usize,
    answered: bool,
    score: usize,
    wrong: Vec<String>,
    streak_applied: bool,
    error: Option<String>,
}

impl Quiz {
    pub fn new(cards: Vec<Card>, show_romaji: bool) -> Quiz {
        let pool: Vec<String> = cards.iter().map(|c| c.jp.clone()).collect();
        let romaji: HashMap<String, String> = cards
            .iter()
            .map(|c| (c.jp.clone(), c.romaji.clone()))
            .collect();
        let mut quiz = Quiz {
            cards,
            rng: StdRng::from_entropy(),
            pool,
            romaji,
            show_romaji,
            deck: Vec::new(),
            index: 0,
            options: Vec::new(),
            correct: 0,
            selected: 0,
            answered: false,
            score: 0,
            wrong: Vec::new(),
            streak_applied: false,
            error: None,
        };
        quiz.restart();
        quiz
    }

    fn restart(&mut self) {
        let mut deck = self.cards.clone();
        deck.shuffle(&mut self.rng);
        deck.truncate(QUESTION_LIMIT);
        self.deck = deck;
        self.index = 0;
        self.score = 0;
        self.wrong.clear();
        self.streak_applied = false;
        self.set_question();
    }

    fn set_question(&mut self) {
        if self.index >= self.deck.len() {
            return;
        }
        let (opts, correct) = study::options(
            &mut self.rng,
            &self.deck[self.index].jp,
            &self.pool,
            OPTION_COUNT,
        );
        self.options = opts;
        self.correct = correct;
        self.selected = 0;
        self.answered = false;
    }

    fn finished(&self) -> bool {
        self.index >= self.deck.len()
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
            if is_confirm(code) {
                self.restart();
            }
        } else if self.answered {
            if is_confirm(code) {
                self.index += 1;
                if !self.finished() {
                    self.set_question();
                }
            }
        } else {
            self.answer_key(code, ctx);
        }
        Transition::Stay
    }

    fn answer_key(&mut self, code: KeyCode, ctx: &Ctx<'_>) {
        match code {
            KeyCode::Up | KeyCode::Char('k') => self.selected = self.selected.saturating_sub(1),
            KeyCode::Down | KeyCode::Char('j') => {
                if self.selected + 1 < self.options.len() {
                    self.selected += 1;
                }
            }
            KeyCode::Char(c @ '1'..='4') => {
                let i = (c as u8 - b'1') as usize;
                if i < self.options.len() {
                    self.selected = i;
                    self.reveal(ctx);
                }
            }
            KeyCode::Enter | KeyCode::Char(' ') => self.reveal(ctx),
            _ => {}
        }
    }

    fn reveal(&mut self, ctx: &Ctx<'_>) {
        if self.answered {
            return;
        }
        self.answered = true;
        let card = self.deck[self.index].clone();
        let correct = self.selected == self.correct;
        if correct {
            self.score += 1;
        } else {
            self.wrong.push(card.source.clone());
        }
        if let Some(pid) = ctx.profile_id {
            if let Err(e) = self.persist(ctx.store, pid, &card, correct) {
                self.error = Some(e);
            }
        }
    }

    fn persist(
        &mut self,
        store: &SqliteStore,
        pid: i64,
        card: &Card,
        correct: bool,
    ) -> Result<(), String> {
        let state = store
            .get_card_state(pid, &card.id)
            .unwrap_or_else(|_| srs::new_card(&card.id));
        let grade = if correct { Grade::Good } else { Grade::Again };
        let now = Utc::now();
        let state = srs::review(&state, grade, now);
        store
            .save_card_state(pid, &state)
            .map_err(|e| e.to_string())?;
        store
            .add_xp(pid, study::xp_for_answer(correct))
            .map_err(|e| e.to_string())?;
        if !self.streak_applied {
            let stats = store.get_stats(pid).map_err(|e| e.to_string())?;
            store
                .save_stats(pid, &study::update_streak(stats, now))
                .map_err(|e| e.to_string())?;
            self.streak_applied = true;
        }
        Ok(())
    }

    pub fn render(&self, f: &mut Frame, inner: Rect, theme: &Theme, msgs: &Messages) {
        let lines = if self.finished() {
            self.summary_lines(theme, msgs)
        } else {
            self.question_lines(theme, msgs)
        };
        f.render_widget(Paragraph::new(lines).wrap(Wrap { trim: false }), inner);
    }

    fn question_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let mut lines = vec![
            Line::styled(
                format!(
                    "{}  {}/{}",
                    msgs.quiz_title,
                    self.index + 1,
                    self.deck.len()
                ),
                theme.title,
            ),
            Line::raw(""),
            Line::styled(
                textfmt::s(&msgs.quiz_question_fmt, &self.deck[self.index].source),
                theme.normal,
            ),
            Line::raw(""),
        ];

        for (i, opt) in self.options.iter().enumerate() {
            let label = if self.show_romaji {
                match self.romaji.get(opt) {
                    Some(r) if !r.is_empty() => format!("{opt} ({r})"),
                    _ => opt.clone(),
                }
            } else {
                opt.clone()
            };
            let (mark, style) = if self.answered && i == self.correct {
                ("✓", theme.success)
            } else if self.answered && i == self.selected {
                ("✗", theme.error)
            } else if i == self.selected {
                ("▸", theme.selected)
            } else {
                (" ", theme.normal)
            };
            lines.push(Line::styled(format!("{mark} {}) {label}", i + 1), style));
        }

        lines.push(Line::raw(""));
        let answered_offset = if self.answered { 1 } else { 0 };
        lines.push(Line::styled(
            format!(
                "{}: {}/{}",
                msgs.score_label,
                self.score,
                self.index + answered_offset
            ),
            theme.normal,
        ));
        let help = if self.answered {
            msgs.continue_help.clone()
        } else {
            msgs.choice_help.clone()
        };
        lines.push(Line::styled(help, theme.help));
        lines
    }

    fn summary_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let mut lines = vec![
            Line::styled(msgs.session_done.clone(), theme.title),
            Line::raw(""),
            Line::styled(
                format!("{}: {}/{}", msgs.score_label, self.score, self.deck.len()),
                theme.normal,
            ),
        ];
        if !self.wrong.is_empty() {
            lines.push(Line::styled(
                format!("{}: {}", msgs.review_label, self.wrong.join(", ")),
                theme.normal,
            ));
        }
        lines.push(Line::raw(""));
        lines.push(Line::styled(msgs.restart_help.clone(), theme.help));
        lines
    }
}

fn is_confirm(code: KeyCode) -> bool {
    matches!(code, KeyCode::Enter | KeyCode::Char(' '))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::app::Ctx;
    use polyglot_core::storage::SqliteStore;

    fn card(jp: &str) -> Card {
        Card {
            id: jp.to_string(),
            source: format!("gloss-{jp}"),
            jp: jp.to_string(),
            romaji: String::new(),
            notes: String::new(),
            jlpt: None,
            functions: Vec::new(),
            freq: 0,
        }
    }

    #[test]
    fn answering_correctly_scores_and_advances() {
        let store = SqliteStore::open_in_memory().unwrap();
        let pid = store.create_profile("A").unwrap().id;
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let cards = ["みず", "ひ", "て", "き", "こ"].map(card).to_vec();
        let mut q = Quiz::new(cards, false);
        assert!(!q.finished());

        // Press the key of the correct option; it reveals and scores.
        let key = (b'1' + q.correct as u8) as char;
        q.handle(KeyCode::Char(key), KeyModifiers::NONE, &ctx);
        assert!(q.answered);
        assert_eq!(q.score, 1);

        // Confirm advances to the next question.
        q.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert_eq!(q.index, 1);
        assert!(!q.answered);
        // The graded card was persisted as a review.
        assert_eq!(store.count_learned_cards(pid).unwrap(), 1);
    }
}
