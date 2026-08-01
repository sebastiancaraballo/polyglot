//! The Katsudoo story runner: a chapter picker, then a beat runtime
//! (narration, dialogue, present, practice) with an end-of-chapter challenge.
//!
//! Port of the Go `internal/screens/story`. A practice beat pauses for one
//! inline check that reuses the exact same grading logic as the kana trainer and
//! quiz. Present-beat material paginates by a fixed item count (the Go original
//! budgets by frame height; simplified here).

use std::collections::HashMap;

use chrono::Utc;
use polyglot_core::content::Course;
use polyglot_core::i18n::Messages;
use polyglot_core::model::{
    Beat, BeatKind, Card, Chapter, KanaItem, KanaType, Lesson, PracticeKind, StoryProgress,
};
use polyglot_core::srs::{self, Grade};
use polyglot_core::storage::SqliteStore;
use polyglot_core::study::{self, ChallengeQuestion};
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
const ITEMS_PER_PAGE: usize = 8;

struct ChapterEntry {
    chapter: Chapter,
    progress: StoryProgress,
    locked: bool,
}

pub struct Story {
    rng: StdRng,
    chapters: Vec<Chapter>,
    lessons: Vec<Lesson>,
    kana: Vec<KanaItem>,
    show_romaji: bool,
    kana_progress: HashMap<String, polyglot_core::model::KanaProgress>,

    picking: bool,
    entries: Vec<ChapterEntry>,
    chapter_cur: usize,

    chapter: Chapter,
    beat_index: usize,
    present_page: usize,

    options: Vec<String>,
    correct: usize,
    selected: usize,
    answered: bool,

    practice_kind: Option<PracticeKind>,
    practice_card: Option<Card>,
    practice_kana: Option<KanaItem>,

    challenge: Option<Vec<ChallengeQuestion>>,
    challenge_intro: bool,
    challenge_idx: usize,
    challenge_right: usize,
    challenge_missed: Vec<ChallengeQuestion>,
    chapter_mastered: bool,
    newly_mastered: bool,

    streak_applied: bool,
    error: Option<String>,
}

impl Story {
    pub fn new(store: &SqliteStore, course: &Course, profile_id: Option<i64>) -> Story {
        let mut kana_progress = HashMap::new();
        if let Some(pid) = profile_id {
            if let Ok(kp) = store.get_kana_progress(pid) {
                kana_progress = kp;
            }
        }
        let mut story = Story {
            rng: StdRng::from_entropy(),
            chapters: course.chapters.clone(),
            lessons: course.lessons.clone(),
            kana: course.kana.clone(),
            show_romaji: true,
            kana_progress,
            picking: true,
            entries: Vec::new(),
            chapter_cur: 0,
            chapter: empty_chapter(),
            beat_index: 0,
            present_page: 0,
            options: Vec::new(),
            correct: 0,
            selected: 0,
            answered: false,
            practice_kind: None,
            practice_card: None,
            practice_kana: None,
            challenge: None,
            challenge_intro: false,
            challenge_idx: 0,
            challenge_right: 0,
            challenge_missed: Vec::new(),
            chapter_mastered: false,
            newly_mastered: false,
            streak_applied: false,
            error: None,
        };
        story.refresh_chapters(store, profile_id);
        story
    }

    pub fn with_romaji(mut self, show: bool) -> Story {
        self.show_romaji = show;
        self
    }

    fn refresh_chapters(&mut self, store: &SqliteStore, profile_id: Option<i64>) {
        let progress = profile_id
            .and_then(|pid| store.get_story_progress(pid).ok())
            .unwrap_or_default();
        self.entries = self
            .chapters
            .iter()
            .enumerate()
            .map(|(i, c)| {
                let locked = i > 0
                    && !progress
                        .get(&self.chapters[i - 1].id)
                        .is_some_and(|p| p.mastered);
                ChapterEntry {
                    chapter: c.clone(),
                    progress: progress.get(&c.id).cloned().unwrap_or_default(),
                    locked,
                }
            })
            .collect();
    }

    fn finished(&self) -> bool {
        self.beat_index >= self.chapter.beats.len()
    }

    fn start_chapter(&mut self, i: usize) {
        let entry = &self.entries[i];
        self.chapter = entry.chapter.clone();
        self.beat_index = 0;
        if !entry.progress.completed
            && (entry.progress.beat_index as usize) < self.chapter.beats.len()
        {
            self.beat_index = entry.progress.beat_index as usize;
        }
        self.picking = false;
        self.streak_applied = false;
        self.challenge = None;
        self.challenge_intro = false;
        self.challenge_idx = 0;
        self.challenge_right = 0;
        self.challenge_missed.clear();
        self.chapter_mastered = entry.progress.mastered;
        self.newly_mastered = false;
        self.enter_beat();
    }

    fn enter_beat(&mut self) {
        if self.finished() {
            return;
        }
        self.present_page = 0;
        if self.chapter.beats[self.beat_index].kind == BeatKind::Practice {
            self.set_practice_question();
        }
    }

