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
}

type item struct {
	icon   string
	label  string
	screen nav.Screen
	quit   bool
}

// Model is the main menu screen.
type Model struct {
	theme   ui.Theme
	msgs    i18n.Messages
	summary Summary
	version string

	items  []item
	cursor int

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
			{"あ", msgs.ItemKana, nav.Kana, false},
			{"▣", msgs.ItemFlashcards, nav.Flashcards, false},
			{"✓", msgs.ItemQuiz, nav.Quiz, false},
			{"▤", msgs.ItemStats, nav.Stats, false},
			{"⚙", msgs.ItemSettings, nav.Settings, false},
			{"⏻", msgs.ItemQuit, nav.Menu, true},
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
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
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
	main := m.mainColumns()
	help := m.theme.Help.Render(m.msgs.MenuHelp)
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

// mainColumns lays the rotating globe beside the app name, progress, active
// profile, and menu options.
func (m Model) mainColumns() string {
	globe := m.theme.Accent.Render(art.GlobeFrames[m.frame])
	info := lipgloss.JoinVertical(lipgloss.Left,
		m.titleLine(),
		m.xpLine(),
		m.streakLine(),
		m.profileLine(),
		"",
		m.menuItems(),
	)
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
		line := fmt.Sprintf("%s  %s", it.icon, it.label)
		if i+1 == m.cursor {
			b.WriteString(m.theme.Selected.Render("▸ " + line))
		} else {
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

func (m Model) titleLine() string {
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
