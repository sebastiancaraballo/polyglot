package menu

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sebastiancaraballo/polyglot/internal/art"
	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

// Animation cadence for the header globe: it rests facing Japan, then spins a
// full turn, then rests again.
const (
	frameInterval = 160 * time.Millisecond
	restDuration  = 25 * time.Second
	columnGap     = 7
)

// restHold is how many ticks the globe pauses on the resting frame (Japan).
var restHold = int(restDuration / frameInterval)

// lockGlyph marks a menu item gated behind kana fluency. It is a non-color
// symbol (it must not rely on color alone) and single display width.
const lockGlyph = "⊘"

// animSeq hands every menu instance a distinct animation id so that a tick left
// in flight by a previous menu can't drive a newer one (which would speed the
// animation up after navigating away and back).
var animSeq int

// tickMsg advances the header globe animation. id ties it to the menu instance
// that scheduled it.
type tickMsg struct{ id int }

// Summary is the progress data shown in the menu header (XP, streak, and the
// number of words learned).
type Summary struct {
	Name    string
	XP      int
	Streak  int
	Learned int
	Total   int
	// ReadingLocked gates the reading activities (Flashcards, Quiz) behind kana
	// fluency: the Foundations decoding gate. See internal/study.Gate.
	ReadingLocked bool
}

type item struct {
	icon   string
	label  string
	screen nav.Screen
	quit   bool
	locked bool
}

// Model is the main menu screen.
type Model struct {
	theme   ui.Theme
	msgs    i18n.Messages
	summary Summary
	version string

	items  []item
	cursor int
	notice string // transient message, e.g. why a locked item can't be opened

	// Header globe animation.
	animate bool
	animID  int
	frame   int
	holding int

	width  int
	height int
}

// New builds a menu model.
func New(theme ui.Theme, msgs i18n.Messages, summary Summary, version string) Model {
	animSeq++
	return Model{
		theme:   theme,
		msgs:    msgs,
		summary: summary,
		version: version,
		cursor:  1,
		// Honor reduced-motion preferences: keep the globe static (resting on
		// Japan) when color is disabled, which also keeps it readable.
		animate: !ui.NoColor() && len(art.GlobeFrames) > 1,
		animID:  animSeq,
		items: []item{
			{"あ", msgs.ItemKana, nav.Kana, false, false},
			{"▦", msgs.ItemKanaChart, nav.KanaChart, false, false},
			{"▣", msgs.ItemFlashcards, nav.Flashcards, false, summary.ReadingLocked},
			{"♻", msgs.ItemReview, nav.Review, false, false},
			{"✓", msgs.ItemQuiz, nav.Quiz, false, summary.ReadingLocked},
			{"▤", msgs.ItemStats, nav.Stats, false, false},
			{"⚙", msgs.ItemSettings, nav.Settings, false, false},
			{"⏻", msgs.ItemQuit, nav.Menu, true, false},
		},
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.animate {
		return m.tick()
	}
	return nil
}

func (m Model) tick() tea.Cmd {
	id := m.animID
	return tea.Tick(frameInterval, func(time.Time) tea.Msg { return tickMsg{id: id} })
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		if !m.animate || msg.id != m.animID {
			return m, nil // a stale tick from an earlier menu instance; let it die.
		}
		m.step()
		return m, m.tick()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			m.notice = ""
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			m.notice = ""
			if m.cursor < len(m.items) {
				m.cursor++
			}
		}
		if ui.IsConfirmKey(msg) {
			return m.choose()
		}
	}
	return m, nil
}

// step advances the globe animation one frame, pausing on the resting frame
// (Japan) for restHold ticks between turns.
func (m *Model) step() {
	if m.frame == 0 && m.holding < restHold {
		m.holding++
		return
	}
	m.frame = (m.frame + 1) % len(art.GlobeFrames)
	if m.frame == 0 {
		m.holding = 0
	}
}

func (m Model) choose() (tea.Model, tea.Cmd) {
	if m.cursor == 0 {
		return m, nav.GoTo(nav.Profiles)
	}
	it := m.items[m.cursor-1]
	if it.locked {
		m.notice = m.msgs.ReadingLocked
		return m, nil
	}
	if it.quit {
		return m, tea.Quit
	}
	return m, nav.GoTo(it.screen)
}

// View implements tea.Model.
func (m Model) View() tea.View {
	view := tea.NewView(ui.Frame(m.theme, m.width, m.height, m.content()))
	view.AltScreen = true
	return view
}

func (m Model) content() string {
	wordmark := m.wordmark()
	main := m.mainColumns(wordmark == "")
	if wordmark != "" {
		// The full-width block wordmark spans the header above the globe/info
		// columns, standing in for the app name (which the columns then omit to
		// avoid repeating it), with a blank line separating it from the columns.
		// To keep the fixed-height frame's footer in view, that blank takes the
		// place of the columns' internal separator, which mainColumns drops here.
		main = lipgloss.JoinVertical(lipgloss.Left, wordmark, "", main)
	}
	// A transient notice replaces the help line (rather than adding a row) so the
	// fixed-height frame never pushes the footer out of view. Moving the cursor
	// clears it and restores the help text.
	help := m.theme.Help.Render(m.msgs.MenuHelp)
	if m.notice != "" {
		help = m.theme.Subtle.Render(lockGlyph + " " + m.notice)
	}
	contentHeight := ui.FrameContentHeight(m.theme, m.height)
	if contentHeight <= 0 {
		return main + "\n" + help
	}

	mainHeight := lipgloss.Height(main)
	helpHeight := lipgloss.Height(help)
	available := contentHeight - helpHeight
	if available <= mainHeight {
		return main + "\n" + help
	}

	topPad := (available - mainHeight) / 2
	bottomPad := available - mainHeight - topPad

	var b strings.Builder
	b.WriteString(strings.Repeat("\n", topPad))
	b.WriteString(main)
	b.WriteString(strings.Repeat("\n", bottomPad+1))
	b.WriteString(help)
	return b.String()
}

