package settings

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

// Deps are the dependencies the settings screen needs. The screen performs no
// storage work itself: the destructive action emits nav.WipeData and the router
// (which owns the storage connection) carries it out.
type Deps struct {
	Theme ui.Theme
	Msgs  i18n.Messages
}

// Model is the settings screen. It shows a list of actions and, for the
// destructive "delete all app data" action, an explicit confirmation step whose
// selection defaults to "Cancel".
type Model struct {
	deps Deps

	cursor     int  // index into the action list
	confirming bool // true while showing the delete confirmation
	confirmYes bool // true when the confirmation cursor is on "Yes, delete"

	width, height int
}

// New builds the settings screen.
func New(deps Deps) Model { return Model{deps: deps} }

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
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.confirming {
		return m.handleConfirm(msg)
	}
	return m.handleList(msg)
}

func (m Model) handleList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, nav.Back()
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(actions)-1 {
			m.cursor++
		}
	}
	if ui.IsConfirmKey(msg) {
		// The only action is the destructive one; open its confirmation.
		m.confirming = true
		m.confirmYes = false // default selection is "Cancel"
	}
	return m, nil
}

func (m Model) handleConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.confirming = false
		return m, nil
	case "up", "k", "down", "j", "left", "h", "right", "l":
		m.confirmYes = !m.confirmYes
		return m, nil
	}
	if ui.IsConfirmKey(msg) {
		if m.confirmYes {
			return m, nav.WipeData()
		}
		m.confirming = false
	}
	return m, nil
}

// actions lists the settings entries. Today there is only the destructive one;
// future settings (romaji, theme, …) extend this list.
var actions = []struct{ label func(i18n.Messages) string }{
	{func(m i18n.Messages) string { return m.DeleteAllData }},
}

// View implements tea.Model.
func (m Model) View() tea.View {
	content := m.listView()
	if m.confirming {
		content = m.confirmView()
	}
	view := tea.NewView(ui.Frame(m.deps.Theme, m.width, m.height, content))
	view.AltScreen = true
	return view
}

func (m Model) listView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.SettingsTitle))
	b.WriteString("\n\n")

	for i, a := range actions {
		line := a.label(m.deps.Msgs)
		if i == m.cursor {
			b.WriteString(t.Selected.Render("▸ " + line))
		} else {
			b.WriteString(t.Normal.Render("  " + line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.BackHelp))
	return b.String()
}

func (m Model) confirmView() string {
	t := m.deps.Theme
	var b strings.Builder
	b.WriteString(t.Title.Render(m.deps.Msgs.DeleteAllData))
	b.WriteString("\n\n")
	b.WriteString(t.Error.Render(m.deps.Msgs.DeleteAllWarning))
	b.WriteString("\n\n")

	options := []struct {
		label string
		yes   bool
	}{
		{m.deps.Msgs.CancelLabel, false},
		{m.deps.Msgs.ConfirmDelete, true},
	}
	for _, opt := range options {
		selected := opt.yes == m.confirmYes
		switch {
		case selected && opt.yes:
			b.WriteString(t.Selected.Render("▸ ") + t.Error.Render(opt.label))
		case selected:
			b.WriteString(t.Selected.Render("▸ " + opt.label))
		default:
			b.WriteString(t.Normal.Render("  " + opt.label))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.ConfirmHelp))
	return b.String()
}
