//! The kana trainer: a group picker, then a timed recognition/recall drill.
//!
//! Port of the Go `internal/screens/kana`. Shows a character and asks the
//! learner to choose its romaji (recognition) — or, in reverse mode, shows the
//! romaji and asks for the glyph (recall). A kana is driven to mastery by a run
//! of correct answers, which drives the Foundations decoding gate. Answers are
//! timed only to record each kana's best time; speed does not affect mastery.

use std::collections::HashMap;
use std::time::Instant;

use polyglot_core::content::Course;
use polyglot_core::i18n::Messages;
use polyglot_core::model::{KanaCategory, KanaItem, KanaProgress, KanaType};
use polyglot_core::storage::SqliteStore;
use polyglot_core::study::{self, Gate};
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

const OPTION_COUNT: usize = 4;

/// A selectable practice set in the pre-session picker.
enum GroupMatch {
    All,
    Cat(KanaType, Vec<KanaCategory>),
}

impl GroupMatch {
    fn matches(&self, it: &KanaItem) -> bool {
        match self {
            GroupMatch::All => true,
            GroupMatch::Cat(typ, cats) => it.kana_type == *typ && cats.contains(&it.category),
        }
    }
}

struct Group {
    label: String,
    locked: bool,
    matcher: GroupMatch,
}

pub struct KanaTrainer {
    items: Vec<KanaItem>,
    rng: StdRng,
    progress: HashMap<String, KanaProgress>,
    gate: Gate,

    intro: bool,
    picking: bool,
    groups: Vec<Group>,
    group_cursor: usize,
    reverse: bool,

    deck: Vec<KanaItem>,
    pool: Vec<String>,
    index: usize,
    options: Vec<String>,
    correct: usize,
    selected: usize,
    answered: bool,
    correct_count: usize,
    shown_at: Option<Instant>,
    error: Option<String>,
}

impl KanaTrainer {
    pub fn new(store: &SqliteStore, course: &Course, profile_id: Option<i64>) -> KanaTrainer {
        let mut progress = HashMap::new();
        let mut intro = false;
        let mut error = None;
        if let Some(pid) = profile_id {
            match store.get_kana_progress(pid) {
                Ok(p) => progress = p,
                Err(e) => error = Some(e.to_string()),
            }
            if let Ok(prof) = store.get_profile(pid) {
                intro = !prof.kana_onboarded;
            }
        }
        let gate = study::new_gate(&course.kana, &progress);
        let groups = build_groups(&gate);
        KanaTrainer {
            items: course.kana.clone(),
            rng: StdRng::from_entropy(),
            progress,
            gate,
            intro,
            picking: !intro,
            groups,
            group_cursor: 0,
            reverse: false,
            deck: Vec::new(),
            pool: Vec::new(),
            index: 0,
            options: Vec::new(),
            correct: 0,
            selected: 0,
            answered: false,
            correct_count: 0,
            shown_at: None,
            error,
        }
    }

    fn answer_text(&self, it: &KanaItem) -> String {
        if self.reverse {
            it.char.clone()
        } else {
            it.romaji.clone()
        }
    }

    fn finished(&self) -> bool {
        self.index >= self.deck.len()
    }

    fn start_session(&mut self) {
        let matcher = &self.groups[self.group_cursor].matcher;
        let items: Vec<KanaItem> = self
            .items
            .iter()
            .filter(|it| matcher.matches(it))
            .cloned()
            .collect();
        self.pool = items.iter().map(|it| self.answer_text(it)).collect();
        self.deck = items;
        self.deck.shuffle(&mut self.rng);
        self.index = 0;
        self.correct_count = 0;
        self.picking = false;
        self.set_question();
    }