// wordmark renders the compact block-letter app name for the header, or "" when
// the frame is too narrow to fit it (in which case the columns keep the plain
// text title instead). A zero width means the size is still unknown, so we show
// the wordmark optimistically rather than flashing the fallback on first paint.
func (m Model) wordmark() string {
	contentWidth := ui.FrameContentWidth(m.theme, m.width)
	if m.width > 0 && contentWidth < lipgloss.Width(art.Wordmark) {
		return ""
	}
	return m.theme.Title.Render(art.Wordmark)
}

// mainColumns lays the rotating globe beside the app name, progress, active
// profile, and menu options. showName keeps the plain text title in the info
// column; it is dropped when the header already shows the block wordmark.
func (m Model) mainColumns(showName bool) string {
	globe := m.theme.Accent.Render(art.GlobeFrames[m.frame])
	rows := []string{
		m.titleLine(showName),
		m.xpLine(),
		m.streakLine(),
		m.profileLine(),
	}
	// Separate the progress block from the menu options. When the wordmark header
	// is shown it already provides that blank (above the columns), so it is
	// dropped here to keep the fixed-height frame's footer in view.
	if showName {
		rows = append(rows, "")
	}
	rows = append(rows, m.menuItems())
	info := lipgloss.JoinVertical(lipgloss.Left, rows...)
	height := max(lipgloss.Height(globe), lipgloss.Height(info))
	return lipgloss.JoinHorizontal(lipgloss.Top,
		centerBlockVertically(globe, height),
		m.columnGap(globe, info),
		centerBlockVertically(info, height),
	)
}

func (m Model) menuItems() string {
	var b strings.Builder
	for i, it := range m.items {
		// A locked item shows the lock glyph in place of its icon — a fixed-width
		// marker that never wraps the row (which would break the column layout).
		// The reason is explained in the footer when the learner opens it.
		icon := it.icon
		if it.locked {
			icon = lockGlyph
		}
		line := fmt.Sprintf("%s  %s", icon, it.label)
		switch {
		case i+1 == m.cursor:
			b.WriteString(m.theme.Selected.Render("▸ " + line))
		case it.locked:
			b.WriteString(m.theme.Subtle.Render("  " + line))
		default:
			b.WriteString(m.theme.Normal.Render("  " + line))
		}
		if i < len(m.items)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m Model) columnGap(globe, info string) string {
	gap := columnGap
	contentWidth := ui.FrameContentWidth(m.theme, m.width)
	if contentWidth > 0 {
		maxGap := contentWidth - lipgloss.Width(globe) - lipgloss.Width(info)
		if maxGap < 1 {
			maxGap = 1
		}
		gap = min(gap, maxGap)
	}
	return strings.Repeat(" ", gap)
}

func centerBlockVertically(content string, height int) string {
	contentHeight := lipgloss.Height(content)
	if height <= contentHeight {
		return content
	}

	width := lipgloss.Width(content)
	topPad := (height - contentHeight) / 2
	bottomPad := height - contentHeight - topPad
	blank := strings.Repeat(" ", width)

	lines := make([]string, 0, height)
	for range topPad {
		lines = append(lines, blank)
	}
	lines = append(lines, strings.Split(content, "\n")...)
	for range bottomPad {
		lines = append(lines, blank)
	}
	return strings.Join(lines, "\n")
}

func (m Model) titleLine(showName bool) string {
	if !showName {
		// The wordmark above already carries the app name; show only the version
		// and language pair here so the name isn't repeated.
		return m.theme.Subtle.Render(fmt.Sprintf("v%s · %s", m.version, m.msgs.Tagline))
	}
	name := m.theme.Title.Render(fmt.Sprintf("%s  v%s", m.msgs.AppName, m.version))
	return name + "  " + m.theme.Subtle.Render(m.msgs.Tagline)
}

func (m Model) xpLine() string {
	return m.theme.Subtle.Render(fmt.Sprintf("★ %s: %d", m.msgs.XPLabel, m.summary.XP))
}

func (m Model) streakLine() string {
	return m.theme.Subtle.Render(fmt.Sprintf("▲ %s: %d %s",
		m.msgs.StreakLabel, m.summary.Streak, m.msgs.DaysSuffix))
}

func (m Model) profileLine() string {
	name := m.summary.Name
	if name == "" {
		name = m.msgs.ProfileNamePlaceholder
	}
	marker := "  "
	if m.cursor == 0 {
		marker = "▸ "
	}
	content := fmt.Sprintf("%s⇄ %s · %s", marker, name, m.msgs.SwitchProfile)
	if m.cursor == 0 {
		return m.theme.Selected.Render(content)
	}
	return m.theme.Normal.Render(content)
}
