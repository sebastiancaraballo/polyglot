package profilesetup

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sebastiancaraballo/polyglot/internal/avatar"
	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

type step int

const (
	stepName step = iota
	stepAvatar
)

// Deps are the dependencies the profile setup flow needs.
type Deps struct {
	Theme    ui.Theme
	Msgs     i18n.Messages
	Store    storage.Storage
	Tutorial bool
}

// Model is the profile creation flow: it asks for a name, lets the learner choose
// one generated text avatar, creates the profile, and reports it to the router.
type Model struct {
	deps Deps

	step      step
	name      string
	submitted bool
	choices   []avatar.Choice
	selected  int
	err       error

	width, height int
}

// New builds the profile setup flow.
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
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	}

	switch m.step {
	case stepAvatar:
		return m.handleAvatar(msg)
	default:
		return m.handleName(msg)
	}
}

func (m Model) handleName(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if !m.deps.Tutorial {
			return m, nav.Back()
		}
	case "enter":
		return m.submitName()
	case "space":
		m.name += " "
	case "backspace", "ctrl+h":
		m.name = dropLastRune(m.name)
	default:
		if text := keyText(msg); text != "" {
			m.name += text
		}
	}
	m.err = nil
	return m, nil
}

func (m Model) submitName() (tea.Model, tea.Cmd) {
	m.submitted = true
	name, err := model.NormalizeName(m.name)
	if err != nil {
		return m, nil
	}
	m.name = name
	m.choices = avatar.Options(name)
	m.selected = 0
	m.step = stepAvatar
	m.err = nil
	return m, nil
}

func (m Model) handleAvatar(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.step = stepName
		m.err = nil
	case "left", "h", "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "right", "l", "down", "j":
		if m.selected < len(m.choices)-1 {
			m.selected++
		}
	case "1", "2", "3", "4", "5":
		i := int(msg.String()[0] - '1')
		if i < len(m.choices) {
			m.selected = i
		}
	}
	if ui.IsConfirmKey(msg) {
		return m.createProfile()
	}
	return m, nil
}

func (m Model) createProfile() (tea.Model, tea.Cmd) {
	name, err := model.NormalizeName(m.name)
	if err != nil {
		m.step = stepName
		m.submitted = true
		return m, nil
	}
	m.name = name
	if len(m.choices) == 0 {
		m.choices = avatar.Options(m.name)
	}
	p, err := m.deps.Store.CreateProfile(context.Background(), m.name, m.choices[m.selected].Spec)
	if err != nil {
		m.err = err
		return m, nil
	}
	return m, nav.ProfileCreated(p.ID, m.deps.Tutorial)
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var content string
	switch m.step {
	case stepAvatar:
		content = m.avatarView()
	default:
		content = m.nameView()
	}
	view := tea.NewView(ui.Frame(m.deps.Theme, m.width, m.height, content))
	view.AltScreen = true
	return view
}

func (m Model) nameView() string {
	t := m.deps.Theme
	msgs := m.deps.Msgs

	var b strings.Builder
	b.WriteString(t.Title.Render(msgs.ProfileNameTitle))
	b.WriteString("\n\n")
	b.WriteString(msgs.ProfileNamePrompt)
	b.WriteString("\n\n")

	input := m.name
	if input == "" {
		input = t.Subtle.Render(msgs.ProfileNamePlaceholder)
	}
	fmt.Fprintf(&b, "> %s\n", input)

	if text := m.validationText(); text != "" {
		b.WriteString("\n")
		b.WriteString(t.Error.Render(text))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	help := msgs.ProfileNameHelpFirst
	if !m.deps.Tutorial {
		help = msgs.ProfileNameHelpCancel
	}
	b.WriteString(t.Help.Render(help))
	return b.String()
}

func (m Model) avatarView() string {
	t := m.deps.Theme
	msgs := m.deps.Msgs

	var b strings.Builder
	b.WriteString(t.Title.Render(msgs.ProfileAvatarTitle))
	b.WriteString("\n\n")
	b.WriteString(msgs.ProfileAvatarPrompt)
	b.WriteString("\n\n")
	b.WriteString(m.avatarOptions())

	if m.err != nil {
		b.WriteString("\n\n")
		b.WriteString(t.Error.Render(msgs.ProfileCreateError))
	}

	b.WriteString("\n\n")
	b.WriteString(t.Help.Render(msgs.ProfileAvatarHelp))
	return b.String()
}

func (m Model) avatarOptions() string {
	blocks := make([]string, 0, len(m.choices))
	for i, choice := range m.choices {
		marker := "  "
		if i == m.selected {
			marker = "▸ "
		}
		block := fmt.Sprintf("%s%d\n%s", marker, i+1, choice.Tile)
		if i == m.selected {
			block = m.deps.Theme.Selected.Render(block)
		} else {
			block = m.deps.Theme.Normal.Render(block)
		}
		block = lipgloss.NewStyle().MarginRight(1).Render(block)
		blocks = append(blocks, block)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, blocks...)
}

func (m Model) validationText() string {
	if !m.submitted && strings.TrimSpace(m.name) == "" {
		return ""
	}
	_, err := model.NormalizeName(m.name)
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, model.ErrEmptyName):
		return m.deps.Msgs.ProfileNameEmpty
	case errors.Is(err, model.ErrNameTooLong):
		return fmt.Sprintf(m.deps.Msgs.ProfileNameTooLongFmt, model.MaxNameLen)
	default:
		return m.deps.Msgs.ProfileNameInvalid
	}
}

func dropLastRune(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
}

func keyText(msg tea.KeyPressMsg) string {
	if msg.Text != "" {
		return msg.Text
	}
	s := msg.String()
	if len([]rune(s)) == 1 {
		return s
	}
	return ""
}
