package profiles

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/avatar"
	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

// Deps are the dependencies the profile switcher needs.
type Deps struct {
	Theme    ui.Theme
	Msgs     i18n.Messages
	Store    storage.Storage
	ActiveID int64
}

// Model is the profile switcher. It lists known profiles plus one row for
// creating a new profile.
type Model struct {
	deps Deps

	profiles []model.Profile
	cursor   int
	err      error

	width, height int
}

// New builds a profile switcher and eagerly reads the profile list.
func New(deps Deps) Model {
	m := Model{deps: deps}
	profiles, err := deps.Store.ListProfiles(context.Background())
	if err != nil {
		m.err = err
		return m
	}
	m.profiles = profiles
	return m
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
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < m.lastCursor() {
			m.cursor++
		}
	}
	if ui.IsConfirmKey(msg) {
		return m.choose()
	}
	return m, nil
}

func (m Model) choose() (tea.Model, tea.Cmd) {
	if m.cursor == len(m.profiles) {
		return m, nav.CreateProfile()
	}
	return m, nav.SwitchProfile(m.profiles[m.cursor].ID)
}

func (m Model) lastCursor() int { return len(m.profiles) }

// View implements tea.Model.
func (m Model) View() tea.View {
	t := m.deps.Theme
	msgs := m.deps.Msgs

	var b strings.Builder
	b.WriteString(t.Title.Render(msgs.ProfilesTitle))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(t.Error.Render(m.err.Error()))
		b.WriteString("\n\n")
	} else if len(m.profiles) == 0 {
		b.WriteString(t.Subtle.Render(msgs.NoProfiles))
		b.WriteString("\n")
	} else {
		for i, p := range m.profiles {
			line := m.profileLine(p)
			if i == m.cursor {
				b.WriteString(t.Selected.Render("▸ " + line))
			} else {
				b.WriteString(t.Normal.Render("  " + line))
			}
			b.WriteString("\n")
		}
	}

	createLine := msgs.ProfileCreateNew
	if m.cursor == len(m.profiles) {
		b.WriteString(t.Selected.Render("▸ " + createLine))
	} else {
		b.WriteString(t.Normal.Render("  " + createLine))
	}
	b.WriteString("\n\n")
	b.WriteString(t.Help.Render(msgs.ProfilesHelp))

	view := tea.NewView(ui.Frame(t, m.width, m.height, b.String()))
	view.AltScreen = true
	return view
}

func (m Model) profileLine(p model.Profile) string {
	line := fmt.Sprintf("[%s] %s", avatar.InlineSpec(p.Avatar, p.Name), p.Name)
	if p.ID == m.deps.ActiveID {
		line += "  ● " + m.deps.Msgs.ActiveProfileLabel
	}
	return line
}
