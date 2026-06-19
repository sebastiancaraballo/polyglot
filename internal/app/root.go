package app

import tea "charm.land/bubbletea/v2"

// rootModel is the top-level Bubble Tea model. It tracks the terminal size and
// delegates to the active screen, providing the seam where additional screens
// will be routed in later steps.
type rootModel struct {
	screen tea.Model
	width  int
	height int
}

func newRoot(screen tea.Model) rootModel {
	return rootModel{screen: screen}
}

func (m rootModel) Init() tea.Cmd {
	return m.screen.Init()
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = ws.Width, ws.Height
	}
	var cmd tea.Cmd
	m.screen, cmd = m.screen.Update(msg)
	return m, cmd
}

func (m rootModel) View() tea.View {
	return m.screen.View()
}
