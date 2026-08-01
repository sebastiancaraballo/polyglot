//! The main menu screen: grouped, drill-down navigation.
//!
//! Port of the Go `internal/screens/menu`. Shows the block wordmark header and
//! the rotating braille globe (from `crate::art`) beside the progress/info
//! column, falling back to a plain-text title on frames too narrow or short to
//! fit them — the same fallback the Go menu uses.

use std::collections::HashSet;

use polyglot_core::content::Course;
use polyglot_core::i18n::Messages;
use polyglot_core::model::Jlpt;
use polyglot_core::storage::{SqliteStore, StorageError};
use polyglot_core::study;
use ratatui::crossterm::event::{KeyCode, KeyModifiers};
use ratatui::layout::{Constraint, Layout, Rect};
use ratatui::text::Line;
use ratatui::widgets::Paragraph;
use ratatui::Frame;

use crate::app::{Dest, Transition};
use crate::art;
use crate::theme::Theme;

/// Marks a menu item gated behind kana fluency. A non-color symbol (never relies
/// on color alone) of single display width.
const LOCK_GLYPH: &str = "⊘";

/// The section value for the root of the menu (no category open).
const TOP_LEVEL: i32 = -1;

/// Progress data shown in the menu header and the gating flags for each
/// activity.
#[derive(Clone, Debug, Default)]
pub struct Summary {
    pub name: String,
    pub xp: i64,
    pub streak: i64,
    // Computed for parity with the Go `Summary`, but not shown in the menu
    // (the learned/total figure lives on the stats screen).
    #[allow(dead_code)]
    pub learned: i64,
    #[allow(dead_code)]
    pub total: i64,
    pub reading_locked: bool,
    pub rikai_locked: bool,
    /// The pair teaches kanji at all; hides the activity entirely when it does not.
    pub teaches_kanji: bool,
    pub kanji_locked: bool,
    pub assessment_locked: bool,
    pub assessment_passed: bool,
}

impl Summary {
    /// Builds the menu summary from the course and the active profile's
    /// progress, computing each activity's gate through the core engine. With no
    /// active profile (first run), everything is locked and counters are zero.
    pub fn build(
        store: &SqliteStore,
        course: &Course,
        profile_id: Option<i64>,
    ) -> Result<Summary, StorageError> {
        let total = (course.kana.len()
            + course.lessons.iter().map(|l| l.cards.len()).sum::<usize>())
            as i64;

        let teaches_kanji = !course.kanji.is_empty();
        let Some(pid) = profile_id else {
            return Ok(Summary {
                total,
                reading_locked: true,
                rikai_locked: true,
                assessment_locked: true,
                teaches_kanji,
                kanji_locked: true,
                ..Summary::default()
            });
        };

        let profile = store.get_profile(pid)?;
        let stats = store.get_stats(pid)?;
        let learned = store.count_learned_cards(pid)?;

        // Reading is locked only while nothing is decodable yet; once the learner
        // can read at least one word, the reading activities open and show the
        // growing decodable subset (matches the Go menu's gate).
        let kana_progress = store.get_kana_progress(pid)?;
        let decoder = study::Decoder::new(&course.kana, &kana_progress);
        let readable = course
            .lessons
            .iter()
            .flat_map(|l| &l.cards)
            .filter(|c| decoder.decodable(&c.jp))
            .count();

        // Kanji readings are written in kana, so reading kana fluently is a real
        // prerequisite and not an arbitrary gate: the same one that opens
        // word-level reading opens kanji.
        let kanji_locked = !study::new_gate(&course.kana, &kana_progress).reading_unlocked();

        let card_states = store.get_card_states(pid)?;
        let known: HashSet<String> = card_states
            .iter()
            .filter(|(_, s)| s.reps > 0)
            .map(|(id, _)| id.clone())
            .collect();
        let rikai_locked = !course
            .patterns
            .iter()
            .any(|p| study::pattern_ready(p, &known));

        let story_progress = store.get_story_progress(pid)?;
        let all_mastered = !course.chapters.is_empty()
            && course
                .chapters
                .iter()
                .all(|c| story_progress.get(&c.id).is_some_and(|sp| sp.mastered));
        let assessment_passed = store.get_assessment_result(pid, Jlpt::N5)?.passed;

        Ok(Summary {
            name: profile.name,
            xp: stats.xp,
            streak: stats.streak,
            learned,
            total,
            reading_locked: readable == 0,
            rikai_locked,
            teaches_kanji,
            kanji_locked,
            assessment_locked: !all_mastered,
            assessment_passed,
        })
    }
}

