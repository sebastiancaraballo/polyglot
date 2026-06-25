package kanachart

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

// Deps are the dependencies the kana chart needs. It is a pure reference screen,
// so it needs no storage or profile.
type Deps struct {
	Theme ui.Theme
	Msgs  i18n.Messages
	Kana  []model.KanaItem
}

// pageDef describes one chart page: a syllabary and the categories shown on it.
type pageDef struct {
	typ  model.KanaType
	cats []model.KanaCategory
	cat  func(i18n.Messages) string
}

// pageDefs is the fixed left-to-right page order.
var pageDefs = []pageDef{
	{model.Hiragana, []model.KanaCategory{model.Base}, func(m i18n.Messages) string { return m.KanaBasic }},
	{model.Hiragana, []model.KanaCategory{model.Dakuten, model.Handakuten}, func(m i18n.Messages) string { return m.KanaVoiced }},
	{model.Hiragana, []model.KanaCategory{model.Combo}, func(m i18n.Messages) string { return m.KanaCombo }},
	{model.Katakana, []model.KanaCategory{model.Base}, func(m i18n.Messages) string { return m.KanaBasic }},
	{model.Katakana, []model.KanaCategory{model.Dakuten, model.Handakuten}, func(m i18n.Messages) string { return m.KanaVoiced }},
	{model.Katakana, []model.KanaCategory{model.Combo}, func(m i18n.Messages) string { return m.KanaCombo }},
}

// pageContent is a fully prepared page: its localized title and its items.
type pageContent struct {
	title string
	items []model.KanaItem
}

// Model is the kana reference chart screen.
type Model struct {
	deps  Deps
	pages []pageContent
	page  int

	width, height int
}

// New builds the chart, precomputing each page's title and items.
func New(deps Deps) Model {
	m := Model{deps: deps}
	for _, def := range pageDefs {
		m.pages = append(m.pages, pageContent{
			title: m.pageTitle(def),
			items: filter(deps.Kana, def),
		})
	}
	return m
}

func (m Model) pageTitle(def pageDef) string {
	syllabary := m.deps.Msgs.HiraganaLabel
	if def.typ == model.Katakana {
		syllabary = m.deps.Msgs.KatakanaLabel
	}
	return fmt.Sprintf("%s · %s", syllabary, def.cat(m.deps.Msgs))
}

