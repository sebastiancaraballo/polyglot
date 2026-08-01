//! The kanji trainer: recognize a character's meaning, then meet its readings.
//!
//! Deliberately a screen of its own rather than a mode of the kana trainer. The
//! two look alike for one question — show a character, pick among four — but
//! diverge everywhere else: the kana trainer's picker is built around syllabary
//! and category groups gated by the hiragana→katakana rule, its answers are a
//! single romaji reading, and it times responses to record a best time. A kanji
//! has *several* readings, none of which is "the" answer, and it is grouped by
//! level, not by syllabary. Threading both through one screen would mean a mode
//! flag in every method; the shared parts (option building, mastery grading,
//! persistence) already live in `core` and are reused here.

use std::collections::HashMap;

use polyglot_core::content::Course;
use polyglot_core::i18n::Messages;
use polyglot_core::model::{KanjiItem, KanjiProgress};
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
use crate::theme::Theme;
use crate::tile::big_tile;

const OPTION_COUNT: usize = 4;

/// Kanji drawn per session. Short on purpose: a kanji carries more to remember
/// than a kana, so sessions stay well inside working-memory limits.
const SESSION_LIMIT: usize = 8;

pub struct KanjiTrainer {
    items: Vec<KanjiItem>,
    rng: StdRng,
    progress: HashMap<String, KanjiProgress>,

    deck: Vec<KanjiItem>,
    index: usize,
    options: Vec<String>,
    correct: usize,
    selected: usize,
    answered: bool,
    correct_count: usize,
    error: Option<String>,
}

impl KanjiTrainer {
    pub fn new(store: &SqliteStore, course: &Course, profile_id: Option<i64>) -> KanjiTrainer {
        let mut progress = HashMap::new();
        let mut error = None;
        if let Some(pid) = profile_id {
            match store.get_kanji_progress(pid) {
                Ok(p) => progress = p,
                Err(e) => error = Some(e.to_string()),
            }
        }
        let mut t = KanjiTrainer {
            items: course.kanji.clone(),
            rng: StdRng::from_entropy(),
            progress,
            deck: Vec::new(),
            index: 0,
            options: Vec::new(),
            correct: 0,
            selected: 0,
            answered: false,
            correct_count: 0,
            error,
        };
        t.start_session();
        t
    }

    fn start_session(&mut self) {
        let mut deck = self.items.clone();
        deck.shuffle(&mut self.rng);
        deck.truncate(SESSION_LIMIT);
        self.deck = deck;
        self.index = 0;
        self.correct_count = 0;
        self.set_question();
    }