struct Item {
    icon: &'static str,
    label: String,
    dest: Option<Dest>,
    quit: bool,
    locked: bool,
    locked_msg: String,
    children: Vec<Item>,
}

impl Item {
    fn leaf(icon: &'static str, label: &str, dest: Dest) -> Item {
        Item {
            icon,
            label: label.to_string(),
            dest: Some(dest),
            quit: false,
            locked: false,
            locked_msg: String::new(),
            children: Vec::new(),
        }
    }

    fn gated(icon: &'static str, label: &str, dest: Dest, locked: bool, msg: &str) -> Item {
        Item {
            locked,
            locked_msg: msg.to_string(),
            ..Item::leaf(icon, label, dest)
        }
    }

    fn category(icon: &'static str, label: &str, children: Vec<Item>) -> Item {
        Item {
            icon,
            label: label.to_string(),
            dest: None,
            quit: false,
            locked: false,
            locked_msg: String::new(),
            children,
        }
    }
}

/// The main menu screen state.
pub struct Menu {
    summary: Summary,
    version: String,
    items: Vec<Item>,
    section: i32,
    cursor: i32,
    notice: String,
    // Header globe animation.
    animate: bool,
    frame: usize,
    holding: usize,
}

/// How many ticks the globe rests on frame 0 (facing Japan) between turns:
/// ~25s at a 160ms tick.
const REST_HOLD: usize = 156;

/// Display height of a globe frame (braille rows).
const GLOBE_HEIGHT: u16 = 8;

/// Rows the block wordmark occupies in the header: 4 glyph rows plus a gap.
const WORDMARK_BLOCK_HEIGHT: u16 = 5;

impl Menu {
    pub fn new(msgs: &Messages, summary: Summary, version: String) -> Menu {
        let assessment_label = if summary.assessment_passed {
            format!("{}  {}", msgs.item_assessment, msgs.assessment_passed_badge)
        } else {
            msgs.item_assessment.clone()
        };
        let items = vec![
            Item::category(
                "◆",
                &msgs.cat_learn,
                vec![
                    Item::leaf("あ", &msgs.item_kana, Dest::Kana),
                    Item::gated(
                        "▣",
                        &msgs.item_flashcards,
                        Dest::Flashcards,
                        summary.reading_locked,
                        &msgs.reading_locked,
                    ),
                    Item::gated(
                        "◧",
                        &msgs.item_rikai,
                        Dest::Rikai,
                        summary.rikai_locked,
                        &msgs.rikai_locked,
                    ),
                ]
                .into_iter()
                .chain(summary.teaches_kanji.then(|| {
                    Item::gated(
                        "漢",
                        &msgs.item_kanji,
                        Dest::Kanji,
                        summary.kanji_locked,
                        &msgs.kanji_locked,
                    )
                }))
                .collect(),
            ),
            Item::category(
                "◫",
                &msgs.cat_read,
                vec![
                    Item::leaf("▧", &msgs.item_story, Dest::Story),
                    Item::leaf("▦", &msgs.item_kana_chart, Dest::KanaChart),
                ],
            ),
            Item::category(
                "◉",
                &msgs.cat_evaluate,
                vec![
                    Item::leaf("♻", &msgs.item_review, Dest::Review),
                    Item::gated(
                        "✓",
                        &msgs.item_quiz,
                        Dest::Quiz,
                        summary.reading_locked,
                        &msgs.reading_locked,
                    ),
                    Item::gated(
                        "▨",
                        &assessment_label,
                        Dest::Assessment,
                        summary.assessment_locked,
                        &msgs.assessment_locked,
                    ),
                ],
            ),
            Item::category(
                "▩",
                &msgs.cat_tools,
                vec![Item::leaf("▤", &msgs.item_stats, Dest::Stats)],
            ),
            Item::leaf("⚙", &msgs.item_settings, Dest::Settings),
            Item {
                quit: true,
                ..Item::leaf("⏻", &msgs.item_quit, Dest::Settings)
            },
        ];
        Menu {
            summary,
            version,
            items,
            section: TOP_LEVEL,
            cursor: 1, // the profile switcher occupies cursor 0
            notice: String::new(),
            // Honor reduced-motion: keep the globe static (resting on Japan) when
            // color is disabled, which also keeps it readable.
            animate: !crate::theme::no_color() && crate::art::GLOBE_FRAMES.len() > 1,
            frame: 0,
            holding: 0,
        }
    }

    /// Advances the header globe one animation frame, pausing on the resting
    /// frame (Japan) between full turns.
    pub fn tick(&mut self) {
        if !self.animate {
            return;
        }
        if self.frame == 0 && self.holding < REST_HOLD {
            self.holding += 1;
            return;
        }
        self.frame = (self.frame + 1) % crate::art::GLOBE_FRAMES.len();
        if self.frame == 0 {
            self.holding = 0;
        }
    }

