package settings

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

// Deps are the dependencies the settings screen needs. The screen performs no
// storage work itself: each entry emits a nav message (nav.SetShowRomaji,
// nav.WipeData, …) and the router (which owns the storage connection) carries it
// out.
type Deps struct {
	Theme      ui.Theme
	Msgs       i18n.Messages
	ShowRomaji bool
}

// Model is the settings screen. The first row is the "show romaji" toggle; the
// remaining rows are destructive actions, each with an explicit confirmation step
// whose selection defaults to "Cancel".
type Model struct {
	deps Deps

	showRomaji bool // current value of the romaji toggle

	cursor        int  // index into the row list (0 = toggle, 1.. = actions)
	confirming    bool // true while showing the delete confirmation
	confirmAction int  // index into actions of the action being confirmed
	confirmYes    bool // true when the confirmation cursor is on "Yes, delete"

	width, height int
}

// New builds the settings screen.
func New(deps Deps) Model { return Model{deps: deps, showRomaji: deps.ShowRomaji} }

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
		if m.cursor < len(actions) { // rows: 0 = toggle, 1..len(actions) = actions
			m.cursor++
		}
	}
	if ui.IsConfirmKey(msg) {
		if m.cursor == 0 {
			m.showRomaji = !m.showRomaji
			return m, nav.SetShowRomaji(m.showRomaji)
		}
		m.confirming = true
		m.confirmAction = m.cursor - 1
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
			return m, actions[m.confirmAction].cmd()
		}
		m.confirming = false
	}
	return m, nil
}

type action struct {
	label   func(i18n.Messages) string
	title   func(i18n.Messages) string
	warning func(i18n.Messages) string
	confirm func(i18n.Messages) string
	cmd     func() tea.Cmd
}

// actions lists the settings entries. All destructive actions use the same
// confirmation UI and differ only in copy and emitted navigation message.
var actions = []action{
	{
		label:   func(m i18n.Messages) string { return m.DeleteProfile },
		title:   func(m i18n.Messages) string { return m.DeleteProfile },
		warning: func(m i18n.Messages) string { return m.DeleteProfileWarning },
		confirm: func(m i18n.Messages) string { return m.ConfirmDeleteProfile },
		cmd:     nav.DeleteProfile,
	},
	{
		label:   func(m i18n.Messages) string { return m.DeleteAllData },
		title:   func(m i18n.Messages) string { return m.DeleteAllData },
		warning: func(m i18n.Messages) string { return m.DeleteAllWarning },
		confirm: func(m i18n.Messages) string { return m.ConfirmDelete },
		cmd:     nav.WipeData,
	},
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

	b.WriteString(m.renderRow(t, 0, m.romajiLabel()))
	for i, a := range actions {
		b.WriteString(m.renderRow(t, i+1, a.label(m.deps.Msgs)))
	}

	b.WriteString("\n")
	b.WriteString(t.Help.Render(m.deps.Msgs.SettingsHelp))
	return b.String()
}

// renderRow renders a single settings row, highlighting it when the cursor is on it.
func (m Model) renderRow(t ui.Theme, index int, line string) string {
	if index == m.cursor {
		return t.Selected.Render("▸ "+line) + "\n"
	}
	return t.Normal.Render("  "+line) + "\n"
}

// romajiLabel renders the toggle row, e.g. "Mostrar romaji: Sí".
func (m Model) romajiLabel() string {
	value := m.deps.Msgs.OptionOff
	if m.showRomaji {
		value = m.deps.Msgs.OptionOn
	}
	return m.deps.Msgs.ShowRomajiLabel + ": " + value
}

func (m Model) confirmView() string {
	t := m.deps.Theme
	action := actions[m.confirmAction]
	var b strings.Builder
	b.WriteString(t.Title.Render(action.title(m.deps.Msgs)))
	b.WriteString("\n\n")
	b.WriteString(t.Error.Render(action.warning(m.deps.Msgs)))
	b.WriteString("\n\n")

	options := []struct {
		label string
		yes   bool
	}{
		{m.deps.Msgs.CancelLabel, false},
		{action.confirm(m.deps.Msgs), true},
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