    fn set_question(&mut self) {
        if self.index >= self.deck.len() {
            return;
        }
        let answer = self.answer_text(&self.deck[self.index]);
        let (opts, correct) = study::options(&mut self.rng, &answer, &self.pool, OPTION_COUNT);
        self.options = opts;
        self.correct = correct;
        self.selected = 0;
        self.answered = false;
        self.shown_at = Some(Instant::now());
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

        if self.intro {
            if is_confirm(code) {
                self.dismiss_intro(ctx);
            }
            return Transition::Stay;
        }
        if self.picking {
            self.handle_pick(code, ctx);
            return Transition::Stay;
        }

        if self.finished() {
            if is_confirm(code) {
                self.picking = true;
                self.groups = build_groups(&self.gate);
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

    fn dismiss_intro(&mut self, ctx: &Ctx<'_>) {
        self.intro = false;
        self.picking = true;
        if let Some(pid) = ctx.profile_id {
            if let Err(e) = ctx.store.set_kana_onboarded(pid) {
                self.error = Some(e.to_string());
            }
        }
    }

    fn handle_pick(&mut self, code: KeyCode, ctx: &Ctx<'_>) {
        match code {
            KeyCode::Up | KeyCode::Char('k') => {
                self.group_cursor = self.group_cursor.saturating_sub(1);
            }
            KeyCode::Down | KeyCode::Char('j') => {
                if self.group_cursor + 1 < self.groups.len() {
                    self.group_cursor += 1;
                }
            }
            KeyCode::Left | KeyCode::Char('h') | KeyCode::Right | KeyCode::Char('l') => {
                self.reverse = !self.reverse; // binary; either arrow flips it
            }
            _ => {}
        }
        if is_confirm(code) && !self.groups[self.group_cursor].locked {
            self.start_session();
            let _ = ctx; // no persistence at session start
        }
    }

    fn answer_key(&mut self, code: KeyCode, ctx: &Ctx<'_>) {
        match code {
            KeyCode::Up | KeyCode::Char('k') => {
                self.selected = self.selected.saturating_sub(1);
            }
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
            if let Err(e) = ctx.store.add_xp(pid, study::xp_for_answer(correct)) {
                self.error = Some(e.to_string());
            }
        }
        self.record_answer(correct, ctx);
    }

    fn record_answer(&mut self, correct: bool, ctx: &Ctx<'_>) {
        let elapsed = self.shown_at.map(|t| t.elapsed()).unwrap_or_default();
        let char = self.deck[self.index].char.clone();
        let mut p = self.progress.get(&char).cloned().unwrap_or_default();
        p.char = char.clone();
        p = study::grade_kana(p, correct, elapsed);
        if let Some(pid) = ctx.profile_id {
            if let Err(e) = ctx.store.save_kana_progress(pid, &p) {
                self.error = Some(e.to_string());
            }
        }
        self.progress.insert(char, p);
        self.gate = study::new_gate(&self.items, &self.progress);
    }

    pub fn render(&self, f: &mut Frame, inner: Rect, theme: &Theme, msgs: &Messages) {
        let (lines, wrap) = if self.intro {
            (self.intro_lines(theme, msgs), true)
        } else if self.picking {
            (self.picker_lines(theme, msgs), false)
        } else if self.finished() {
            (self.summary_lines(theme, msgs), false)
        } else {
            (self.question_lines(inner, theme, msgs), false)
        };
        let mut para = Paragraph::new(lines);
        if wrap {
            para = para.wrap(Wrap { trim: true });
        }
        f.render_widget(para, inner);
    }

    fn intro_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let mut lines = vec![
            Line::styled(msgs.kana_intro_title.clone(), theme.title),
            Line::raw(""),
        ];
        for l in msgs.kana_intro_body.split('\n') {
            lines.push(Line::styled(l.to_string(), theme.normal));
        }
        lines.push(Line::raw(""));
        lines.push(Line::styled(msgs.kana_intro_help.clone(), theme.help));
        lines
    }

    fn picker_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let mut lines = vec![
            Line::styled(msgs.kana_title.clone(), theme.title),
            Line::raw(""),
            Line::styled(
                fmt1(&msgs.kana_direction_fmt, &self.direction_label(msgs)),
                theme.subtle,
            ),
            Line::raw(""),
        ];
        for (i, g) in self.groups.iter().enumerate() {
            let label = format!("{}{}", g.label, self.group_suffix(g, msgs));
            let line = if g.locked {
                Line::styled(format!("⊘ {label}"), theme.subtle)
            } else if i == self.group_cursor {
                Line::styled(format!("▸ {label}"), theme.selected)
            } else {
                Line::styled(format!("  {label}"), theme.normal)
            };
            lines.push(line);
        }
        lines.push(Line::raw(""));
        if self.groups[self.group_cursor].locked {
            let hint = fmt2(
                &msgs.kana_unlock_hint_fmt,
                self.gate.hiragana.mastered,
                self.gate.hiragana.total,
            );
            lines.push(Line::styled(hint, theme.subtle));
        } else {
            lines.push(Line::styled(msgs.kana_pick_help.clone(), theme.help));
        }
        lines.push(Line::styled(msgs.kana_mastery_note.clone(), theme.subtle));
        lines
    }

    fn direction_label(&self, msgs: &Messages) -> String {
        if self.reverse {
            msgs.kana_dir_reverse.clone()
        } else {
            msgs.kana_dir_forward.clone()
        }
    }

    fn group_suffix(&self, g: &Group, msgs: &Messages) -> String {
        let (mut total, mut mastered) = (0i64, 0i64);
        for it in &self.items {
            if !g.matcher.matches(it) {
                continue;
            }
            total += 1;
            if self.progress.get(&it.char).is_some_and(|p| p.mastered) {
                mastered += 1;
            }
        }
        if total == 0 {
            String::new()
        } else if mastered >= total {
            format!("  ✓ {}", msgs.kana_fluent)
        } else {
            format!("  {}", fmt2(&msgs.kana_mastered_fmt, mastered, total))
        }
    }

    fn question_lines<'a>(&self, inner: Rect, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let mut lines = vec![Line::styled(
            format!(
                "{}  {}/{}",
                msgs.kana_title,
                self.index + 1,
                self.deck.len()
            ),
            theme.title,
        )];
        lines.push(Line::raw(""));

        let stimulus = if self.reverse {
            self.deck[self.index].romaji.clone()
        } else {
            self.deck[self.index].char.clone()
        };
        for tile_line in big_tile(&stimulus, inner.width) {
            lines.push(Line::styled(tile_line, theme.accent));
        }
        lines.push(Line::raw(""));

        let prompt = if self.reverse {
            msgs.kana_prompt_reverse.clone()
        } else {
            msgs.kana_prompt.clone()
        };
        lines.push(Line::styled(prompt, theme.normal));
        lines.push(Line::raw(""));
        for (i, opt) in self.options.iter().enumerate() {
            lines.push(self.option_line(i, opt, theme));
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

    fn option_line<'a>(&self, i: usize, opt: &str, theme: &Theme) -> Line<'a> {
        let (mark, style) = if self.answered && i == self.correct {
            ("✓", theme.success)
        } else if self.answered && i == self.selected {
            ("✗", theme.error)
        } else if i == self.selected {
            ("▸", theme.selected)
        } else {
            (" ", theme.normal)
        };
        Line::styled(format!("{mark} {}) {opt}", i + 1), style)
    }

    fn summary_lines<'a>(&self, theme: &Theme, msgs: &Messages) -> Vec<Line<'a>> {
        let mut lines = vec![
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
        ];
        if self.gate.reading_unlocked() {
            lines.push(Line::styled(
                format!("✓ {}", msgs.fluent_badge),
                theme.success,
            ));
            lines.push(Line::raw(""));
        }
        lines.push(Line::styled(msgs.restart_help.clone(), theme.help));
        lines
    }
}