    fn current_items(&self) -> &[Item] {
        if self.section == TOP_LEVEL {
            &self.items
        } else {
            &self.items[self.section as usize].children
        }
    }

    fn cursor_offset(&self) -> i32 {
        if self.section == TOP_LEVEL {
            1
        } else {
            0
        }
    }

    fn max_cursor(&self) -> i32 {
        if self.section == TOP_LEVEL {
            self.items.len() as i32
        } else {
            self.current_items().len() as i32 - 1
        }
    }

    pub fn handle(
        &mut self,
        code: KeyCode,
        mods: KeyModifiers,
        _ctx: &crate::app::Ctx<'_>,
    ) -> Transition {
        match code {
            KeyCode::Char('c') if mods.contains(KeyModifiers::CONTROL) => Transition::Quit,
            KeyCode::Char('q') => Transition::Quit,
            KeyCode::Esc | KeyCode::Left | KeyCode::Char('h') | KeyCode::Backspace => {
                if self.section != TOP_LEVEL {
                    self.cursor = self.section + 1;
                    self.section = TOP_LEVEL;
                    self.notice.clear();
                }
                Transition::Stay
            }
            KeyCode::Up | KeyCode::Char('k') => {
                self.notice.clear();
                if self.cursor > 0 {
                    self.cursor -= 1;
                }
                Transition::Stay
            }
            KeyCode::Down | KeyCode::Char('j') => {
                self.notice.clear();
                if self.cursor < self.max_cursor() {
                    self.cursor += 1;
                }
                Transition::Stay
            }
            KeyCode::Enter | KeyCode::Char(' ') => self.choose(),
            _ => Transition::Stay,
        }
    }

    fn choose(&mut self) -> Transition {
        if self.section == TOP_LEVEL {
            if self.cursor == 0 {
                return Transition::Push(Dest::Profiles);
            }
            let idx = (self.cursor - 1) as usize;
            if !self.items[idx].children.is_empty() {
                self.section = idx as i32;
                self.cursor = 0;
                self.notice.clear();
                return Transition::Stay;
            }
            return self.activate(idx, None);
        }
        let sec = self.section as usize;
        let ci = self.cursor as usize;
        self.activate(sec, Some(ci))
    }

    fn activate(&mut self, idx: usize, child: Option<usize>) -> Transition {
        let it = match child {
            None => &self.items[idx],
            Some(ci) => &self.items[idx].children[ci],
        };
        let (locked, quit, dest) = (it.locked, it.quit, it.dest);
        let locked_msg = it.locked_msg.clone();
        if locked {
            self.notice = locked_msg;
            return Transition::Stay;
        }
        if quit {
            return Transition::Quit;
        }
        match dest {
            Some(d) => Transition::Push(d),
            None => Transition::Stay,
        }
    }

