//! The kana reference chart: a browsable gojūon table across six pages.
//!
//! Port of the Go `internal/screens/kanachart`. A pure reference screen (no
//! storage). The grid always shows the five a·i·u·e·o columns so they line up
//! across pages, with a consonant group per row.

use polyglot_core::content::Course;
use polyglot_core::i18n::Messages;
use polyglot_core::model::{KanaCategory, KanaItem, KanaType};
use ratatui::crossterm::event::{KeyCode, KeyModifiers};
use ratatui::layout::Rect;
use ratatui::text::Line;
use ratatui::widgets::Paragraph;
use ratatui::Frame;

use crate::app::Transition;
use crate::theme::Theme;

/// The gojūon column order.
const VOWELS: [&str; 5] = ["a", "i", "u", "e", "o"];
/// Fixed display width of every column, so columns align across pages.
const COL_WIDTH: usize = 12;

struct Page {
    title: String,
    items: Vec<KanaItem>,
}

pub struct KanaChart {
    pages: Vec<Page>,
    page: usize,
}

impl KanaChart {
    /// Builds the chart, precomputing each page's title and items.
    pub fn new(course: &Course) -> KanaChart {
        // (syllabary, categories, title-key) in fixed left-to-right order.
        let defs: [(KanaType, &[KanaCategory], TitleKey); 6] = [
            (KanaType::Hiragana, &[KanaCategory::Base], TitleKey::Basic),
            (
                KanaType::Hiragana,
                &[KanaCategory::Dakuten, KanaCategory::Handakuten],
                TitleKey::Voiced,
            ),
            (KanaType::Hiragana, &[KanaCategory::Combo], TitleKey::Combo),
            (KanaType::Katakana, &[KanaCategory::Base], TitleKey::Basic),
            (
                KanaType::Katakana,
                &[KanaCategory::Dakuten, KanaCategory::Handakuten],
                TitleKey::Voiced,
            ),
            (KanaType::Katakana, &[KanaCategory::Combo], TitleKey::Combo),
        ];
        let pages = defs
            .iter()
            .map(|(typ, cats, key)| Page {
                title: page_title(*typ, *key),
                items: course
                    .kana
                    .iter()
                    .filter(|it| it.kana_type == *typ && cats.contains(&it.category))
                    .cloned()
                    .collect(),
            })
            .collect();
        KanaChart { pages, page: 0 }
    }

    pub fn handle(
        &mut self,
        code: KeyCode,
        mods: KeyModifiers,
        _ctx: &crate::app::Ctx<'_>,
    ) -> Transition {
        match code {
            KeyCode::Char('c') if mods.contains(KeyModifiers::CONTROL) => Transition::Quit,
            KeyCode::Esc => Transition::Pop,
            KeyCode::Left | KeyCode::Char('h') => {
                self.page = self.page.saturating_sub(1);
                Transition::Stay
            }
            KeyCode::Right | KeyCode::Char('l') => {
                if self.page + 1 < self.pages.len() {
                    self.page += 1;
                }
                Transition::Stay
            }
            _ => Transition::Stay,
        }
    }

    pub fn render(&self, f: &mut Frame, inner: Rect, theme: &Theme, msgs: &Messages) {
        let page = &self.pages[self.page];
        let mut lines: Vec<Line> = vec![
            Line::styled(
                format!(
                    "{}   ‹ {}/{} ›",
                    page.title,
                    self.page + 1,
                    self.pages.len()
                ),
                theme.title,
            ),
            Line::raw(""),
        ];
        lines.extend(grid_lines(&page.items, theme));

        let body = Rect {
            height: inner.height.saturating_sub(1),
            ..inner
        };
        let help_area = Rect {
            y: inner.y + inner.height.saturating_sub(1),
            height: 1,
            ..inner
        };
        f.render_widget(Paragraph::new(lines), body);
        f.render_widget(
            Paragraph::new(Line::styled(msgs.kana_chart_help.clone(), theme.help)),
            help_area,
        );
    }
}

#[derive(Clone, Copy)]
enum TitleKey {
    Basic,
    Voiced,
    Combo,
}

