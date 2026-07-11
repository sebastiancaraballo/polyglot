//! The Rikai grammar-pattern drill: a pattern picker, then a minimal-pair
//! substitution drill.
//!
//! Port of the Go `internal/screens/rikai`. Each round shows the frame with one
//! slot blanked (the rest fixed at their default filler) and asks the learner to
//! choose the missing word from a few known options.

use std::collections::{HashMap, HashSet};

use chrono::Utc;
use polyglot_core::content::Course;
use polyglot_core::i18n::Messages;
use polyglot_core::model::{Card, Pattern, Slot};
use polyglot_core::storage::SqliteStore;
use polyglot_core::study;
use rand::rngs::StdRng;
use rand::{Rng, SeedableRng};
use ratatui::crossterm::event::{KeyCode, KeyModifiers};
use ratatui::layout::Rect;
use ratatui::text::Line;
use ratatui::widgets::{Paragraph, Wrap};
use ratatui::Frame;

use crate::app::{Ctx, Transition};
use crate::textfmt;
use crate::theme::Theme;

const OPTION_COUNT: usize = 4;
const ROUND_LIMIT: usize = 6;
const BLANK: &str = "▁▁▁▁";

struct Round {
    slot_idx: usize,
    correct: Card,
    options: Vec<Card>,
}

pub struct Rikai {
    rng: StdRng,
    cards: HashMap<String, Card>,
    all_patterns: Vec<Pattern>,
    known: HashSet<String>,
    progress: HashMap<String, polyglot_core::model::PatternProgress>,
    show_romaji: bool,

    entries: Vec<(Pattern, bool)>, // (pattern, locked)
    picking: bool,
    pattern_cur: usize,

    pattern: Pattern,
    deck: Vec<Round>,
    index: usize,
    options: Vec<String>,
    correct: usize,
    selected: usize,
    answered: bool,
    correct_count: usize,
    streak_applied: bool,
    error: Option<String>,
}

impl Rikai {
    pub fn new(store: &SqliteStore, course: &Course, profile_id: Option<i64>) -> Rikai {
        let cards: HashMap<String, Card> = course
            .lessons
            .iter()
            .flat_map(|l| l.cards.iter().map(|c| (c.id.clone(), c.clone())))
            .collect();

        let mut known = HashSet::new();
        let mut progress = HashMap::new();
        let mut error = None;
        if let Some(pid) = profile_id {
            match store.get_card_states(pid) {
                Ok(states) => {
                    for (id, st) in states {
                        if study::card_known(&st) {
                            known.insert(id);
                        }
                    }
                }
                Err(e) => error = Some(e.to_string()),
            }
            match store.get_pattern_progress(pid) {
                Ok(p) => progress = p,
                Err(e) => error = Some(e.to_string()),
            }
        }

        let entries = build_entries(&course.patterns, &known);
        Rikai {
            rng: StdRng::from_entropy(),
            cards,
            all_patterns: course.patterns.clone(),
            known,
            progress,
            show_romaji: true,
            entries,
            picking: true,
            pattern_cur: 0,
            pattern: placeholder_pattern(),
            deck: Vec::new(),
            index: 0,
            options: Vec::new(),
            correct: 0,
            selected: 0,
            answered: false,
            correct_count: 0,
            streak_applied: false,
            error,
        }
    }

    pub fn with_romaji(mut self, show: bool) -> Rikai {
        self.show_romaji = show;
        self
    }

    fn known_cards(&self, slot: &Slot) -> Vec<Card> {
        slot.card_ids
            .iter()
            .filter(|id| self.known.contains(*id))
            .filter_map(|id| self.cards.get(id).cloned())
            .collect()
    }

    fn start_session(&mut self) {
        self.pattern = self.entries[self.pattern_cur].0.clone();
        let mut deck = Vec::with_capacity(ROUND_LIMIT);
        for i in 0..ROUND_LIMIT {
            let slot_idx = study::variable_slot_index(self.pattern.slots.len(), i);
            let candidates = self.known_cards(&self.pattern.slots[slot_idx]);
            if candidates.is_empty() {
                continue;
            }
            let correct = candidates[self.rng.gen_range(0..candidates.len())].clone();
            deck.push(Round {
                slot_idx,
                correct,
                options: candidates,
            });
        }
        self.deck = deck;
        self.index = 0;
        self.correct_count = 0;
        self.picking = false;
        self.set_question();
    }