    fn set_practice_question(&mut self) {
        let beat = self.chapter.beats[self.beat_index].clone();
        self.practice_kind = beat.practice;
        match beat.practice {
            Some(PracticeKind::Vocab) => {
                if let Some(lesson) = lesson_by_id(&self.lessons, &beat.ref_id) {
                    let card = lesson.cards[self.rng.gen_range(0..lesson.cards.len())].clone();
                    let pool: Vec<String> = lesson.cards.iter().map(|c| c.jp.clone()).collect();
                    let (opts, correct) =
                        study::options(&mut self.rng, &card.jp, &pool, OPTION_COUNT);
                    self.practice_card = Some(card);
                    self.options = opts;
                    self.correct = correct;
                }
            }
            Some(PracticeKind::Kana) => {
                let filtered = filter_kana(&self.kana, &beat.ref_id);
                if !filtered.is_empty() {
                    let k = filtered[self.rng.gen_range(0..filtered.len())].clone();
                    let pool: Vec<String> = filtered.iter().map(|k| k.romaji.clone()).collect();
                    let (opts, correct) =
                        study::options(&mut self.rng, &k.romaji, &pool, OPTION_COUNT);
                    self.practice_kana = Some(k);
                    self.options = opts;
                    self.correct = correct;
                }
            }
            None => {}
        }
        self.selected = 0;
        self.answered = false;
    }

    fn advance(&mut self, ctx: &Ctx<'_>) {
        self.beat_index += 1;
        if let Some(pid) = ctx.profile_id {
            let p = StoryProgress {
                chapter_id: self.chapter.id.clone(),
                beat_index: self.beat_index as i64,
                completed: self.finished(),
                mastered: self.chapter_mastered,
            };
            if let Err(e) = ctx.store.save_story_progress(pid, &p) {
                self.error = Some(e.to_string());
            }
        }
        if self.finished() {
            self.start_challenge(ctx);
        } else {
            self.enter_beat();
        }
    }

    fn start_challenge(&mut self, ctx: &Ctx<'_>) {
        let qs = study::build_challenge(&mut self.rng, &self.chapter, &self.lessons, &self.kana);
        if qs.is_empty() {
            self.mark_mastered(ctx);
            return;
        }
        self.challenge = Some(qs);
        self.challenge_intro = true;
        self.challenge_idx = 0;
        self.challenge_right = 0;
        self.challenge_missed.clear();
    }

    fn challenge_len(&self) -> usize {
        self.challenge.as_ref().map_or(0, Vec::len)
    }

    fn challenge_finished(&self) -> bool {
        self.challenge_idx >= self.challenge_len()
    }

    fn challenge_passed(&self) -> bool {
        study::challenge_passed(self.challenge_right as i64, self.challenge_len() as i64)
    }

    fn set_challenge_question(&mut self) {
        let Some(q) = self
            .challenge
            .as_ref()
            .and_then(|c| c.get(self.challenge_idx))
            .cloned()
        else {
            return;
        };
        self.practice_kind = Some(q.practice);
        match q.practice {
            PracticeKind::Vocab => {
                if let Some(card) = q.card.clone() {
                    let pool: Vec<String> = lesson_by_id(&self.lessons, &q.ref_id)
                        .map(|l| l.cards.iter().map(|c| c.jp.clone()).collect())
                        .unwrap_or_default();
                    let (opts, correct) =
                        study::options(&mut self.rng, &card.jp, &pool, OPTION_COUNT);
                    self.practice_card = Some(card);
                    self.options = opts;
                    self.correct = correct;
                }
            }
            PracticeKind::Kana => {
                if let Some(k) = q.kana.clone() {
                    let filtered = filter_kana(&self.kana, &q.ref_id);
                    let pool: Vec<String> = filtered.iter().map(|k| k.romaji.clone()).collect();
                    let (opts, correct) =
                        study::options(&mut self.rng, &k.romaji, &pool, OPTION_COUNT);
                    self.practice_kana = Some(k);
                    self.options = opts;
                    self.correct = correct;
                }
            }
        }
        self.selected = 0;
        self.answered = false;
    }

    fn record_challenge_answer(&mut self, ctx: &Ctx<'_>) {
        if self.selected == self.correct {
            self.challenge_right += 1;
        } else if let Some(q) = self
            .challenge
            .as_ref()
            .and_then(|c| c.get(self.challenge_idx))
            .cloned()
        {
            self.challenge_missed.push(q);
        }
        self.challenge_idx += 1;
        if !self.challenge_finished() {
            self.set_challenge_question();
        } else if self.challenge_passed() {
            self.mark_mastered(ctx);
        }
    }