fn build_groups(gate: &Gate) -> Vec<Group> {
    let cat = |typ: KanaType, cats: Vec<KanaCategory>, label: &str| {
        let syllabary = match typ {
            KanaType::Hiragana => "Hiragana",
            KanaType::Katakana => "Katakana",
        };
        Group {
            label: format!("{syllabary} · {label}"),
            locked: typ == KanaType::Katakana && !gate.katakana_unlocked(),
            matcher: GroupMatch::Cat(typ, cats),
        }
    };
    vec![
        Group {
            label: "Todo".to_string(),
            locked: !gate.katakana_unlocked(),
            matcher: GroupMatch::All,
        },
        cat(KanaType::Hiragana, vec![KanaCategory::Base], "Básico"),
        cat(
            KanaType::Hiragana,
            vec![KanaCategory::Dakuten, KanaCategory::Handakuten],
            "Dakuten / Handakuten",
        ),
        cat(
            KanaType::Hiragana,
            vec![KanaCategory::Combo],
            "Combinaciones",
        ),
        cat(KanaType::Katakana, vec![KanaCategory::Base], "Básico"),
        cat(
            KanaType::Katakana,
            vec![KanaCategory::Dakuten, KanaCategory::Handakuten],
            "Dakuten / Handakuten",
        ),
        cat(
            KanaType::Katakana,
            vec![KanaCategory::Combo],
            "Combinaciones",
        ),
    ]
}