    pub fn render(&self, f: &mut Frame, inner: Rect, theme: &Theme, msgs: &Messages) {
        let help = if !self.notice.is_empty() {
            Line::styled(format!("{LOCK_GLYPH} {}", self.notice), theme.subtle)
        } else if self.section == TOP_LEVEL {
            Line::styled(msgs.menu_help.clone(), theme.help)
        } else {
            Line::styled(msgs.menu_help_sub.clone(), theme.help)
        };

        // Reserve the bottom row for the help/notice line.
        let [area, help_area] =
            Layout::vertical([Constraint::Min(0), Constraint::Length(1)]).areas(inner);

        // Show the block wordmark when it fits horizontally and leaves room for
        // the globe/info columns below it; otherwise the info column keeps the
        // plain-text title (the Go fallback).
        let show_globe = area.width >= 44;
        let column_height = |info_len: usize| {
            if show_globe {
                (info_len as u16).max(GLOBE_HEIGHT)
            } else {
                info_len as u16
            }
        };

        // The frame's content height is a fixed budget that does not grow with a
        // taller terminal, so a long level must drop the wordmark rather than
        // push its own rows out of view.
        let mut show_wordmark = area.width >= 55 && area.height >= 13;
        if show_wordmark {
            let with_wordmark = self.info_lines(theme, msgs, true).len();
            show_wordmark = WORDMARK_BLOCK_HEIGHT + column_height(with_wordmark) <= area.height;
        }
        let info = self.info_lines(theme, msgs, show_wordmark);

        // Measure the header block so it can be centered vertically in the frame
        // (matching the Go menu, which centers its content rather than anchoring
        // it to the top).
        let cols_h = column_height(info.len());
        let wm_h = if show_wordmark {
            WORDMARK_BLOCK_HEIGHT
        } else {
            0
        };
        let block_h = (wm_h + cols_h).min(area.height);
        let top_pad = area.height.saturating_sub(block_h) / 2;
        let block = Rect {
            y: area.y + top_pad,
            height: block_h,
            ..area
        };

        let cols = if show_wordmark {
            let [wm, _gap, rest] = Layout::vertical([
                Constraint::Length(4),
                Constraint::Length(1),
                Constraint::Min(0),
            ])
            .areas(block);
            f.render_widget(Paragraph::new(art::WORDMARK).style(theme.title), wm);
            rest
        } else {
            block
        };

        // Draw the rotating globe beside the info column when there is room:
        // the menu reads left-to-right, so the options lead and the art sits on
        // the right.
        if show_globe {
            let [info_area, _gap, globe] = Layout::horizontal([
                Constraint::Min(0),
                Constraint::Length(7),
                Constraint::Length(16),
            ])
            .areas(cols);
            // The globe is shorter than the options column, so center it against
            // that column rather than letting it hang from the top.
            let globe = Rect {
                y: globe.y + globe.height.saturating_sub(GLOBE_HEIGHT) / 2,
                height: GLOBE_HEIGHT.min(globe.height),
                ..globe
            };
            f.render_widget(
                Paragraph::new(art::GLOBE_FRAMES[self.frame]).style(theme.accent),
                globe,
            );
            f.render_widget(Paragraph::new(info), info_area);
        } else {
            f.render_widget(Paragraph::new(info), cols);
        }
        f.render_widget(Paragraph::new(help), help_area);
    }

    /// The info column: title/progress block plus the menu options. `show_name`
    /// is false when the block wordmark already carries the app name.
    fn info_lines<'a>(&self, theme: &Theme, msgs: &Messages, show_name: bool) -> Vec<Line<'a>> {
        let mut lines: Vec<Line> = Vec::new();
        // Without the block wordmark the plain-text app name is the screen's
        // title, so it stays on top; the version line rides with the stats at
        // the foot instead.
        if !show_name {
            lines.push(Line::styled(
                format!("{}  v{}", msgs.app_name, self.version),
                theme.title,
            ));
            lines.push(Line::styled(msgs.tagline.clone(), theme.subtle));
            lines.push(Line::raw(""));
        }

        if self.section == TOP_LEVEL {
            let name = if self.summary.name.is_empty() {
                msgs.profile_name_placeholder.clone()
            } else {
                self.summary.name.clone()
            };
            let (marker, style) = if self.cursor == 0 {
                ("▸ ", theme.selected)
            } else {
                ("  ", theme.normal)
            };
            lines.push(Line::styled(
                format!("{marker}⇄ {name} · {}", msgs.switch_profile),
                style,
            ));
            lines.push(Line::raw(""));
        }

        let offset = self.cursor_offset();
        for (i, it) in self.current_items().iter().enumerate() {
            let icon = if it.locked { LOCK_GLYPH } else { it.icon };
            let mut label = it.label.clone();
            if !it.children.is_empty() {
                label.push_str(" ›");
            }
            let selected = i as i32 + offset == self.cursor;
            let (prefix, style) = if selected {
                ("▸ ", theme.selected)
            } else if it.locked {
                ("  ", theme.subtle)
            } else {
                ("  ", theme.normal)
            };
            lines.push(Line::styled(format!("{prefix}{icon}  {label}"), style));
        }

        // Progress sits under the options: the menu is for choosing, the
        // numbers are context. The learned/total figure lives on the stats
        // screen (matching the Go menu).
        lines.push(Line::raw(""));
        if show_name {
            lines.push(Line::styled(
                format!("v{} · {}", self.version, msgs.tagline),
                theme.subtle,
            ));
        }
        lines.push(Line::styled(
            format!("★ {}: {}", msgs.xp_label, self.summary.xp),
            theme.subtle,
        ));
        lines.push(Line::styled(
            format!(
                "▲ {}: {} {}",
                msgs.streak_label, self.summary.streak, msgs.days_suffix
            ),
            theme.subtle,
        ));
        lines
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::frame::draw_frame;
    use ratatui::backend::TestBackend;
    use ratatui::Terminal;

    fn flatten(terminal: &Terminal<TestBackend>) -> String {
        let buf = terminal.backend().buffer();
        let area = buf.area;
        let mut out = String::new();
        for y in 0..area.height {
            for x in 0..area.width {
                out.push_str(buf[(area.x + x, area.y + y)].symbol());
            }
            out.push('\n');
        }
        out
    }

    #[test]
    fn renders_categories_and_progress() {
        let msgs = polyglot_core::i18n::default();
        let summary = Summary {
            name: "Yui".to_string(),
            xp: 42,
            streak: 3,
            learned: 5,
            total: 100,
            ..Default::default()
        };
        let menu = Menu::new(msgs, summary, "0.1.0".to_string());
        let theme = Theme::plain();

        let backend = TestBackend::new(80, 24);
        let mut terminal = Terminal::new(backend).unwrap();
        terminal
            .draw(|f| {
                let inner = draw_frame(f, &theme);
                menu.render(f, inner, &theme, msgs);
            })
            .unwrap();

        let screen = flatten(&terminal);
        // At this width the block wordmark carries the app name.
        assert!(screen.contains('█'), "shows the block wordmark");
        assert!(screen.contains("Aprender"), "shows the Aprender category");
        assert!(screen.contains("Evaluar"), "shows the Evaluar category");
        assert!(screen.contains("Yui"), "shows the profile name");
        assert!(screen.contains('★'), "shows the XP line");
    }

    #[test]
    fn narrow_frame_falls_back_to_text_title() {
        let msgs = polyglot_core::i18n::default();
        let menu = Menu::new(msgs, Summary::default(), "0.1.0".to_string());
        let theme = Theme::plain();
        // A short terminal drops the wordmark; the info column keeps the name.
        let backend = TestBackend::new(80, 12);
        let mut terminal = Terminal::new(backend).unwrap();
        terminal
            .draw(|f| {
                let inner = draw_frame(f, &theme);
                menu.render(f, inner, &theme, msgs);
            })
            .unwrap();
        assert!(
            flatten(&terminal).contains("Polyglot"),
            "text title fallback"
        );
    }

    fn ctx(store: &polyglot_core::storage::SqliteStore) -> crate::app::Ctx<'_> {
        crate::app::Ctx {
            store,
            profile_id: None,
        }
    }