    fn mark_mastered(&mut self, ctx: &Ctx<'_>) {
        if !self.chapter_mastered {
            self.newly_mastered = true;
        }
        self.chapter_mastered = true;
        if let Some(pid) = ctx.profile_id {
            let p = StoryProgress {
                chapter_id: self.chapter.id.clone(),
                beat_index: self.chapter.beats.len() as i64,
                completed: true,
                mastered: true,
            };
            if let Err(e) = ctx.store.save_story_progress(pid, &p) {
                self.error = Some(e.to_string());
            }
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
        if self.picking {
            self.handle_pick(ctx, code);
            return Transition::Stay;
        }

        let in_challenge = self.challenge.is_some();
        if in_challenge && self.challenge_intro {
            if is_confirm(code) {
                self.challenge_intro = false;
                self.set_challenge_question();
            }
        } else if in_challenge && !self.challenge_finished() {
            if !self.answered {
                self.answer_key(code, ctx);
            } else if is_confirm(code) {
                self.record_challenge_answer(ctx);
            }
        } else if in_challenge && !self.challenge_passed() {
            if is_confirm(code) {
                self.start_challenge(ctx); // retry with a fresh draw
            }
        } else if self.finished() {
            if is_confirm(code) {
                self.refresh_chapters(ctx.store, ctx.profile_id);
                self.picking = true;
            }
        } else if self.chapter.beats[self.beat_index].kind == BeatKind::Practice && !self.answered {
            self.answer_key(code, ctx);
        } else if self.chapter.beats[self.beat_index].kind == BeatKind::Present {
            self.handle_present_key(code, ctx);
        } else if is_confirm(code) {
            self.advance(ctx);
        }
        Transition::Stay
    }

    fn handle_present_key(&mut self, code: KeyCode, ctx: &Ctx<'_>) {
        let pages = self.present_page_count();
        match code {
            KeyCode::Up
            | KeyCode::Char('k')
            | KeyCode::Left
            | KeyCode::Char('h')
            | KeyCode::PageUp => {
                self.present_page = self.present_page.saturating_sub(1);
            }
            KeyCode::Down
            | KeyCode::Char('j')
            | KeyCode::Right
            | KeyCode::Char('l')
            | KeyCode::PageDown => {
                if self.present_page + 1 < pages {
                    self.present_page += 1;
                }
            }
            KeyCode::Enter | KeyCode::Char(' ') => {
                if self.present_page + 1 < pages {
                    self.present_page += 1;
                } else {
                    self.advance(ctx);
                }
            }
            _ => {}
        }
    }

    fn handle_pick(&mut self, ctx: &Ctx<'_>, code: KeyCode) {
        match code {
            KeyCode::Up | KeyCode::Char('k') => {
                self.chapter_cur = self.chapter_cur.saturating_sub(1)
            }
            KeyCode::Down | KeyCode::Char('j') if self.chapter_cur + 1 < self.entries.len() => {
                self.chapter_cur += 1;
            }
            _ => {}
        }
        let _ = ctx;
        if is_confirm(code) && !self.entries.is_empty() && !self.entries[self.chapter_cur].locked {
            self.start_chapter(self.chapter_cur);
        }
    }

    fn answer_key(&mut self, code: KeyCode, ctx: &Ctx<'_>) {
        match code {
            KeyCode::Up | KeyCode::Char('k') => self.selected = self.selected.saturating_sub(1),
            KeyCode::Down | KeyCode::Char('j') if self.selected + 1 < self.options.len() => {
                self.selected += 1;
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
        let result = match self.practice_kind {
            Some(PracticeKind::Vocab) => self.persist_vocab(ctx, correct),
            Some(PracticeKind::Kana) => self.persist_kana(ctx, correct),
            None => Ok(()),
        };
        if let Err(e) = result {
            self.error = Some(e);
        }
    }

    fn persist_vocab(&mut self, ctx: &Ctx<'_>, correct: bool) -> Result<(), String> {
        let (Some(pid), Some(card)) = (ctx.profile_id, self.practice_card.clone()) else {
            return Ok(());
        };
        let state = ctx
            .store
            .get_card_state(pid, &card.id)
            .unwrap_or_else(|_| srs::new_card(&card.id));
        let grade = if correct { Grade::Good } else { Grade::Again };
        let state = srs::review(&state, grade, Utc::now());
        ctx.store
            .save_card_state(pid, &state)
            .map_err(|e| e.to_string())?;
        self.award_xp(ctx, pid, correct)
    }

    fn persist_kana(&mut self, ctx: &Ctx<'_>, correct: bool) -> Result<(), String> {
        let (Some(pid), Some(kana)) = (ctx.profile_id, self.practice_kana.clone()) else {
            return Ok(());
        };
        let mut p = self
            .kana_progress
            .get(&kana.char)
            .cloned()
            .unwrap_or_default();
        p.char = kana.char.clone();
        p = study::grade_kana(p, correct, std::time::Duration::ZERO);
        ctx.store
            .save_kana_progress(pid, &p)
            .map_err(|e| e.to_string())?;
        self.kana_progress.insert(kana.char, p);
        self.award_xp(ctx, pid, correct)
    }

    fn award_xp(&mut self, ctx: &Ctx<'_>, pid: i64, correct: bool) -> Result<(), String> {
        ctx.store
            .add_xp(pid, study::xp_for_answer(correct))
            .map_err(|e| e.to_string())?;
        if !self.streak_applied {
            let stats = ctx.store.get_stats(pid).map_err(|e| e.to_string())?;
            ctx.store
                .save_stats(pid, &study::update_streak(stats, Utc::now()))
                .map_err(|e| e.to_string())?;
            self.streak_applied = true;
        }
        Ok(())
    }

    // --- Rendering --------------------------------------------------------

    pub fn render(&self, f: &mut Frame, inner: Rect, theme: &Theme, msgs: &Messages) {
        let lines = if self.entries.is_empty() {
            vec![
                Line::styled(msgs.story_title.clone(), theme.title),
                Line::raw(""),
                Line::styled(msgs.story_empty.clone(), theme.subtle),
            ]
        } else if self.picking {
            self.picker_lines(theme, msgs)
        } else if self.challenge.is_some() && self.challenge_intro {
            self.challenge_intro_lines(theme, msgs)
        } else if self.challenge.is_some() && !self.challenge_finished() {
            self.question_lines(theme, msgs, msgs.story_challenge_title.clone(), true)
        } else if self.challenge.is_some() && !self.challenge_passed() {
            self.challenge_fail_lines(theme, msgs)
        } else if self.finished() {
            self.done_lines(theme, msgs)
        } else {
            match self.chapter.beats[self.beat_index].kind {
                BeatKind::Practice => {
                    self.question_lines(theme, msgs, self.chapter.title.clone(), false)
                }
                BeatKind::Present => self.present_lines(theme, msgs),
                _ => self.beat_lines(theme, msgs),
            }
        };
        f.render_widget(Paragraph::new(lines).wrap(Wrap { trim: false }), inner);
    }

    fn picker_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let mut lines = vec![
            Line::styled(msgs.story_title.clone(), theme.title),
            Line::raw(""),
        ];
        for (i, entry) in self.entries.iter().enumerate() {
            if entry.locked {
                lines.push(Line::styled(
                    format!("⊘ {}", entry.chapter.title),
                    theme.subtle,
                ));
            } else {
                let label = format!("{}{}", entry.chapter.title, chapter_suffix(msgs, entry));
                lines.push(if i == self.chapter_cur {
                    Line::styled(format!("▸ {label}"), theme.selected)
                } else {
                    Line::styled(format!("  {label}"), theme.normal)
                });
            }
        }
        lines.push(Line::raw(""));
        if self.entries[self.chapter_cur].locked {
            let prev = &self.entries[self.chapter_cur - 1].chapter.title;
            lines.push(Line::styled(
                textfmt::s(&msgs.story_locked_hint_fmt, prev),
                theme.subtle,
            ));
        } else {
            lines.push(Line::styled(msgs.story_pick_help.clone(), theme.help));
        }
        lines.push(Line::styled(msgs.story_gate_note.clone(), theme.subtle));
        lines
    }

    fn beat_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let beat = &self.chapter.beats[self.beat_index];
        let mut lines = vec![
            Line::styled(self.chapter.title.clone(), theme.title),
            Line::raw(""),
        ];
        if !beat.place.is_empty() {
            lines.push(Line::styled(beat.place.clone(), theme.subtle));
            lines.push(Line::raw(""));
        }
        if beat.kind == BeatKind::Dialogue {
            lines.push(Line::styled(beat.speaker.clone(), theme.accent));
        }
        lines.push(Line::styled(self.jp_line(beat), theme.normal));
        lines.push(Line::styled(beat.source.clone(), theme.subtle));
        lines.push(Line::raw(""));
        lines.push(Line::styled(msgs.continue_help.clone(), theme.help));
        lines
    }

    fn jp_line(&self, beat: &Beat) -> String {
        if self.show_romaji && !beat.romaji.is_empty() {
            format!("{} ({})", beat.jp, beat.romaji)
        } else {
            beat.jp.clone()
        }
    }

    fn present_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let beat = &self.chapter.beats[self.beat_index];
        let items = self.present_items(beat);
        let pages: Vec<&[String]> = if items.is_empty() {
            vec![&[]]
        } else {
            items.chunks(ITEMS_PER_PAGE).collect()
        };
        let page = self.present_page.min(pages.len() - 1);

        let mut lines = vec![
            Line::styled(self.chapter.title.clone(), theme.title),
            Line::raw(""),
        ];
        if !beat.place.is_empty() {
            lines.push(Line::styled(beat.place.clone(), theme.subtle));
            lines.push(Line::raw(""));
        }
        if !beat.jp.is_empty() {
            if !beat.speaker.is_empty() {
                lines.push(Line::styled(beat.speaker.clone(), theme.accent));
            }
            lines.push(Line::styled(self.jp_line(beat), theme.normal));
            lines.push(Line::styled(beat.source.clone(), theme.subtle));
            lines.push(Line::raw(""));
        }
        lines.push(Line::styled(msgs.story_present_label.clone(), theme.subtle));
        for it in pages[page] {
            lines.push(Line::styled(format!("  {it}"), theme.normal));
        }
        if pages.len() > 1 {
            lines.push(Line::styled(
                textfmt::dd(
                    &msgs.story_present_page_fmt,
                    (page + 1) as i64,
                    pages.len() as i64,
                ),
                theme.subtle,
            ));
        }
        lines.push(Line::raw(""));
        let help = if pages.len() > 1 && page + 1 < pages.len() {
            msgs.story_present_more_help.clone()
        } else {
            msgs.continue_help.clone()
        };
        lines.push(Line::styled(help, theme.help));
        lines
    }

