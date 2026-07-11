//! The N5 mock assessment: a cross-curriculum retrieval exam (vocabulary, kana,
//! grammar patterns) graded against the same 80% mastery band as the
//! end-of-chapter challenge.
//!
//! Port of the Go `internal/screens/assessment`. Passing certifies the level and
//! is persisted (never revoked); every answer flows through the regular
//! spaced-repetition and XP paths, so a failed attempt is still learning.

use std::collections::HashMap;

use chrono::Utc;
use polyglot_core::content::Course;
use polyglot_core::i18n::Messages;
use polyglot_core::model::{AssessmentResult, Card, Jlpt, KanaItem, Lesson, Pattern};
use polyglot_core::srs::{self, Grade};
use polyglot_core::storage::SqliteStore;
use polyglot_core::study::{self, AssessKind, AssessQuestion};
use rand::rngs::StdRng;
use rand::SeedableRng;
use ratatui::crossterm::event::{KeyCode, KeyModifiers};
use ratatui::layout::Rect;
use ratatui::text::Line;
use ratatui::widgets::{Paragraph, Wrap};
use ratatui::Frame;

use crate::app::{Ctx, Transition};
use crate::textfmt;
use crate::theme::Theme;

const BLANK: &str = "▁▁▁▁";
const MISSED_CAP: usize = 8;

#[derive(PartialEq, Eq)]
enum Phase {
    Intro,
    Question,
    Result,
}

pub struct Assessment {
    rng: StdRng,
    level: Jlpt,
    lessons: Vec<Lesson>,
    kana: Vec<KanaItem>,
    patterns: Vec<Pattern>,
    cards: HashMap<String, Card>,
    romaji: HashMap<String, String>,
    show_romaji: bool,

    phase: Phase,
    deck: Vec<AssessQuestion>,
    index: usize,
    selected: usize,
    answered: bool,
    correct_count: usize,
    missed: Vec<AssessQuestion>,

    kana_progress: HashMap<String, polyglot_core::model::KanaProgress>,
    pattern_progress: HashMap<String, polyglot_core::model::PatternProgress>,
    streak_applied: bool,
    prior: AssessmentResult,
    error: Option<String>,
}

impl Assessment {
    pub fn new(store: &SqliteStore, course: &Course, profile_id: Option<i64>) -> Assessment {
        let level = Jlpt::N5;
        let cards: HashMap<String, Card> = course
            .lessons
            .iter()
            .flat_map(|l| l.cards.iter().map(|c| (c.id.clone(), c.clone())))
            .collect();
        let romaji: HashMap<String, String> = cards
            .values()
            .map(|c| (c.jp.clone(), c.romaji.clone()))
            .collect();

        let mut kana_progress = HashMap::new();
        let mut pattern_progress = HashMap::new();
        let mut prior = AssessmentResult {
            level,
            passed: false,
            best_correct: 0,
            total: 0,
            taken_at: None,
        };
        let mut error = None;
        if let Some(pid) = profile_id {
            match store.get_kana_progress(pid) {
                Ok(p) => kana_progress = p,
                Err(e) => error = Some(e.to_string()),
            }
            match store.get_pattern_progress(pid) {
                Ok(p) => pattern_progress = p,
                Err(e) => error = Some(e.to_string()),
            }
            match store.get_assessment_result(pid, level) {
                Ok(r) => prior = r,
                Err(e) => error = Some(e.to_string()),
            }
        }

        let mut rng = StdRng::from_entropy();
        let deck = study::build_assessment(
            &mut rng,
            level,
            &course.lessons,
            &course.kana,
            &course.patterns,
            &cards,
        );

        Assessment {
            rng,
            level,
            lessons: course.lessons.clone(),
            kana: course.kana.clone(),
            patterns: course.patterns.clone(),
            cards,
            romaji,
            show_romaji: true,
            phase: Phase::Intro,
            deck,
            index: 0,
            selected: 0,
            answered: false,
            correct_count: 0,
            missed: Vec::new(),
            kana_progress,
            pattern_progress,
            streak_applied: false,
            prior,
            error,
        }
    }

    pub fn with_romaji(mut self, show: bool) -> Assessment {
        self.show_romaji = show;
        self
    }

    fn attempt_passed(&self) -> bool {
        study::challenge_passed(self.correct_count as i64, self.deck.len() as i64)
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
        match self.phase {
            Phase::Intro => {
                if is_confirm(code) {
                    self.start(ctx);
                }
            }
            Phase::Question => {
                if !self.answered {
                    self.answer_key(code, ctx);
                } else if is_confirm(code) {
                    self.advance(ctx);
                }
            }
            Phase::Result => {
                if is_confirm(code) {
                    if self.attempt_passed() {
                        return Transition::Pop;
                    }
                    self.restart(ctx);
                }
            }
        }
        Transition::Stay
    }