    #[test]
    fn drills_into_category_and_back() {
        let msgs = polyglot_core::i18n::default();
        let store = polyglot_core::storage::SqliteStore::open_in_memory().unwrap();
        let c = ctx(&store);
        let mut menu = Menu::new(msgs, Summary::default(), "0.1.0".to_string());
        // cursor starts at 1 (first category "Aprender"); ENTER descends.
        assert!(matches!(
            menu.handle(KeyCode::Enter, KeyModifiers::NONE, &c),
            Transition::Stay
        ));
        assert_eq!(menu.section, 0);
        assert_eq!(menu.cursor, 0);
        // ESC returns to the top level, restoring the cursor to the category.
        menu.handle(KeyCode::Esc, KeyModifiers::NONE, &c);
        assert_eq!(menu.section, TOP_LEVEL);
        assert_eq!(menu.cursor, 1);
    }

    #[test]
    fn locked_item_shows_notice_instead_of_navigating() {
        let msgs = polyglot_core::i18n::default();
        let store = polyglot_core::storage::SqliteStore::open_in_memory().unwrap();
        let c = ctx(&store);
        let summary = Summary {
            reading_locked: true,
            ..Default::default()
        };
        let mut menu = Menu::new(msgs, summary, "0.1.0".to_string());
        // Descend into "Aprender", move to Flashcards (locked), activate.
        menu.handle(KeyCode::Enter, KeyModifiers::NONE, &c);
        menu.handle(KeyCode::Down, KeyModifiers::NONE, &c);
        let t = menu.handle(KeyCode::Enter, KeyModifiers::NONE, &c);
        assert!(matches!(t, Transition::Stay));
        assert!(!menu.notice.is_empty(), "a locked item sets the notice");
    }

    #[test]
    fn reading_gate_opens_once_something_is_decodable() {
        use polyglot_core::model::KanaProgress;
        let store = polyglot_core::storage::SqliteStore::open_in_memory().unwrap();
        let course = polyglot_core::content::load_embedded("es-ja").unwrap();
        let p = store.create_profile("A").unwrap();

        // Fresh profile: nothing is decodable yet, so reading is locked.
        let s = Summary::build(&store, &course, Some(p.id)).unwrap();
        assert!(s.reading_locked, "reading locked while nothing decodable");

        // Master every kana: words become decodable and reading unlocks.
        for k in &course.kana {
            store
                .save_kana_progress(
                    p.id,
                    &KanaProgress {
                        char: k.char.clone(),
                        mastered: true,
                        ..Default::default()
                    },
                )
                .unwrap();
        }
        let s = Summary::build(&store, &course, Some(p.id)).unwrap();
        assert!(
            !s.reading_locked,
            "reading unlocks once a word is decodable"
        );
    }

