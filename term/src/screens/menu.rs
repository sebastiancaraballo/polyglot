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
    pub learned: i64,
    pub total: i64,
    pub reading_locked: bool,
    pub rikai_locked: bool,
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

        let Some(pid) = profile_id else {
            return Ok(Summary {
                total,
                reading_locked: true,
                rikai_locked: true,
                assessment_locked: true,
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
                ],
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
        let [content, help_area] =
            Layout::vertical([Constraint::Min(0), Constraint::Length(1)]).areas(inner);

        // Show the block wordmark when it fits horizontally and leaves room for
        // the globe/info columns below it; otherwise the info column keeps the
        // plain-text title (the Go fallback).
        let show_wordmark = inner.width >= 55 && content.height >= 13;
        let cols = if show_wordmark {
            let [wm, _gap, rest] = Layout::vertical([
                Constraint::Length(4),
                Constraint::Length(1),
                Constraint::Min(0),
            ])
            .areas(content);
            f.render_widget(Paragraph::new(art::WORDMARK).style(theme.title), wm);
            rest
        } else {
            content
        };

        let info = self.info_lines(theme, msgs, show_wordmark);
        // Draw the rotating globe beside the info column when there is room.
        if cols.width >= 44 {
            let [globe, _gap, info_area] = Layout::horizontal([
                Constraint::Length(16),
                Constraint::Length(7),
                Constraint::Min(0),
            ])
            .areas(cols);
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
        if show_name {
            lines.push(Line::styled(
                format!("v{} · {}", self.version, msgs.tagline),
                theme.subtle,
            ));
        } else {
            lines.push(Line::styled(
                format!("{}  v{}", msgs.app_name, self.version),
                theme.title,
            ));
            lines.push(Line::styled(msgs.tagline.clone(), theme.subtle));
        }
        lines.push(Line::raw(""));
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
        lines.push(Line::styled(
            format!(
                "{}/{} {}",
                self.summary.learned, self.summary.total, msgs.learned_suffix
            ),
            theme.subtle,
        ));
        lines.push(Line::raw(""));

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
}