    fn start(&mut self, ctx: &Ctx<'_>) {
        self.index = 0;
        self.correct_count = 0;
        self.selected = 0;
        self.answered = false;
        self.missed.clear();
        if self.deck.is_empty() {
            self.finish(ctx);
        } else {
            self.phase = Phase::Question;
        }
    }

    fn restart(&mut self, ctx: &Ctx<'_>) {
        self.deck = study::build_assessment(
            &mut self.rng,
            self.level,
            &self.lessons,
            &self.kana,
            &self.patterns,
            &self.cards,
        );
        self.start(ctx);
    }

    fn answer_key(&mut self, code: KeyCode, ctx: &Ctx<'_>) {
        let len = self.deck[self.index].options.len();
        match code {
            KeyCode::Up | KeyCode::Char('k') => self.selected = self.selected.saturating_sub(1),
            KeyCode::Down | KeyCode::Char('j') if self.selected + 1 < len => self.selected += 1,
            KeyCode::Char(c @ '1'..='4') => {
                let i = (c as u8 - b'1') as usize;
                if i < len {
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
        let q = self.deck[self.index].clone();
        let correct = self.selected == q.correct;
        if correct {
            self.correct_count += 1;
        } else {
            self.missed.push(q.clone());
        }
        if let Some(pid) = ctx.profile_id {
            if let Err(e) = self.grade(ctx.store, pid, &q, correct) {
                self.error = Some(e);
            }
        }
    }

    fn advance(&mut self, ctx: &Ctx<'_>) {
        self.index += 1;
        self.selected = 0;
        self.answered = false;
        if self.index >= self.deck.len() {
            self.finish(ctx);
        }
    }

    fn finish(&mut self, ctx: &Ctx<'_>) {
        self.phase = Phase::Result;
        let Some(pid) = ctx.profile_id else { return };
        let mut r = self.prior.clone();
        r.level = self.level;
        r.passed = r.passed || self.attempt_passed();
        if r.taken_at.is_none() || self.correct_count as i64 > r.best_correct {
            r.best_correct = self.correct_count as i64;
            r.total = self.deck.len() as i64;
        }
        r.taken_at = Some(Utc::now());
        match ctx.store.save_assessment_result(pid, &r) {
            Ok(()) => self.prior = r,
            Err(e) => self.error = Some(e.to_string()),
        }
    }

    fn grade(
        &mut self,
        store: &SqliteStore,
        pid: i64,
        q: &AssessQuestion,
        correct: bool,
    ) -> Result<(), String> {
        match q.kind {
            AssessKind::Kana => {
                if let Some(k) = &q.kana {
                    let mut p = self.kana_progress.get(&k.char).cloned().unwrap_or_default();
                    p.char = k.char.clone();
                    p = study::grade_kana(p, correct, std::time::Duration::ZERO);
                    store
                        .save_kana_progress(pid, &p)
                        .map_err(|e| e.to_string())?;
                    self.kana_progress.insert(k.char.clone(), p);
                }
            }
            AssessKind::Pattern => {
                if let Some(pat) = &q.pattern {
                    let slot = &pat.slots[q.slot_idx];
                    let key = format!("{}:{}", pat.id, slot.name);
                    let mut p = self.pattern_progress.get(&key).cloned().unwrap_or_default();
                    p.pattern_id = pat.id.clone();
                    p.slot = slot.name.clone();
                    p = study::grade_pattern_slot(p, correct);
                    store
                        .save_pattern_progress(pid, &p)
                        .map_err(|e| e.to_string())?;
                    self.pattern_progress.insert(key, p);
                }
                if let Some(card) = &q.card {
                    review_card(store, pid, card, correct)?;
                }
            }
            AssessKind::Vocab => {
                if let Some(card) = &q.card {
                    review_card(store, pid, card, correct)?;
                }
            }
        }
        self.award_xp(store, pid, correct)
    }

    fn award_xp(&mut self, store: &SqliteStore, pid: i64, correct: bool) -> Result<(), String> {
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
        let lines = match self.phase {
            Phase::Result => self.result_lines(theme, msgs),
            Phase::Question => self.question_lines(theme, msgs),
            Phase::Intro => self.intro_lines(theme, msgs),
        };
        f.render_widget(Paragraph::new(lines).wrap(Wrap { trim: false }), inner);
    }

    fn intro_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let needed = study::challenge_needed(self.deck.len() as i64);
        let mut lines = vec![
            Line::styled(msgs.assessment_title.clone(), theme.title),
            Line::raw(""),
            Line::styled(
                textfmt::dd(&msgs.assessment_intro_fmt, needed, self.deck.len() as i64),
                theme.normal,
            ),
        ];
        if self.prior.taken_at.is_some() {
            let mut best = textfmt::dd(
                &msgs.assessment_best_fmt,
                self.prior.best_correct,
                self.prior.total,
            );
            if self.prior.passed {
                best.push(' ');
                best.push_str(&msgs.assessment_passed_badge);
            }
            lines.push(Line::raw(""));
            lines.push(Line::styled(best, theme.subtle));
        }
        lines.push(Line::raw(""));
        lines.push(Line::styled(msgs.continue_help.clone(), theme.help));
        lines
    }

    fn question_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let q = &self.deck[self.index];
        let mut lines = vec![
            Line::styled(
                format!(
                    "{}  {}/{}",
                    msgs.assessment_title,
                    self.index + 1,
                    self.deck.len()
                ),
                theme.title,
            ),
            Line::raw(""),
        ];

        match q.kind {
            AssessKind::Kana => {
                lines.push(Line::styled(msgs.kana_prompt.clone(), theme.normal));
                lines.push(Line::raw(""));
                if let Some(k) = &q.kana {
                    lines.push(Line::styled(k.char.clone(), theme.accent));
                }
            }
            AssessKind::Pattern => {
                if let Some(pat) = &q.pattern {
                    let mut fill = q.fill.clone();
                    fill.insert(pat.slots[q.slot_idx].name.clone(), BLANK.to_string());
                    lines.push(Line::styled(
                        study::render_frame(&pat.frame, &fill),
                        theme.accent,
                    ));
                    lines.push(Line::raw(""));
                    let src = q.card.as_ref().map(|c| c.source.as_str()).unwrap_or("");
                    lines.push(Line::styled(
                        textfmt::s(&msgs.assessment_pattern_prompt_fmt, src),
                        theme.normal,
                    ));
                }
            }
            AssessKind::Vocab => {
                let src = q.card.as_ref().map(|c| c.source.as_str()).unwrap_or("");
                lines.push(Line::styled(
                    textfmt::s(&msgs.quiz_question_fmt, src),
                    theme.normal,
                ));
            }
        }
        lines.push(Line::raw(""));

        for (i, opt) in q.options.iter().enumerate() {
            let label = if self.show_romaji && q.kind != AssessKind::Kana {
                match self.romaji.get(opt) {
                    Some(r) if !r.is_empty() => format!("{opt} ({r})"),
                    _ => opt.clone(),
                }
            } else {
                opt.clone()
            };
            let (mark, style) = if self.answered && i == q.correct {
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

    fn result_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let passed = self.attempt_passed();
        let total = self.deck.len() as i64;
        let mut lines = Vec::new();
        if passed {
            lines.push(Line::styled(
                msgs.assessment_pass_title.clone(),
                theme.success,
            ));
            lines.push(Line::raw(""));
            lines.push(Line::styled(
                textfmt::dd(&msgs.assessment_pass_fmt, self.correct_count as i64, total),
                theme.normal,
            ));
        } else {
            lines.push(Line::styled(
                msgs.assessment_fail_title.clone(),
                theme.title,
            ));
            lines.push(Line::raw(""));
            let fail = msgs
                .assessment_fail_fmt
                .replacen("%d", &self.correct_count.to_string(), 1)
                .replacen("%d", &total.to_string(), 1)
                .replacen("%d", &study::challenge_needed(total).to_string(), 1);
            lines.push(Line::styled(fail, theme.normal));
        }

        if !passed && !self.missed.is_empty() {
            lines.push(Line::raw(""));
            lines.push(Line::styled(
                msgs.assessment_missed_lbl.clone(),
                theme.subtle,
            ));
            for q in self.missed.iter().take(MISSED_CAP) {
                lines.push(Line::styled(missed_line(q, self.show_romaji), theme.subtle));
            }
            if self.missed.len() > MISSED_CAP {
                lines.push(Line::styled(
                    textfmt::d(
                        &msgs.assessment_more_fmt,
                        (self.missed.len() - MISSED_CAP) as i64,
                    ),
                    theme.subtle,
                ));
            }
        }

        lines.push(Line::raw(""));
        let help = if passed {
            msgs.assessment_done_help.clone()
        } else {
            msgs.assessment_retry_help.clone()
        };
        lines.push(Line::styled(help, theme.help));
        lines
    }
}

fn review_card(store: &SqliteStore, pid: i64, card: &Card, correct: bool) -> Result<(), String> {
    let state = store
        .get_card_state(pid, &card.id)
        .unwrap_or_else(|_| srs::new_card(&card.id));
    let grade = if correct { Grade::Good } else { Grade::Again };
    let state = srs::review(&state, grade, Utc::now());
    store
        .save_card_state(pid, &state)
        .map_err(|e| e.to_string())
}

fn missed_line(q: &AssessQuestion, show_romaji: bool) -> String {
    match q.kind {
        AssessKind::Kana => match &q.kana {
            Some(k) => format!("{} ({})", k.char, k.romaji),
            None => String::new(),
        },
        _ => match &q.card {
            Some(c) if show_romaji && !c.romaji.is_empty() => {
                format!("{} ({}) — {}", c.jp, c.romaji, c.source)
            }
            Some(c) => format!("{} — {}", c.jp, c.source),
            None => String::new(),
        },
    }
}

fn is_confirm(code: KeyCode) -> bool {
    matches!(code, KeyCode::Enter | KeyCode::Char(' '))
}