    /// Kanji readings are written in kana, so the activity stays locked until
    /// both syllabaries are fluent — the same gate that opens word reading.
    #[test]
    fn kanji_is_gated_on_kana_fluency() {
        use polyglot_core::model::KanaProgress;
        let store = polyglot_core::storage::SqliteStore::open_in_memory().unwrap();
        let course = polyglot_core::content::load_embedded("es-ja").unwrap();
        let p = store.create_profile("A").unwrap();

        let s = Summary::build(&store, &course, Some(p.id)).unwrap();
        assert!(s.teaches_kanji, "the course ships a kanji table");
        assert!(s.kanji_locked, "locked before kana fluency");

        // Master every kana: the gate opens.
        for k in &course.kana {
            store
                .save_kana_progress(
                    p.id,
                    &KanaProgress {
                        char: k.char.clone(),
                        mastered: true,
                        ..Default::default()
                    },
                )
                .unwrap();
        }
        let s = Summary::build(&store, &course, Some(p.id)).unwrap();
        assert!(!s.kanji_locked, "kana fluency unlocks kanji");
    }

    /// A pair that teaches no kanji does not show the activity at all — an
    /// entry that can never do anything is worse than no entry.
    #[test]
    fn kanji_activity_is_absent_when_the_pair_teaches_none() {
        let msgs = polyglot_core::i18n::default();
        let without = Menu::new(
            msgs,
            Summary {
                teaches_kanji: false,
                ..Default::default()
            },
            "test".to_string(),
        );
        let learn = &without.items[0];
        assert!(
            !learn.children.iter().any(|c| c.dest == Some(Dest::Kanji)),
            "no kanji entry without a kanji table"
        );

        let with = Menu::new(
            msgs,
            Summary {
                teaches_kanji: true,
                ..Default::default()
            },
            "test".to_string(),
        );
        assert!(
            with.items[0]
                .children
                .iter()
                .any(|c| c.dest == Some(Dest::Kanji)),
            "the entry appears once the pair teaches kanji"
        );
    }

    /// A menu over an in-memory store, for the key-handling tests.
    fn test_menu() -> Menu {
        let msgs = polyglot_core::i18n::default();
        let summary = Summary {
            name: "Sebastián".to_string(),
            xp: 1240,
            streak: 5,
            learned: 8,
            total: 20,
            ..Default::default()
        };
        Menu::new(msgs, summary, "test".to_string())
    }

    fn locked_menu() -> Menu {
        let msgs = polyglot_core::i18n::default();
        let summary = Summary {
            name: "Sebastián".to_string(),
            total: 20,
            reading_locked: true,
            ..Default::default()
        };
        Menu::new(msgs, summary, "test".to_string())
    }

    /// Positions the menu on the leaf that navigates to `dest`, descending into
    /// the category holding it.
    fn open_leaf(menu: &mut Menu, dest: Dest) {
        for (ci, cat) in menu.items.iter().enumerate() {
            for (li, child) in cat.children.iter().enumerate() {
                if child.dest == Some(dest) && !child.quit {
                    menu.section = ci as i32;
                    menu.cursor = li as i32;
                    return;
                }
            }
        }
        panic!("no leaf navigates to {dest:?}");
    }

    /// Renders the menu at `w`×`h` and flattens it to text.
    fn view_at(menu: &Menu, w: u16, h: u16) -> String {
        let msgs = polyglot_core::i18n::default();
        let theme = Theme::plain();
        let mut terminal = Terminal::new(TestBackend::new(w, h)).unwrap();
        terminal
            .draw(|f| {
                let inner = draw_frame(f, &theme);
                menu.render(f, inner, &theme, msgs);
            })
            .unwrap();
        flatten(&terminal)
    }

    #[test]
    fn navigation_moves_cursor() {
        let store = polyglot_core::storage::SqliteStore::open_in_memory().unwrap();
        let c = ctx(&store);
        let mut menu = test_menu();

        menu.handle(KeyCode::Down, KeyModifiers::NONE, &c);
        assert_eq!(menu.cursor, 2, "cursor after down");
        menu.handle(KeyCode::Up, KeyModifiers::NONE, &c);
        assert_eq!(menu.cursor, 1, "cursor after up");
    }