    fn present_page_count(&self) -> usize {
        let items = self.present_items(&self.chapter.beats[self.beat_index]);
        if items.is_empty() {
            1
        } else {
            items.len().div_ceil(ITEMS_PER_PAGE)
        }
    }

    fn present_items(&self, beat: &Beat) -> Vec<String> {
        match beat.practice {
            Some(PracticeKind::Vocab) => lesson_by_id(&self.lessons, &beat.ref_id)
                .map(|l| {
                    l.cards
                        .iter()
                        .map(|c| {
                            if self.show_romaji && !c.romaji.is_empty() {
                                format!("{} ({}) — {}", c.jp, c.romaji, c.source)
                            } else {
                                format!("{} — {}", c.jp, c.source)
                            }
                        })
                        .collect()
                })
                .unwrap_or_default(),
            Some(PracticeKind::Kana) => filter_kana(&self.kana, &beat.ref_id)
                .iter()
                .map(|k| format!("{} ({})", k.char, k.romaji))
                .collect(),
            None => Vec::new(),
        }
    }

    fn question_lines<'a>(
        &self,
        theme: &Theme,
        msgs: &Messages,
        title: String,
        challenge: bool,
    ) -> Vec<Line<'a>> {
        let mut lines = Vec::new();
        if challenge {
            lines.push(Line::styled(
                format!(
                    "{}  {}",
                    title,
                    textfmt::dd(
                        &msgs.story_challenge_q_fmt,
                        (self.challenge_idx + 1) as i64,
                        self.challenge_len() as i64
                    )
                ),
                theme.title,
            ));
        } else {
            lines.push(Line::styled(title, theme.title));
        }
        lines.push(Line::raw(""));

        let ref_id = self.current_ref_id();
        match self.practice_kind {
            Some(PracticeKind::Vocab) => {
                let src = self
                    .practice_card
                    .as_ref()
                    .map(|c| c.source.as_str())
                    .unwrap_or("");
                lines.push(Line::styled(
                    textfmt::s(&msgs.quiz_question_fmt, src),
                    theme.normal,
                ));
            }
            Some(PracticeKind::Kana) => {
                lines.push(Line::styled(msgs.kana_prompt.clone(), theme.normal));
                lines.push(Line::raw(""));
                if let Some(k) = &self.practice_kana {
                    lines.push(Line::styled(k.char.clone(), theme.accent));
                }
            }
            None => {}
        }
        lines.push(Line::raw(""));

        let romaji: HashMap<String, String> =
            if matches!(self.practice_kind, Some(PracticeKind::Vocab)) {
                lesson_by_id(&self.lessons, &ref_id)
                    .map(|l| {
                        l.cards
                            .iter()
                            .map(|c| (c.jp.clone(), c.romaji.clone()))
                            .collect()
                    })
                    .unwrap_or_default()
            } else {
                HashMap::new()
            };
        for (i, opt) in self.options.iter().enumerate() {
            let label =
                if self.show_romaji && matches!(self.practice_kind, Some(PracticeKind::Vocab)) {
                    match romaji.get(opt) {
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

    fn current_ref_id(&self) -> String {
        if let Some(challenge) = &self.challenge {
            challenge
                .get(self.challenge_idx)
                .map(|q| q.ref_id.clone())
                .unwrap_or_default()
        } else if !self.finished() {
            self.chapter.beats[self.beat_index].ref_id.clone()
        } else {
            String::new()
        }
    }

    fn challenge_intro_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let needed = study::challenge_needed(self.challenge_len() as i64);
        vec![
            Line::styled(msgs.story_challenge_title.clone(), theme.title),
            Line::raw(""),
            Line::styled(
                textfmt::dd(
                    &msgs.story_challenge_intro_fmt,
                    needed,
                    self.challenge_len() as i64,
                ),
                theme.normal,
            ),
            Line::raw(""),
            Line::styled(msgs.continue_help.clone(), theme.help),
        ]
    }

    fn challenge_fail_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let total = self.challenge_len() as i64;
        let fail = msgs
            .story_challenge_fail_fmt
            .replacen("%d", &self.challenge_right.to_string(), 1)
            .replacen("%d", &total.to_string(), 1)
            .replacen("%d", &study::challenge_needed(total).to_string(), 1);
        let mut lines = vec![
            Line::styled(msgs.story_challenge_title.clone(), theme.title),
            Line::raw(""),
            Line::styled(fail, theme.normal),
        ];
        if !self.challenge_missed.is_empty() {
            lines.push(Line::raw(""));
            lines.push(Line::styled(
                msgs.story_challenge_missed_lbl.clone(),
                theme.subtle,
            ));
            for q in &self.challenge_missed {
                lines.push(Line::styled(missed_line(q, self.show_romaji), theme.normal));
            }
        }
        lines.push(Line::raw(""));
        lines.push(Line::styled(
            msgs.story_challenge_retry_help.clone(),
            theme.help,
        ));
        lines
    }

    fn done_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let mut lines = vec![
            Line::styled(msgs.story_done_title.clone(), theme.title),
            Line::raw(""),
        ];
        if self.challenge.is_some() {
            lines.push(Line::styled(
                textfmt::dd(
                    &msgs.story_challenge_pass_fmt,
                    self.challenge_right as i64,
                    self.challenge_len() as i64,
                ),
                theme.normal,
            ));
        }
        if self.newly_mastered {
            if let Some(next) = self.next_chapter_title() {
                lines.push(Line::styled(
                    textfmt::s(&msgs.story_unlocked_fmt, &next),
                    theme.success,
                ));
            }
        }
        lines.push(Line::raw(""));
        lines.push(Line::styled(msgs.story_done_next.clone(), theme.help));
        lines
    }

    fn next_chapter_title(&self) -> Option<String> {
        let idx = self.chapters.iter().position(|c| c.id == self.chapter.id)?;
        self.chapters.get(idx + 1).map(|c| c.title.clone())
    }
}