// filter returns the items matching a page's syllabary and categories, preserving
// their content order.
func filter(items []model.KanaItem, def pageDef) []model.KanaItem {
	var out []model.KanaItem
	for _, it := range items {
		if it.Type != def.typ {
			continue
		}
		for _, c := range def.cats {
			if it.Category == c {
				out = append(out, it)
				break
			}
		}
	}
	return out
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		return m, nav.Back()
	case "left", "h":
		if m.page > 0 {
			m.page--
		}
	case "right", "l":
		if m.page < len(m.pages)-1 {
			m.page++
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	view := tea.NewView(ui.FitFrame(m.deps.Theme, m.width, m.height, m.content()))
	view.AltScreen = true
	return view
}

func (m Model) content() string {
	t := m.deps.Theme
	page := m.pages[m.page]

	header := fmt.Sprintf("%s   %s",
		t.Title.Render(page.title),
		t.Subtle.Render(fmt.Sprintf("‹ %d/%d ›", m.page+1, len(m.pages))),
	)
	help := t.Help.Render(m.deps.Msgs.KanaChartHelp)
	grid := m.grid(page.items)

	// The table sits just under the header with the help at the bottom; the frame
	// hugs the content instead of filling the screen. Every page's body is padded
	// to the tallest page's height (top-anchored) so the frame stays one constant
	// height as the user pages through the chart.
	body := lipgloss.PlaceVertical(m.bodyHeight(), lipgloss.Top, grid)
	return header + "\n\n" + body + "\n\n" + help
}

// bodyHeight is the display height of the tallest page's grid, used to pad every
// page's body to the same height so the frame does not grow and shrink as the user
// pages through the chart.
func (m Model) bodyHeight() int {
	h := 0
	for _, p := range m.pages {
		if gh := lipgloss.Height(m.grid(p.items)); gh > h {
			h = gh
		}
	}
	return h
}

// canonicalVowels is the gojūon column order.
var canonicalVowels = []string{"a", "i", "u", "e", "o"}

// vowelOf returns the trailing vowel of a kana's romaji (its column), or "" for
// ん/ン, which has no vowel.
func vowelOf(romaji string) string {
	if romaji == "" {
		return ""
	}
	switch last := romaji[len(romaji)-1:]; last {
	case "a", "i", "u", "e", "o":
		return last
	default:
		return ""
	}
}

// colOf returns the fixed column (0..4) for a romaji's vowel, or -1 for ん/ン.
func colOf(romaji string) int {
	v := vowelOf(romaji)
	for i, vv := range canonicalVowels {
		if vv == v {
			return i
		}
	}
	return -1
}

// gojuonRows arranges items into the traditional gojūon table. The grid always has
// the five a·i·u·e·o columns so columns line up across every page (combinations land
// under a/u/o), and each row is a consonant group. A new row begins whenever the
// vowel returns to the first column, which keeps irregular readings (shi, chi, tsu,
// fu, ji, zu) in their proper column and leaves gaps where a syllable has no
// counterpart (や ゆ よ, わ を). present reports which columns are used on this page
// (for the header); ん/ン is returned as a final row.
func gojuonRows(items []model.KanaItem) (present []bool, rows [][]model.KanaItem) {
	present = make([]bool, len(canonicalVowels))
	var (
		row   []model.KanaItem
		nItem *model.KanaItem
	)
	for i := range items {
		c := colOf(items[i].Romaji)
		if c < 0 {
			nItem = &items[i]
			continue
		}
		present[c] = true
		if c == 0 && row != nil {
			rows = append(rows, row)
			row = nil
		}
		if row == nil {
			row = make([]model.KanaItem, len(canonicalVowels))
		}
		row[c] = items[i]
	}
	if row != nil {
		rows = append(rows, row)
	}
	if nItem != nil {
		last := make([]model.KanaItem, len(canonicalVowels))
		last[0] = *nItem
		rows = append(rows, last)
	}
	return present, rows
}

func (m Model) cell(it model.KanaItem) string {
	if it.Char == "" {
		return ""
	}
	return m.deps.Theme.Accent.Render(it.Char) + " " + m.deps.Theme.Subtle.Render(it.Romaji)
}

// colWidth is the fixed width of every column, identical on every page so the
// columns sit at the same x positions across pages. It is the widest cell in the
// whole course, widened to fill the body so the five columns span the frame.
func (m Model) colWidth() int {
	w := 0
	for _, it := range m.deps.Kana {
		if cw := lipgloss.Width(m.cell(it)); cw > w {
			w = cw
		}
	}
	w += 2 // minimum gutter
	if fill := ui.FrameContentWidth(m.deps.Theme, m.width) / len(canonicalVowels); fill > w {
		w = fill
	}
	return w
}

// rowSep is the separator between table rows. It is the same on every page (so the
// interlineado is consistent) and is a blank line only when the tallest page would
// still fit the body spaced, so nothing is ever clipped.
func (m Model) rowSep() string {
	budget := ui.MaxFrameContentHeight(m.deps.Theme, m.height) - 2 // header + help
	if 2*m.tallestPage()-1 <= budget {
		return "\n\n"
	}
	return "\n"
}

// tallestPage returns the maximum number of table lines (header + rows) across all
// pages, so the row spacing can be decided uniformly.
func (m Model) tallestPage() int {
	max := 0
	for _, p := range m.pages {
		_, rows := gojuonRows(p.items)
		if n := len(rows) + 1; n > max { // +1 for the header row
			max = n
		}
	}
	return max
}

// grid renders the items as a fixed gojūon table (five aligned columns with a vowel
// header), centered with equal margins and with a uniform blank line between rows.
func (m Model) grid(items []model.KanaItem) string {
	t := m.deps.Theme
	if len(items) == 0 {
		return ""
	}
	present, rows := gojuonRows(items)

	colStyle := lipgloss.NewStyle().Width(m.colWidth())
	line := func(cells []string) string {
		var b strings.Builder
		for _, c := range cells {
			b.WriteString(colStyle.Render(c))
		}
		return b.String()
	}

	header := make([]string, len(canonicalVowels))
	for i, v := range canonicalVowels {
		if present[i] {
			header[i] = t.Subtle.Render(v)
		}
	}
	lines := []string{line(header)}
	for _, row := range rows {
		cells := make([]string, len(row))
		for i, it := range row {
			cells[i] = m.cell(it)
		}
		lines = append(lines, line(cells))
	}

	block := strings.Join(lines, m.rowSep())
	if cw := ui.FrameContentWidth(t, m.width); cw > 0 {
		block = lipgloss.PlaceHorizontal(cw, lipgloss.Center, block)
	}
	return block
}