    /// The cursor never leaves the list, at either end or either level.
    #[test]
    fn cursor_is_clamped() {
        let store = polyglot_core::storage::SqliteStore::open_in_memory().unwrap();
        let c = ctx(&store);

        let mut menu = test_menu();
        menu.cursor = 0;
        menu.handle(KeyCode::Up, KeyModifiers::NONE, &c);
        assert_eq!(menu.cursor, 0, "up at the top stays");

        // Inside a category, down stops at the last child.
        let mut menu = test_menu();
        menu.section = 0;
        menu.cursor = menu.items[0].children.len() as i32 - 1;
        menu.handle(KeyCode::Down, KeyModifiers::NONE, &c);
        assert_eq!(
            menu.cursor,
            menu.items[0].children.len() as i32 - 1,
            "down clamps at the last child"
        );
    }

    /// The menu opens at the top level with the first category selected (the
    /// profile switcher occupies cursor 0).
    #[test]
    fn defaults_to_first_category() {
        let msgs = polyglot_core::i18n::default();
        let menu = test_menu();
        assert_eq!(menu.section, TOP_LEVEL);
        assert_eq!(menu.cursor, 1);
        let first = &menu.items[(menu.cursor - 1) as usize];
        assert!(!first.children.is_empty(), "the first item is a category");
        assert_eq!(first.label, msgs.cat_learn);
    }

    /// Left also leaves a category, and esc at the top level does nothing.
    #[test]
    fn back_navigation() {
        let store = polyglot_core::storage::SqliteStore::open_in_memory().unwrap();
        let c = ctx(&store);

        // Open the third category, then go back: the cursor lands on it again.
        let mut menu = test_menu();
        menu.section = 2;
        menu.cursor = 1;
        menu.handle(KeyCode::Esc, KeyModifiers::NONE, &c);
        assert_eq!(menu.section, TOP_LEVEL);
        assert_eq!(menu.cursor, 3, "restored to the category row");

        // Left arrow leaves the category too.
        let mut menu = test_menu();
        menu.section = 0;
        menu.handle(KeyCode::Left, KeyModifiers::NONE, &c);
        assert_eq!(menu.section, TOP_LEVEL);

        // Esc at the top level is a no-op.
        let mut menu = test_menu();
        let t = menu.handle(KeyCode::Esc, KeyModifiers::NONE, &c);
        assert!(matches!(t, Transition::Stay));
        assert_eq!(menu.section, TOP_LEVEL);
    }

    /// Both the Quit item and the `q` key quit.
    #[test]
    fn quit_item_and_key() {
        let store = polyglot_core::storage::SqliteStore::open_in_memory().unwrap();
        let c = ctx(&store);

        let mut menu = test_menu();
        menu.cursor = menu.items.len() as i32; // Quit is the last top-level leaf
        assert!(matches!(
            menu.handle(KeyCode::Enter, KeyModifiers::NONE, &c),
            Transition::Quit
        ));

        let mut menu = test_menu();
        assert!(matches!(
            menu.handle(KeyCode::Char('q'), KeyModifiers::NONE, &c),
            Transition::Quit
        ));
    }

    /// Enter and space both activate the selected leaf.
    #[test]
    fn enter_and_space_navigate() {
        let store = polyglot_core::storage::SqliteStore::open_in_memory().unwrap();
        let c = ctx(&store);

        for code in [KeyCode::Enter, KeyCode::Char(' ')] {
            let mut menu = test_menu();
            open_leaf(&mut menu, Dest::Kana);
            let t = menu.handle(code, KeyModifiers::NONE, &c);
            assert!(
                matches!(t, Transition::Push(Dest::Kana)),
                "{code:?} should navigate to Kana, got {t:?}"
            );
        }
    }

    /// The profile header row opens the profile switcher.
    #[test]
    fn profile_header_navigates_to_profiles() {
        let store = polyglot_core::storage::SqliteStore::open_in_memory().unwrap();
        let c = ctx(&store);
        let mut menu = test_menu();
        menu.cursor = 0;
        let t = menu.handle(KeyCode::Enter, KeyModifiers::NONE, &c);
        assert!(matches!(t, Transition::Push(Dest::Profiles)), "got {t:?}");
    }

    /// Icons are text symbols, never color-only or emoji.
    #[test]
    fn uses_text_symbols() {
        let menu = test_menu();
        let want = ["◆", "◫", "◉", "▩", "⚙", "⏻"];
        assert_eq!(menu.items.len(), want.len(), "top-level item count");
        for (i, icon) in want.iter().enumerate() {
            assert_eq!(&menu.items[i].icon, icon, "item {i} icon");
        }
    }