fn is_confirm(code: KeyCode) -> bool {
    matches!(code, KeyCode::Enter | KeyCode::Char(' '))
}

/// Renders a stimulus as a large bordered focal tile, centered in `width`. A
/// terminal cannot change the font size, so prominence comes from a wide border
/// and generous padding (2 rows, 6 columns), mirroring the Go tile.
fn big_tile(stimulus: &str, width: u16) -> Vec<String> {
    const PAD_H: usize = 6;
    const PAD_V: usize = 2;
    let sw = display_width(stimulus);
    let inner_w = sw + PAD_H * 2;
    let lead = " ".repeat((width as usize).saturating_sub(inner_w + 2) / 2);

    let border = |l: &str, r: &str| format!("{lead}{l}{}{r}", "─".repeat(inner_w));
    let blank = format!("{lead}│{}│", " ".repeat(inner_w));
    let glyph = format!(
        "{lead}│{}{stimulus}{}│",
        " ".repeat(PAD_H),
        " ".repeat(inner_w - sw - PAD_H)
    );

    let mut rows = vec![border("╭", "╮")];
    rows.extend(std::iter::repeat_n(blank.clone(), PAD_V));
    rows.push(glyph);
    rows.extend(std::iter::repeat_n(blank, PAD_V));
    rows.push(border("╰", "╯"));
    rows
}

fn display_width(s: &str) -> usize {
    s.chars()
        .map(|c| {
            let u = c as u32;
            if (0x3040..=0x30FF).contains(&u) || (0x4E00..=0x9FFF).contains(&u) {
                2
            } else {
                1
            }
        })
        .sum()
}

/// Substitutes the first `%s` in a format template.
fn fmt1(template: &str, a: &str) -> String {
    template.replacen("%s", a, 1)
}

/// Substitutes the first two `%d` placeholders in a format template.
fn fmt2(template: &str, a: i64, b: i64) -> String {
    template
        .replacen("%d", &a.to_string(), 1)
        .replacen("%d", &b.to_string(), 1)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn store_with_profile() -> (SqliteStore, i64) {
        let s = SqliteStore::open_in_memory().unwrap();
        let p = s.create_profile("A").unwrap();
        s.set_kana_onboarded(p.id).unwrap(); // skip the intro in tests
        (s, p.id)
    }

    fn course() -> Course {
        polyglot_core::content::load_embedded(polyglot_core::content::DEFAULT_PAIR).unwrap()
    }

    #[test]
    fn opens_on_picker_after_onboarding() {
        let (store, pid) = store_with_profile();
        let trainer = KanaTrainer::new(&store, &course(), Some(pid));
        assert!(!trainer.intro);
        assert!(trainer.picking);
        assert!(!trainer.groups.is_empty());
        // "Todo" (All) is gated until hiragana is fluent; hiragana base never is.
        assert!(trainer.groups[0].locked, "All is gated for a new learner");
        assert!(
            !trainer.groups[1].locked,
            "hiragana base is always available"
        );
    }

    #[test]
    fn answering_grades_and_persists_progress() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut trainer = KanaTrainer::new(&store, &course(), Some(pid));
        // Pick the hiragana base group (index 1) and start.
        trainer.group_cursor = 1;
        trainer.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert!(!trainer.picking, "session started");
        assert!(!trainer.deck.is_empty());

        // Answer the first question (whatever is selected), then it is revealed.
        trainer.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert!(trainer.answered);
        // Progress for the current kana was persisted.
        let saved = store.get_kana_progress(pid).unwrap();
        assert!(!saved.is_empty(), "an answer persisted kana progress");
    }

    #[test]
    fn big_tile_is_a_padded_centered_box() {
        let rows = big_tile("あ", 40);
        assert_eq!(rows.len(), 7, "top + 2 pad + glyph + 2 pad + bottom");
        assert!(rows[0].contains('╭') && rows[0].contains('╮'));
        assert!(rows[3].contains('あ'), "the glyph sits in the middle row");
        assert!(rows[6].contains('╰') && rows[6].contains('╯'));
        // Centered: the box is indented from the left edge.
        assert!(rows[0].starts_with(' '));
    }
}