    fn set_question(&mut self) {
        if self.finished() {
            return;
        }
        // Distractors are other kanji's meanings, so the choice is about the
        // character and not about which gloss looks plausible in isolation.
        let pool: Vec<String> = self.items.iter().map(|k| k.meaning.clone()).collect();
        let answer = self.deck[self.index].meaning.clone();
        let (options, correct) = study::options(&mut self.rng, &answer, &pool, OPTION_COUNT);
        self.options = options;
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
        if self.deck.is_empty() {
            return Transition::Stay;
        }

        if self.finished() {
            if is_confirm(code) {
                self.start_session();
            }
        } else if self.answered {
            if is_confirm(code) {
                self.index += 1;
                self.set_question();
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
        let correct = self.selected == self.correct;
        if correct {
            self.correct_count += 1;
        }
        if let Err(e) = self.persist(ctx, correct) {
            self.error = Some(e);
        }
    }

    fn persist(&mut self, ctx: &Ctx<'_>, correct: bool) -> Result<(), String> {
        let char = self.deck[self.index].char.clone();
        let mut p = self.progress.get(&char).cloned().unwrap_or_default();
        p.char = char.clone();
        p = study::grade_kanji(p, correct);

        if let Some(pid) = ctx.profile_id {
            ctx.store
                .save_kanji_progress(pid, &p)
                .map_err(|e| e.to_string())?;
            ctx.store
                .add_xp(pid, study::xp_for_answer(correct))
                .map_err(|e| e.to_string())?;
        }
        self.progress.insert(char, p);
        Ok(())
    }

    pub fn render(&self, f: &mut Frame, inner: Rect, theme: &Theme, msgs: &Messages) {
        let lines = if self.deck.is_empty() {
            vec![Line::styled(msgs.kanji_none.clone(), theme.title)]
        } else if self.finished() {
            self.summary_lines(theme, msgs)
        } else {
            self.question_lines(inner, theme, msgs)
        };
        f.render_widget(Paragraph::new(lines).wrap(Wrap { trim: false }), inner);
    }

    fn question_lines<'a>(&self, inner: Rect, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let k = &self.deck[self.index];
        let mut lines = vec![
            Line::styled(
                format!(
                    "{}  {}/{}",
                    msgs.kanji_title,
                    self.index + 1,
                    self.deck.len()
                ),
                theme.title,
            ),
            Line::raw(""),
        ];
        // The character is the subject of the question, so it gets the same
        // focal tile the kana trainer uses rather than a line of its own.
        for row in big_tile(&k.char, inner.width) {
            lines.push(Line::styled(row, theme.accent));
        }
        lines.push(Line::raw(""));
        lines.push(Line::styled(msgs.kanji_prompt.clone(), theme.normal));

        for (i, opt) in self.options.iter().enumerate() {
            let (mark, style) = if self.answered && i == self.correct {
                ("✓", theme.success)
            } else if self.answered && i == self.selected {
                ("✗", theme.error)
            } else if i == self.selected {
                ("▸", theme.selected)
            } else {
                (" ", theme.normal)
            };
            lines.push(Line::styled(format!("{mark} {}) {opt}", i + 1), style));
        }

        if self.answered {
            // Readings are shown only after answering: meeting them at recall
            // time is what ties the character to how it sounds. They share one
            // line so the help line still fits the fixed frame.
            let mut readings = Vec::new();
            if !k.on.is_empty() {
                readings.push(format!("{}: {}", msgs.kanji_on_label, k.on.join("・")));
            }
            if !k.kun.is_empty() {
                readings.push(format!("{}: {}", msgs.kanji_kun_label, k.kun.join("・")));
            }
            lines.push(Line::raw(""));
            lines.push(Line::styled(readings.join("   "), theme.subtle));
        }

        lines.push(Line::raw(""));
        lines.push(Line::styled(
            if self.answered {
                msgs.continue_help.clone()
            } else {
                msgs.choice_help.clone()
            },
            theme.help,
        ));
        if let Some(e) = &self.error {
            lines.push(Line::styled(e.clone(), theme.error));
        }
        lines
    }

    fn summary_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let mastered = self.progress.values().filter(|p| p.mastered).count();
        vec![
            Line::styled(msgs.session_done.clone(), theme.title),
            Line::raw(""),
            Line::styled(
                format!(
                    "{}: {}/{}",
                    msgs.score_label,
                    self.correct_count,
                    self.deck.len()
                ),
                theme.normal,
            ),
            Line::styled(format!("{mastered}/{}", self.items.len()), theme.subtle),
            Line::raw(""),
            Line::styled(msgs.kanji_intro_note.clone(), theme.subtle),
            Line::raw(""),
            Line::styled(msgs.restart_help.clone(), theme.help),
        ]
    }
}

fn is_confirm(code: KeyCode) -> bool {
    matches!(code, KeyCode::Enter | KeyCode::Char(' '))
}

#[cfg(test)]
mod tests {
    use super::*;
    use polyglot_core::content;

    fn course_with_kanji() -> Course {
        content::load_embedded(content::DEFAULT_PAIR).unwrap()
    }

    fn store_with_profile() -> (SqliteStore, i64) {
        let s = SqliteStore::open_in_memory().unwrap();
        let pid = s.create_profile("tester").unwrap().id;
        (s, pid)
    }

    fn view(t: &KanjiTrainer) -> String {
        let msgs = polyglot_core::i18n::default();
        crate::testutil::snapshot(|f, inner, theme| t.render(f, inner, theme, msgs))
    }