    /// The top level shows the profile and progress, and hides the activities
    /// that live one level down.
    #[test]
    fn top_level_shows_progress_but_not_submenu_items() {
        let msgs = polyglot_core::i18n::default();
        let content = view_at(&test_menu(), 80, 30);
        for want in [
            "Sebastián",
            &msgs.switch_profile,
            "1240",
            &msgs.cat_learn,
            &msgs.cat_evaluate,
        ] {
            assert!(content.contains(want), "view is missing {want:?}");
        }
        assert!(
            !content.contains(&msgs.item_kana),
            "activity labels belong to the submenu"
        );
    }

    /// A category shows its children, the back hint, and drops the profile row.
    #[test]
    fn submenu_shows_children() {
        let msgs = polyglot_core::i18n::default();
        let mut menu = test_menu();
        menu.section = 0; // Aprender
        let content = view_at(&menu, 80, 30);
        for want in [&msgs.item_kana, &msgs.item_flashcards, &msgs.item_rikai] {
            assert!(content.contains(want), "submenu is missing {want:?}");
        }
        assert!(
            !content.contains(&msgs.switch_profile),
            "the submenu drops the profile switcher"
        );
    }

    /// The block wordmark renders when it fits and is dropped when the level's
    /// list would push the frame's bottom border off-screen — even on a
    /// generously tall terminal, since the frame height is a fixed budget.
    #[test]
    fn wordmark_yields_to_a_long_level() {
        let wordmark_top = crate::art::WORDMARK.lines().next().unwrap();

        // The real grouped menu is short enough to keep the wordmark.
        let content = view_at(&test_menu(), 120, 100);
        assert!(
            content.contains(wordmark_top),
            "the grouped top level keeps the wordmark"
        );

        // Eleven rows no longer fit the wordmark on top of the fixed budget.
        let msgs = polyglot_core::i18n::default();
        let mut menu = test_menu();
        menu.items = (0..11)
            .map(|_| Item::leaf("▤", &msgs.item_stats, Dest::Stats))
            .collect();
        let content = view_at(&menu, 120, 100);
        assert!(
            !content.contains(wordmark_top),
            "a long level drops the wordmark rather than clipping the frame"
        );
        assert!(
            content.contains('╰'),
            "the frame's bottom border stays visible"
        );
    }

    /// A locked item shows the lock glyph instead of its own icon, and explains
    /// itself once activated.
    #[test]
    fn locked_item_is_marked_and_explained() {
        let msgs = polyglot_core::i18n::default();
        let store = polyglot_core::storage::SqliteStore::open_in_memory().unwrap();
        let c = ctx(&store);
        let mut menu = locked_menu();
        open_leaf(&mut menu, Dest::Flashcards); // lives under "Aprender"

        let content = view_at(&menu, 80, 30);
        assert_eq!(
            content.matches(LOCK_GLYPH).count(),
            1,
            "exactly the locked Flashcards row carries the lock"
        );
        assert!(
            !content.contains('▣'),
            "the locked item's own icon is replaced by the lock"
        );

        menu.handle(KeyCode::Enter, KeyModifiers::NONE, &c);
        let content = view_at(&menu, 80, 30);
        assert!(
            content.contains(&msgs.reading_locked),
            "activating a locked item explains why"
        );
    }

    /// With nothing gated, no item is locked.
    #[test]
    fn unlocked_menu_has_no_locks() {
        let menu = test_menu();
        for cat in &menu.items {
            for child in &cat.children {
                assert!(!child.locked, "item {:?} should be unlocked", child.label);
            }
        }
    }

    /// The globe rests on the frame facing Japan, then spins a full turn and
    /// comes back to rest.
    #[test]
    fn globe_rests_then_spins() {
        let mut menu = test_menu();
        menu.animate = true;
        menu.frame = 0;
        menu.holding = 0;

        // It holds on the resting frame for the whole rest interval.
        for _ in 0..REST_HOLD {
            menu.tick();
            assert_eq!(menu.frame, 0, "still resting");
        }
        // Then it starts turning.
        menu.tick();
        assert_eq!(menu.frame, 1, "leaves the resting frame");

        // A full turn returns to frame 0 and re-arms the rest.
        for _ in 1..crate::art::GLOBE_FRAMES.len() {
            menu.tick();
        }
        assert_eq!(menu.frame, 0, "back to the resting frame");
        assert_eq!(menu.holding, 0, "the rest interval is re-armed");
    }

    /// Reduced motion (or `NO_COLOR`) keeps the globe static.
    #[test]
    fn static_globe_never_advances() {
        let mut menu = test_menu();
        menu.animate = false;
        for _ in 0..(REST_HOLD + 10) {
            menu.tick();
        }
        assert_eq!(menu.frame, 0, "a static globe never advances");
    }
}
