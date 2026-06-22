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
			{"◇", msgs.ItemKana, nav.Kana, false},
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
	var b strings.Builder

	b.WriteString(m.header())
	b.WriteString("\n\n")
	b.WriteString(m.theme.Accent.Render("│") + "   " + m.msgs.MenuPrompt)
	b.WriteString("\n\n")

	for i, it := range m.items {
		line := fmt.Sprintf("%s  %s", it.icon, it.label)
		if i+1 == m.cursor {
			b.WriteString(m.theme.Selected.Render("▸ " + line))
		} else {
			b.WriteString(m.theme.Normal.Render("  " + line))
		}
		b.WriteString("\n")
	}

	b.WriteString(m.theme.Help.Render(m.msgs.MenuHelp))

	view := tea.NewView(ui.Frame(m.theme, m.width, m.height, b.String()))
	view.AltScreen = true
	return view
}

// header lays the rotating globe beside the app name, progress, and the active
// profile.
func (m Model) header() string {
	globe := m.theme.Accent.Render(art.GlobeFrames[m.frame])
	info := lipgloss.JoinVertical(lipgloss.Left,
		m.titleLine(),
		m.xpLine(),
		m.streakLine(),
		m.profileLine(),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, globe, "   ", info)
}

func (m Model) titleLine() string {
	name := m.theme.Title.Render(fmt.Sprintf("%s  v%s", m.msgs.AppName, m.version))
	return name + "  " + m.theme.Subtle.Render(m.msgs.Tagline)
}

func (m Model) xpLine() string {
	return m.theme.Subtle.Render(fmt.Sprintf("★ %s: %d", m.msgs.XPLabel, m.summary.XP))
}

func (m Model) streakLine() string {
	return m.theme.Subtle.Render(fmt.Sprintf("▲ %s: %d %s · ✓ %d/%d",
		m.msgs.StreakLabel, m.summary.Streak, m.msgs.DaysSuffix,
		m.summary.Learned, m.summary.Total))
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