fn lesson_by_id<'a>(lessons: &'a [Lesson], id: &str) -> Option<&'a Lesson> {
    lessons.iter().find(|l| l.id == id)
}

fn filter_kana(kana: &[KanaItem], ref_id: &str) -> Vec<KanaItem> {
    let typ = KanaType::from_str(ref_id);
    kana.iter()
        .filter(|k| typ.is_some_and(|t| k.kana_type == t))
        .cloned()
        .collect()
}

fn chapter_suffix(msgs: &Messages, entry: &ChapterEntry) -> String {
    let p = &entry.progress;
    if p.mastered {
        format!("  {}", msgs.story_mastered_badge)
    } else if p.completed {
        format!("  {}", msgs.story_complete_badge)
    } else if p.beat_index > 0 {
        format!(
            "  {}",
            textfmt::dd(
                &msgs.story_progress_fmt,
                p.beat_index,
                entry.chapter.beats.len() as i64
            )
        )
    } else {
        String::new()
    }
}

fn missed_line(q: &ChallengeQuestion, show_romaji: bool) -> String {
    match q.practice {
        PracticeKind::Kana => match &q.kana {
            Some(k) => format!("{} ({})", k.char, k.romaji),
            None => String::new(),
        },
        PracticeKind::Vocab => match &q.card {
            Some(c) if show_romaji && !c.romaji.is_empty() => {
                format!("{} ({}) — {}", c.jp, c.romaji, c.source)
            }
            Some(c) => format!("{} — {}", c.jp, c.source),
            None => String::new(),
        },
    }
}