fn page_title(typ: KanaType, key: TitleKey) -> String {
    let syllabary = match typ {
        KanaType::Hiragana => "Hiragana",
        KanaType::Katakana => "Katakana",
    };
    let cat = match key {
        TitleKey::Basic => "Básico",
        TitleKey::Voiced => "Dakuten / Handakuten",
        TitleKey::Combo => "Combinaciones",
    };
    format!("{syllabary} · {cat}")
}

/// The trailing vowel column (0..=4) of a romaji, or `None` for ん/ン.
fn col_of(romaji: &str) -> Option<usize> {
    let last = romaji.chars().last()?;
    VOWELS.iter().position(|v| *v == last.to_string())
}

/// Arranges items into the traditional gojūon table: a new row begins whenever
/// the vowel returns to the first column. Returns which columns are used (for
/// the header) and the rows; ん/ン becomes a final row.
fn gojuon_rows(items: &[KanaItem]) -> ([bool; 5], Vec<[Option<KanaItem>; 5]>) {
    let mut present = [false; 5];
    let mut rows: Vec<[Option<KanaItem>; 5]> = Vec::new();
    let mut row: Option<[Option<KanaItem>; 5]> = None;
    let mut n_item: Option<KanaItem> = None;

    for it in items {
        let Some(c) = col_of(&it.romaji) else {
            n_item = Some(it.clone());
            continue;
        };
        present[c] = true;
        if c == 0 && row.is_some() {
            rows.push(row.take().unwrap());
        }
        row.get_or_insert_with(Default::default)[c] = Some(it.clone());
    }
    if let Some(r) = row.take() {
        rows.push(r);
    }
    if let Some(n) = n_item {
        let mut last: [Option<KanaItem>; 5] = Default::default();
        last[0] = Some(n);
        rows.push(last);
    }
    (present, rows)
}

fn grid_lines<'a>(items: &[KanaItem], theme: &Theme) -> Vec<Line<'a>> {
    if items.is_empty() {
        return Vec::new();
    }
    let (present, rows) = gojuon_rows(items);

    let mut lines = Vec::new();
    // Header row of vowels in the used columns.
    let mut header = String::new();
    for (i, v) in VOWELS.iter().enumerate() {
        header.push_str(&pad(if present[i] { v } else { "" }, COL_WIDTH));
    }
    lines.push(Line::styled(header, theme.subtle));

    for row in &rows {
        let mut cells = String::new();
        for cell in row {
            let text = match cell {
                Some(it) => format!("{} {}", it.char, it.romaji),
                None => String::new(),
            };
            cells.push_str(&pad(&text, COL_WIDTH));
        }
        lines.push(Line::styled(cells, theme.normal));
        lines.push(Line::raw("")); // uniform blank line between rows
    }
    lines
}

/// Pads `s` with spaces to `width` display cells (kana count as two).
fn pad(s: &str, width: usize) -> String {
    let cur = display_width(s);
    if cur >= width {
        s.to_string()
    } else {
        format!("{s}{}", " ".repeat(width - cur))
    }
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::frame::draw_frame;
    use ratatui::backend::TestBackend;
    use ratatui::Terminal;

    #[test]
    fn renders_hiragana_base_first_page() {
        let course = polyglot_core::content::load_embedded(polyglot_core::content::DEFAULT_PAIR)
            .expect("embedded course");
        let chart = KanaChart::new(&course);
        assert_eq!(chart.pages.len(), 6);
        assert!(chart.pages[0].title.contains("Hiragana"));

        let msgs = polyglot_core::i18n::default();
        let theme = Theme::plain();
        let mut terminal = Terminal::new(TestBackend::new(80, 24)).unwrap();
        terminal
            .draw(|f| {
                let inner = draw_frame(f, &theme);
                chart.render(f, inner, &theme, msgs);
            })
            .unwrap();
        // The vowel header labels should be present on the base page.
        let buf = terminal.backend().buffer();
        let mut screen = String::new();
        for y in 0..buf.area.height {
            for x in 0..buf.area.width {
                screen.push_str(buf[(x, y)].symbol());
            }
        }
        assert!(screen.contains('あ'), "shows the あ kana");
    }
}