    fn set_question(&mut self) {
        if self.index >= self.deck.len() {
            return;
        }
        let pool: Vec<String> = self.deck[self.index]
            .options
            .iter()
            .map(|c| c.jp.clone())
            .collect();
        let correct_jp = self.deck[self.index].correct.jp.clone();
        let (opts, correct) = study::options(&mut self.rng, &correct_jp, &pool, OPTION_COUNT);
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
        if self.picking {
            self.handle_pick(code);
            return Transition::Stay;
        }
        if self.finished() {
            if is_confirm(code) {
                self.picking = true;
                self.entries = build_entries(&self.all_patterns, &self.known);
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

    fn handle_pick(&mut self, code: KeyCode) {
        match code {
            KeyCode::Up | KeyCode::Char('k') => {
                self.pattern_cur = self.pattern_cur.saturating_sub(1)
            }
            KeyCode::Down | KeyCode::Char('j') if self.pattern_cur + 1 < self.entries.len() => {
                self.pattern_cur += 1;
            }
            _ => {}
        }
        if is_confirm(code) && !self.entries.is_empty() && !self.entries[self.pattern_cur].1 {
            self.start_session();
        }
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
        if let Some(pid) = ctx.profile_id {
            if let Err(e) = self.persist(ctx.store, pid, correct) {
                self.error = Some(e);
            }
        }
    }

    fn persist(&mut self, store: &SqliteStore, pid: i64, correct: bool) -> Result<(), String> {
        let slot = &self.pattern.slots[self.deck[self.index].slot_idx];
        let key = format!("{}:{}", self.pattern.id, slot.name);
        let mut p = self.progress.get(&key).cloned().unwrap_or_default();
        p.pattern_id = self.pattern.id.clone();
        p.slot = slot.name.clone();
        p = study::grade_pattern_slot(p, correct);
        store
            .save_pattern_progress(pid, &p)
            .map_err(|e| e.to_string())?;
        self.progress.insert(key, p);
        store
            .add_xp(pid, study::xp_for_answer(correct))
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
        let lines = if self.entries.is_empty() {
            vec![
                Line::styled(msgs.rikai_title.clone(), theme.title),
                Line::raw(""),
                Line::styled(msgs.rikai_locked.clone(), theme.subtle),
            ]
        } else if self.picking {
            self.picker_lines(theme, msgs)
        } else if self.finished() {
            self.summary_lines(theme, msgs)
        } else {
            self.question_lines(theme, msgs)
        };
        f.render_widget(Paragraph::new(lines).wrap(Wrap { trim: false }), inner);
    }

    fn picker_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let mut lines = vec![
            Line::styled(msgs.rikai_title.clone(), theme.title),
            Line::raw(""),
        ];
        for (i, (pattern, locked)) in self.entries.iter().enumerate() {
            let label = format!("{}{}", pattern.title, self.pattern_suffix(pattern, msgs));
            lines.push(if *locked {
                Line::styled(format!("⊘ {label}"), theme.subtle)
            } else if i == self.pattern_cur {
                Line::styled(format!("▸ {label}"), theme.selected)
            } else {
                Line::styled(format!("  {label}"), theme.normal)
            });
        }
        lines.push(Line::raw(""));
        if self.entries[self.pattern_cur].1 {
            lines.push(Line::styled(msgs.rikai_unlock_hint.clone(), theme.subtle));
        } else {
            lines.push(Line::styled(msgs.rikai_pick_help.clone(), theme.help));
        }
        lines.push(Line::styled(msgs.rikai_mastery_note.clone(), theme.subtle));
        lines
    }

    fn pattern_suffix(&self, p: &Pattern, msgs: &Messages) -> String {
        if p.slots.is_empty() {
            return String::new();
        }
        let mastered = p
            .slots
            .iter()
            .filter(|slot| {
                self.progress
                    .get(&format!("{}:{}", p.id, slot.name))
                    .is_some_and(|pp| pp.mastered)
            })
            .count();
        if mastered >= p.slots.len() {
            format!("  ✓ {}", msgs.rikai_pattern_fluent)
        } else {
            format!(
                "  {}",
                textfmt::dd(
                    &msgs.rikai_mastered_fmt,
                    mastered as i64,
                    p.slots.len() as i64
                )
            )
        }
    }

    fn question_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let round = &self.deck[self.index];
        let mut fill: HashMap<String, String> = HashMap::new();
        for (i, slot) in self.pattern.slots.iter().enumerate() {
            let value = if i == round.slot_idx {
                BLANK.to_string()
            } else {
                self.cards
                    .get(&slot.default)
                    .map(|c| c.jp.clone())
                    .unwrap_or_default()
            };
            fill.insert(slot.name.clone(), value);
        }

        let mut lines = vec![
            Line::styled(
                format!(
                    "{}  {}/{}",
                    msgs.rikai_title,
                    self.index + 1,
                    self.deck.len()
                ),
                theme.title,
            ),
            Line::raw(""),
            Line::styled(
                study::render_frame(&self.pattern.frame, &fill),
                theme.accent,
            ),
            Line::raw(""),
            Line::styled(
                textfmt::s(&msgs.rikai_question_fmt, &round.correct.source),
                theme.normal,
            ),
            Line::raw(""),
        ];

        for (i, opt) in self.options.iter().enumerate() {
            let label = if self.show_romaji {
                match self.romaji_for(opt) {
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
        let help = if self.answered {
            msgs.continue_help.clone()
        } else {
            msgs.choice_help.clone()
        };
        lines.push(Line::styled(help, theme.help));
        lines
    }

    fn romaji_for(&self, jp: &str) -> Option<String> {
        self.deck[self.index]
            .options
            .iter()
            .find(|c| c.jp == jp)
            .map(|c| c.romaji.clone())
    }

    fn summary_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
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
            Line::raw(""),
            Line::styled(msgs.restart_help.clone(), theme.help),
        ]
    }
}

fn build_entries(patterns: &[Pattern], known: &HashSet<String>) -> Vec<(Pattern, bool)> {
    patterns
        .iter()
        .map(|p| (p.clone(), !study::pattern_ready(p, known)))
        .collect()
}

fn placeholder_pattern() -> Pattern {
    Pattern {
        id: String::new(),
        title: String::new(),
        jlpt: None,
        frame: String::new(),
        slots: Vec::new(),
        notes: String::new(),
    }
}

fn is_confirm(code: KeyCode) -> bool {
    matches!(code, KeyCode::Enter | KeyCode::Char(' '))
}