fn empty_chapter() -> Chapter {
    Chapter {
        id: String::new(),
        title: String::new(),
        beats: Vec::new(),
    }
}

fn is_confirm(code: KeyCode) -> bool {
    matches!(code, KeyCode::Enter | KeyCode::Char(' '))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::app::Ctx;
    use polyglot_core::content;
    use polyglot_core::storage::SqliteStore;

    #[test]
    fn picker_starts_the_first_chapter() {
        let store = SqliteStore::open_in_memory().unwrap();
        let course = content::load_embedded("es-ja").unwrap();
        let pid = store.create_profile("A").unwrap().id;
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut s = Story::new(&store, &course, Some(pid));
        assert!(s.picking);
        assert!(!s.entries.is_empty(), "the course ships story chapters");
        assert!(!s.entries[0].locked, "the first chapter is always open");
        assert!(
            s.entries[1..].iter().all(|e| e.locked),
            "later chapters gated"
        );

        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // start chapter 0
        assert!(!s.picking, "a chapter was started");
        assert!(!s.chapter.beats.is_empty());
    }

    fn beat(kind: BeatKind, speaker: &str, source: &str, jp: &str) -> Beat {
        Beat {
            kind,
            speaker: speaker.to_string(),
            place: String::new(),
            source: source.to_string(),
            jp: jp.to_string(),
            romaji: String::new(),
            practice: None,
            ref_id: String::new(),
        }
    }

    fn practice_beat(kind: PracticeKind, ref_id: &str) -> Beat {
        Beat {
            practice: Some(kind),
            ref_id: ref_id.to_string(),
            ..beat(BeatKind::Practice, "", "", "")
        }
    }

    /// A 4-beat chapter: narration, dialogue, one vocab practice, closing line.
    fn test_chapter() -> Chapter {
        Chapter {
            id: "test-chapter".to_string(),
            title: "Capítulo de prueba".to_string(),
            beats: vec![
                beat(BeatKind::Narration, "", "Es de día.", "昼です。"),
                beat(BeatKind::Dialogue, "Yui", "Hola.", "こんにちは。"),
                practice_beat(PracticeKind::Vocab, "test-lesson"),
                beat(BeatKind::Dialogue, "Yui", "Adiós.", "さようなら。"),
            ],
        }
    }

    /// Sits behind `test_chapter`'s mastery gate in the picker.
    fn second_chapter() -> Chapter {
        Chapter {
            id: "test-chapter-2".to_string(),
            title: "Segundo capítulo".to_string(),
            beats: vec![beat(BeatKind::Narration, "", "Sigue.", "続く。")],
        }
    }

    /// Nothing to evaluate: completing it masters it.
    fn no_practice_chapter() -> Chapter {
        Chapter {
            id: "no-practice".to_string(),
            title: "Sin práctica".to_string(),
            beats: vec![beat(
                BeatKind::Narration,
                "",
                "Solo texto.",
                "テキストだけ。",
            )],
        }
    }

    fn test_course(chapters: Vec<Chapter>) -> Course {
        Course {
            pair: "es-xx".to_string(),
            lessons: vec![Lesson {
                id: "test-lesson".to_string(),
                title: "t".to_string(),
                jlpt: None,
                functions: Vec::new(),
                cards: vec![
                    Card {
                        id: "test-lesson:1".to_string(),
                        source: "Hola".to_string(),
                        jp: "こんにちは".to_string(),
                        romaji: "konnichiwa".to_string(),
                        notes: String::new(),
                        jlpt: None,
                        functions: Vec::new(),
                        freq: 0,
                    },
                    Card {
                        id: "test-lesson:2".to_string(),
                        source: "Adiós".to_string(),
                        jp: "さようなら".to_string(),
                        romaji: "sayounara".to_string(),
                        notes: String::new(),
                        jlpt: None,
                        functions: Vec::new(),
                        freq: 0,
                    },
                ],
            }],
            kana: vec![
                KanaItem {
                    char: "あ".to_string(),
                    romaji: "a".to_string(),
                    kana_type: polyglot_core::model::KanaType::Hiragana,
                    category: polyglot_core::model::KanaCategory::Base,
                },
                KanaItem {
                    char: "い".to_string(),
                    romaji: "i".to_string(),
                    kana_type: polyglot_core::model::KanaType::Hiragana,
                    category: polyglot_core::model::KanaCategory::Base,
                },
            ],
            kanji: Vec::new(),
            patterns: Vec::new(),
            chapters,
        }
    }

    fn store_with_profile() -> (SqliteStore, i64) {
        let s = SqliteStore::open_in_memory().unwrap();
        let pid = s.create_profile("tester").unwrap().id;
        (s, pid)
    }

    /// Plays `test_chapter`'s four beats to the end, answering its practice beat
    /// correctly, and leaves the story on the first challenge question.
    fn finish_beats(s: &mut Story, ctx: &Ctx<'_>) {
        s.handle(KeyCode::Enter, KeyModifiers::NONE, ctx); // narration -> dialogue
        s.handle(KeyCode::Enter, KeyModifiers::NONE, ctx); // dialogue -> practice
        s.selected = s.correct;
        s.handle(KeyCode::Enter, KeyModifiers::NONE, ctx); // reveal
        s.handle(KeyCode::Enter, KeyModifiers::NONE, ctx); // -> closing dialogue
        s.handle(KeyCode::Enter, KeyModifiers::NONE, ctx); // -> finished, challenge
        assert!(s.challenge.is_some(), "the challenge opened");
        assert!(s.challenge_intro, "on the challenge intro");
        s.handle(KeyCode::Enter, KeyModifiers::NONE, ctx); // dismiss the intro
    }

    /// Answers every remaining challenge question correctly or incorrectly.
    fn run_challenge(s: &mut Story, ctx: &Ctx<'_>, correct: bool) {
        while !s.challenge_finished() {
            s.selected = if correct {
                s.correct
            } else {
                (s.correct + 1) % s.options.len().max(1)
            };
            s.handle(KeyCode::Enter, KeyModifiers::NONE, ctx); // reveal
            s.handle(KeyCode::Enter, KeyModifiers::NONE, ctx); // record + advance
        }
    }

    /// With no chapters at all the screen still renders.
    #[test]
    fn empty_course_renders_without_panicking() {
        let (store, pid) = store_with_profile();
        let s = Story::new(&store, &test_course(Vec::new()), Some(pid));
        assert!(s.entries.is_empty());
        let msgs = polyglot_core::i18n::default();
        let _ = crate::testutil::snapshot(|f, inner, theme| s.render(f, inner, theme, msgs));
    }

    /// The picker lists every chapter and gates all but the first.
    #[test]
    fn picker_lists_chapters_and_gates_them() {
        let (store, pid) = store_with_profile();
        let s = Story::new(
            &store,
            &test_course(vec![test_chapter(), second_chapter()]),
            Some(pid),
        );
        assert_eq!(s.entries.len(), 2);
        assert!(!s.entries[0].locked);
        assert!(s.entries[1].locked, "chapter 2 is gated on chapter 1");

        let msgs = polyglot_core::i18n::default();
        let view = crate::testutil::snapshot(|f, inner, theme| s.render(f, inner, theme, msgs));
        assert!(view.contains("Capítulo de prueba"));
        assert!(view.contains("Segundo capítulo"));
    }

    /// Starting a chapter enters its first beat; confirming walks narration and
    /// dialogue until a practice beat asks a question.
    #[test]
    fn advancing_reaches_the_practice_beat() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut s = Story::new(&store, &test_course(vec![test_chapter()]), Some(pid));
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert_eq!(s.beat_index, 0, "starts on the first beat");

        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert_eq!(s.beat_index, 1);
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert_eq!(s.beat_index, 2);
        assert_eq!(s.practice_kind, Some(PracticeKind::Vocab));
        assert!(!s.options.is_empty(), "the practice beat asks a question");
    }

    /// A vocab practice answer persists card state and awards XP.
    #[test]
    fn answering_vocab_practice_persists_card_state_and_xp() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut s = Story::new(&store, &test_course(vec![test_chapter()]), Some(pid));
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // start
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // on the practice beat
        s.selected = s.correct;
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // reveal

        assert!(
            !store.get_card_states(pid).unwrap().is_empty(),
            "the answer scheduled the card"
        );
        assert!(
            store.get_stats(pid).unwrap().xp > 0,
            "the answer awarded XP"
        );
    }

    /// A kana practice beat records progress under the character.
    #[test]
    fn answering_kana_practice_persists_kana_progress() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let chapter = Chapter {
            id: "kana-chapter".to_string(),
            title: "Kana".to_string(),
            beats: vec![practice_beat(PracticeKind::Kana, "hiragana")],
        };
        let mut s = Story::new(&store, &test_course(vec![chapter]), Some(pid));
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // start on the practice beat
        assert_eq!(s.practice_kind, Some(PracticeKind::Kana));
        s.selected = s.correct;
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // reveal

        assert!(
            !store.get_kana_progress(pid).unwrap().is_empty(),
            "the answer recorded kana progress"
        );
    }

    /// Failing the end-of-chapter challenge neither masters the chapter nor
    /// unlocks the next one.
    #[test]
    fn failing_the_challenge_keeps_the_next_chapter_locked() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut s = Story::new(
            &store,
            &test_course(vec![test_chapter(), second_chapter()]),
            Some(pid),
        );
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        finish_beats(&mut s, &ctx);
        run_challenge(&mut s, &ctx, false);

        assert!(!s.challenge_passed(), "all-wrong answers must not pass");
        s.refresh_chapters(&store, Some(pid));
        assert!(s.entries[1].locked, "the next chapter stays locked");
        let progress = store.get_story_progress(pid).unwrap();
        assert!(
            !progress["test-chapter"].mastered,
            "a failed challenge must not persist mastery"
        );
    }

    /// Passing masters the chapter, persists it, and unlocks the next one.
    #[test]
    fn passing_the_challenge_masters_and_unlocks() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut s = Story::new(
            &store,
            &test_course(vec![test_chapter(), second_chapter()]),
            Some(pid),
        );
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        finish_beats(&mut s, &ctx);
        run_challenge(&mut s, &ctx, true);

        assert!(s.challenge_passed(), "all-correct answers pass");
        let progress = store.get_story_progress(pid).unwrap();
        assert!(
            progress["test-chapter"].mastered,
            "passing persists mastery"
        );

        // Confirming on the done screen returns to a picker that reflects it.
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert!(s.picking, "confirming returns to the picker");
        assert!(s.entries[0].progress.mastered, "the picker shows mastery");
        assert!(!s.entries[1].locked, "the next chapter is unlocked");
    }

    /// A failed challenge can be retried immediately with a fresh draw.
    #[test]
    fn challenge_retry_rebuilds_questions() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut s = Story::new(&store, &test_course(vec![test_chapter()]), Some(pid));
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        finish_beats(&mut s, &ctx);
        run_challenge(&mut s, &ctx, false);
        assert!(s.challenge_finished() && !s.challenge_passed());

        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // retry
        assert_eq!(
            s.challenge_idx, 0,
            "the retry starts from the first question"
        );
        assert_eq!(s.challenge_right, 0, "the score is reset");
        assert!(s.challenge_intro, "the retry re-opens the intro");
    }

    /// A chapter with nothing to practice masters on completion.
    #[test]
    fn no_practice_chapter_auto_masters() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut s = Story::new(&store, &test_course(vec![no_practice_chapter()]), Some(pid));
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // start
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // the single narration beat

        assert!(s.challenge.is_none(), "no practice, no challenge");
        let progress = store.get_story_progress(pid).unwrap();
        assert!(
            progress["no-practice"].mastered,
            "completing a no-practice chapter masters it"
        );
    }

    /// A partially played chapter resumes where the learner left off; a
    /// completed one replays from the start, keeping its mastery.
    #[test]
    fn resume_and_replay() {
        let (store, pid) = store_with_profile();
        store
            .save_story_progress(
                pid,
                &StoryProgress {
                    chapter_id: "test-chapter".to_string(),
                    beat_index: 2,
                    completed: false,
                    mastered: false,
                },
            )
            .unwrap();
        let mut s = Story::new(&store, &test_course(vec![test_chapter()]), Some(pid));
        s.start_chapter(0);
        assert_eq!(s.beat_index, 2, "resumed where the learner left off");

        // A completed (and mastered) chapter restarts from scratch.
        store
            .save_story_progress(
                pid,
                &StoryProgress {
                    chapter_id: "test-chapter".to_string(),
                    beat_index: 4,
                    completed: true,
                    mastered: true,
                },
            )
            .unwrap();
        let mut s = Story::new(&store, &test_course(vec![test_chapter()]), Some(pid));
        s.start_chapter(0);
        assert_eq!(
            s.beat_index, 0,
            "a completed chapter replays from the start"
        );
        assert!(s.chapter_mastered, "replaying preserves mastery");
    }

    /// Confirming on a locked chapter does nothing, and the picker explains the
    /// gate.
    #[test]
    fn locked_chapter_does_not_start_and_explains_itself() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut s = Story::new(
            &store,
            &test_course(vec![test_chapter(), second_chapter()]),
            Some(pid),
        );
        s.chapter_cur = 1; // the locked chapter
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert!(s.picking, "a locked chapter must not start");

        let msgs = polyglot_core::i18n::default();
        let view = crate::testutil::snapshot(|f, inner, theme| s.render(f, inner, theme, msgs));
        assert!(
            view.contains(&msgs.story_gate_note),
            "the picker states the gating rule"
        );
    }

    /// A present beat lists its pool and advances to the next beat.
    #[test]
    fn present_beat_renders_pool_and_advances() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let chapter = Chapter {
            id: "present-chapter".to_string(),
            title: "Presenta".to_string(),
            beats: vec![
                Beat {
                    practice: Some(PracticeKind::Vocab),
                    ref_id: "test-lesson".to_string(),
                    ..beat(BeatKind::Present, "", "", "")
                },
                beat(BeatKind::Narration, "", "Fin.", "おわり。"),
            ],
        };
        let mut s = Story::new(&store, &test_course(vec![chapter]), Some(pid));
        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // start on the present beat
        assert_eq!(s.beat_index, 0);

        let msgs = polyglot_core::i18n::default();
        let view = crate::testutil::snapshot(|f, inner, theme| s.render(f, inner, theme, msgs));
        // Wide glyphs occupy two buffer cells, so the flattened view carries a
        // filler space after each one: compare with spaces collapsed.
        let dense: String = view.chars().filter(|c| !c.is_whitespace()).collect();
        assert!(
            dense.contains("こんにちは"),
            "the present beat lists the pool it introduces; view:\n{view}"
        );

        s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert_eq!(s.beat_index, 1, "confirming leaves the present beat");
    }

    /// The embedded story fits the fixed frame at every beat of every chapter —
    /// the guard against content that overflows in production.
    #[test]
    fn embedded_story_fits_the_frame() {
        let store = SqliteStore::open_in_memory().unwrap();
        let course = content::load_embedded(content::DEFAULT_PAIR).unwrap();
        let pid = store.create_profile("A").unwrap().id;
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let msgs = polyglot_core::i18n::default();

        for (i, chapter) in course.chapters.iter().enumerate() {
            let mut s = Story::new(&store, &course, Some(pid));
            s.chapter = chapter.clone();
            s.picking = false;
            for beat_index in 0..chapter.beats.len() {
                s.beat_index = beat_index;
                s.enter_beat();
                for page in 0..s.present_page_count().max(1) {
                    s.present_page = page;
                    let view = crate::testutil::snapshot(|f, inner, theme| {
                        s.render(f, inner, theme, msgs)
                    });
                    let widest = view.lines().map(|l| l.chars().count()).max().unwrap_or(0);
                    assert!(
                        widest <= 80,
                        "chapter {i} beat {beat_index} page {page}: line of {widest} cells overflows"
                    );
                }
            }
            let _ = &ctx;
        }
    }
}
