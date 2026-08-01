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
use crate::tile::big_tile;

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

    fn kana(ch: &str, romaji: &str, typ: KanaType, cat: KanaCategory) -> KanaItem {
        KanaItem {
            char: ch.to_string(),
            romaji: romaji.to_string(),
            kana_type: typ,
            category: cat,
        }
    }

    /// Wraps `kana` in an otherwise-empty course, so a trainer can be built over
    /// a tiny fixture instead of the whole embedded syllabary.
    fn course_of(kana: Vec<KanaItem>) -> Course {
        Course {
            pair: "es-xx".to_string(),
            lessons: Vec::new(),
            kana,
            kanji: Vec::new(),
            patterns: Vec::new(),
            chapters: Vec::new(),
        }
    }

    /// A minimal two-syllabary set whose base gojūon is one kana each, so a
    /// single mastered kana makes a syllabary fluent.
    fn gate_course() -> Course {
        course_of(vec![
            kana("あ", "a", KanaType::Hiragana, KanaCategory::Base),
            kana("ア", "a", KanaType::Katakana, KanaCategory::Base),
        ])
    }

    /// Five hiragana, enough to fill an option list in either direction.
    fn reverse_course() -> Course {
        course_of(vec![
            kana("あ", "a", KanaType::Hiragana, KanaCategory::Base),
            kana("い", "i", KanaType::Hiragana, KanaCategory::Base),
            kana("う", "u", KanaType::Hiragana, KanaCategory::Base),
            kana("え", "e", KanaType::Hiragana, KanaCategory::Base),
            kana("お", "o", KanaType::Hiragana, KanaCategory::Base),
        ])
    }

    fn group_index(trainer: &KanaTrainer, prefix: &str) -> usize {
        trainer
            .groups
            .iter()
            .position(|g| g.label.starts_with(prefix))
            .unwrap_or_else(|| panic!("no group labelled {prefix:?}"))
    }

    /// Renders the trainer and flattens it to text.
    fn view(trainer: &KanaTrainer) -> String {
        let msgs = polyglot_core::i18n::default();
        crate::testutil::snapshot(|f, inner, theme| trainer.render(f, inner, theme, msgs))
    }

    /// Confirming a group builds a deck restricted to it.
    #[test]
    fn picker_starts_filtered_session() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let course = course_of(vec![
            kana("あ", "a", KanaType::Hiragana, KanaCategory::Base),
            kana("が", "ga", KanaType::Hiragana, KanaCategory::Dakuten),
            kana("ぱ", "pa", KanaType::Hiragana, KanaCategory::Handakuten),
            kana("きゃ", "kya", KanaType::Hiragana, KanaCategory::Combo),
            kana("ア", "a", KanaType::Katakana, KanaCategory::Base),
        ]);
        let mut trainer = KanaTrainer::new(&store, &course, Some(pid));
        assert!(trainer.picking, "the trainer opens on the group picker");

        // Move to "Hiragana · Dakuten / Handakuten" and start.
        trainer.handle(KeyCode::Down, KeyModifiers::NONE, &ctx);
        trainer.handle(KeyCode::Down, KeyModifiers::NONE, &ctx);
        trainer.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);

        assert!(!trainer.picking, "confirming a group starts the session");
        assert_eq!(trainer.deck.len(), 2, "dakuten + handakuten");
        for it in &trainer.deck {
            assert!(
                matches!(
                    it.category,
                    KanaCategory::Dakuten | KanaCategory::Handakuten
                ),
                "out-of-group item {:?}",
                it.char
            );
        }
    }

    /// Space answers the current question, and a correct answer awards XP.
    #[test]
    fn space_answers_and_awards_xp() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut trainer = KanaTrainer::new(&store, &gate_course(), Some(pid));
        trainer.group_cursor = group_index(&trainer, "Hiragana");
        trainer.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);

        trainer.handle(KeyCode::Char(' '), KeyModifiers::NONE, &ctx);
        assert!(trainer.answered, "space answers the question");
        assert!(store.get_stats(pid).unwrap().xp > 0, "answering awards XP");
    }

    /// Katakana — and the all-syllabary group, which spans it — stay locked
    /// until hiragana is fluent, and confirming a locked group does nothing.
    #[test]
    fn katakana_and_all_groups_are_gated_on_hiragana() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let msgs = polyglot_core::i18n::default();

        let mut trainer = KanaTrainer::new(&store, &gate_course(), Some(pid));
        for label in [msgs.katakana_label.as_str(), msgs.kana_group_all.as_str()] {
            let idx = group_index(&trainer, label);
            assert!(trainer.groups[idx].locked, "{label} is locked at first");

            trainer.group_cursor = idx;
            trainer.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
            assert!(
                trainer.picking,
                "confirming the locked {label} group must not start a session"
            );
        }

        // Mastering the single hiragana opens both gates.
        store
            .save_kana_progress(
                pid,
                &KanaProgress {
                    char: "あ".to_string(),
                    mastered: true,
                    ..Default::default()
                },
            )
            .unwrap();
        let trainer = KanaTrainer::new(&store, &gate_course(), Some(pid));
        for label in [msgs.katakana_label.as_str(), msgs.kana_group_all.as_str()] {
            let idx = group_index(&trainer, label);
            assert!(
                !trainer.groups[idx].locked,
                "{label} unlocks once hiragana is fluent"
            );
        }
    }

    /// The locked-group hint shows live progress toward opening the gate.
    #[test]
    fn locked_group_hint_shows_live_progress() {
        let (store, pid) = store_with_profile();
        let msgs = polyglot_core::i18n::default();
        let mut trainer = KanaTrainer::new(&store, &gate_course(), Some(pid));
        trainer.group_cursor = group_index(&trainer, &msgs.katakana_label);

        // The fixture has one hiragana base kana, none mastered yet.
        assert!(
            view(&trainer).contains("0/1"),
            "the hint should show live hiragana progress"
        );
    }

    /// A first-time visitor sees the intro; dismissing it persists and is not
    /// shown again.
    #[test]
    fn intro_is_shown_once_and_persisted() {
        let store = SqliteStore::open_in_memory().unwrap();
        let pid = store.create_profile("A").unwrap().id; // not kana-onboarded
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };

        let mut trainer = KanaTrainer::new(&store, &gate_course(), Some(pid));
        assert!(trainer.intro, "a first-time visitor sees the intro");
        assert!(!trainer.picking, "the picker waits behind the intro");

        trainer.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        assert!(!trainer.intro, "confirming dismisses the intro");
        assert!(trainer.picking, "and falls through to the picker");
        assert!(
            store.get_profile(pid).unwrap().kana_onboarded,
            "dismissing persists that the intro was seen"
        );

        let again = KanaTrainer::new(&store, &gate_course(), Some(pid));
        assert!(!again.intro, "the intro does not reappear");
        assert!(again.picking);
    }

    /// ← and → flip the drill direction in the picker.
    #[test]
    fn arrows_toggle_direction_in_picker() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut trainer = KanaTrainer::new(&store, &reverse_course(), Some(pid));
        assert!(!trainer.reverse, "the trainer defaults to recognition");

        trainer.handle(KeyCode::Right, KeyModifiers::NONE, &ctx);
        assert!(trainer.reverse, "→ flips to the reverse direction");
        trainer.handle(KeyCode::Left, KeyModifiers::NONE, &ctx);
        assert!(!trainer.reverse, "← flips back");
    }

    /// In reverse the prompt is the romaji and every option is a glyph — the
    /// learner must produce the character, not recognize it.
    #[test]
    fn reverse_session_asks_for_the_glyph() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let course = reverse_course();
        let mut trainer = KanaTrainer::new(&store, &course, Some(pid));
        trainer.reverse = true;
        trainer.group_cursor = group_index(&trainer, "Hiragana");
        trainer.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);

        let glyphs: std::collections::HashSet<&str> =
            course.kana.iter().map(|k| k.char.as_str()).collect();
        for opt in &trainer.options {
            assert!(
                glyphs.contains(opt.as_str()),
                "reverse option {opt:?} is not a kana glyph"
            );
        }
        assert_eq!(
            trainer.options[trainer.correct], trainer.deck[trainer.index].char,
            "the correct option is the deck's glyph"
        );

        let msgs = polyglot_core::i18n::default();
        assert!(
            view(&trainer).contains(&msgs.kana_prompt_reverse),
            "the reverse question uses the recall prompt"
        );
    }

    /// Mastery is keyed by character, not by direction: a reverse answer
    /// advances the same per-character streak the forward drill uses.
    #[test]
    fn reverse_answer_records_mastery_by_character() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut trainer = KanaTrainer::new(&store, &reverse_course(), Some(pid));
        trainer.reverse = true;
        trainer.group_cursor = group_index(&trainer, "Hiragana");
        trainer.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);

        let answered_char = trainer.deck[trainer.index].char.clone();
        trainer.selected = trainer.correct;
        trainer.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);

        let saved = store.get_kana_progress(pid).unwrap();
        let p = saved
            .get(&answered_char)
            .unwrap_or_else(|| panic!("no progress recorded under {answered_char:?}"));
        assert!(p.attempts > 0, "the answer advanced the character's streak");
    }

    /// Revealing an answer must not shift the layout: the kana tile keeps its
    /// column so the glyph does not jump under the learner's eyes.
    #[test]
    fn tile_position_is_stable_after_answering() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        let mut trainer = KanaTrainer::new(&store, &reverse_course(), Some(pid));
        trainer.group_cursor = group_index(&trainer, "Hiragana");
        trainer.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);

        let tile_column = |v: &str| -> usize {
            v.lines()
                .find_map(|l| l.find('╭'))
                .expect("the view contains the kana tile")
        };
        let before = tile_column(&view(&trainer));
        trainer.selected = (trainer.correct + 1) % trainer.options.len();
        trainer.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
        let after = tile_column(&view(&trainer));

        assert_eq!(before, after, "the tile column moved after answering");
    }

    /// Every question state, in both directions, fits the fixed frame.
    #[test]
    fn question_view_fits_the_frame_in_both_directions() {
        let (store, pid) = store_with_profile();
        let ctx = Ctx {
            store: &store,
            profile_id: Some(pid),
        };
        for reverse in [false, true] {
            let mut trainer = KanaTrainer::new(&store, &reverse_course(), Some(pid));
            trainer.reverse = reverse;
            trainer.group_cursor = group_index(&trainer, "Hiragana");
            trainer.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);

            for answered in [false, true] {
                if answered {
                    trainer.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
                }
                // `snapshot` renders into the real frame; nothing may overflow it,
                // which shows up as content on the border rows/columns.
                let v = view(&trainer);
                let widest = v.lines().map(|l| l.chars().count()).max().unwrap_or(0);
                assert!(
                    widest <= 80,
                    "reverse={reverse} answered={answered}: line of {widest} cells overflows the terminal"
                );
                assert!(
                    v.lines().filter(|l| !l.trim().is_empty()).count() <= 30,
                    "reverse={reverse} answered={answered}: content overflows the terminal height"
                );
            }
        }
    }
}