    /// A session draws from the table, capped, and asks for the meaning.
    #[test]
    fn session_is_capped_and_asks_for_the_meaning() {
        let (store, pid) = store_with_profile();
        let course = course_with_kanji();
        let t = KanjiTrainer::new(&store, &course, Some(pid));

        assert_eq!(t.deck.len(), SESSION_LIMIT.min(course.kanji.len()));
        assert_eq!(t.options.len(), OPTION_COUNT);
        assert_eq!(
            t.options[t.correct], t.deck[0].meaning,
            "the correct option is the character's meaning"
        );
        // Every option is some kanji's meaning, never an invented gloss.
        let meanings: std::collections::HashSet<&str> =
            course.kanji.iter().map(|k| k.meaning.as_str()).collect();
        for o in &t.options {
            assert!(
                meanings.contains(o.as_str()),
                "option {o:?} is a real meaning"
            );
        }
    }

    /// Answering grades the character, persists it and awards XP.
    #[test]
    fn answering_persists_progress_and_xp() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut t = KanjiTrainer::new(&store, &course_with_kanji(), Some(pid));
        let answered = t.deck[0].char.clone();

        t.selected = t.correct;
        t.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert!(t.answered);
        assert_eq!(t.correct_count, 1);

        let saved = store.get_kanji_progress(pid).unwrap();
        let p = saved
            .get(&answered)
            .unwrap_or_else(|| panic!("no progress for {answered:?}"));
        assert_eq!(p.attempts, 1);
        assert_eq!(p.streak, 1);
        assert!(
            store.get_stats(pid).unwrap().xp > 0,
            "the answer awarded XP"
        );
    }

    /// The readings appear only once the learner has committed to an answer —
    /// showing them with the question would give it away.
    #[test]
    fn readings_are_revealed_only_after_answering() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut t = KanjiTrainer::new(&store, &course_with_kanji(), Some(pid));
        let msgs = polyglot_core::i18n::default();

        let before = view(&t);
        assert!(!before.contains(&msgs.kanji_on_label) && !before.contains(&msgs.kanji_kun_label));

        t.handle(KeyCode::Char('1'), KeyModifiers::NONE, &ctx);
        let after = view(&t);
        assert!(
            after.contains(&msgs.kanji_on_label) || after.contains(&msgs.kanji_kun_label),
            "the readings show once answered; view:\n{after}"
        );
    }

    /// Confirming moves to the next character, and the session ends in a summary
    /// that can be restarted.
    #[test]
    fn advances_through_the_deck_and_restarts() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut t = KanjiTrainer::new(&store, &course_with_kanji(), Some(pid));

        for _ in 0..t.deck.len() {
            t.selected = t.correct;
            t.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // reveal
            t.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // next
        }
        assert!(t.finished(), "the deck is exhausted");
        let msgs = polyglot_core::i18n::default();
        assert!(view(&t).contains(&msgs.session_done));

        t.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert!(!t.finished(), "confirming starts a fresh session");
        assert_eq!(t.index, 0);
    }

    /// Every state must fit the fixed frame — the answered view is the tall one,
    /// since it adds the readings under the options.
    #[test]
    fn every_state_fits_the_frame() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let course = course_with_kanji();
        let msgs = polyglot_core::i18n::default();

        // Walk a whole session, checking the question and answered views of each
        // character: the help line must survive every one of them.
        let mut t = KanjiTrainer::new(&store, &course, Some(pid));
        for _ in 0..t.deck.len() {
            for answered in [false, true] {
                if answered {
                    t.selected = t.correct;
                    t.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
                }
                let v = view(&t);
                assert!(
                    v.contains(&msgs.choice_help) || v.contains(&msgs.continue_help),
                    "the help line is cut off; view:\n{v}"
                );
            }
            t.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        }
    }

    /// A pair that teaches no kanji renders an explanation instead of crashing.
    #[test]
    fn a_course_without_kanji_renders_an_empty_state() {
        let (store, pid) = store_with_profile();
        let course = Course {
            kanji: Vec::new(),
            ..course_with_kanji()
        };
        let t = KanjiTrainer::new(&store, &course, Some(pid));
        assert!(t.deck.is_empty());
        let msgs = polyglot_core::i18n::default();
        assert!(view(&t).contains(&msgs.kanji_none));
    }
}
