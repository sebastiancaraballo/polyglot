// Package nav defines the messages used to navigate between screens. It lives in
// its own package so both the router and the individual screens can depend on it
// without creating an import cycle.
package nav

import tea "charm.land/bubbletea/v2"

// Screen identifies a top-level screen.
type Screen int

const (
	Menu Screen = iota
	Kana
	Flashcards
	Quiz
	Stats
)

// GoToMsg requests navigation to a screen.
type GoToMsg struct{ Screen Screen }

// GoTo returns a command that requests navigation to s.
func GoTo(s Screen) tea.Cmd {
	return func() tea.Msg { return GoToMsg{Screen: s} }
}

// BackMsg requests navigation back to the main menu.
type BackMsg struct{}

// Back returns a command that requests navigation back to the main menu.
func Back() tea.Cmd {
	return func() tea.Msg { return BackMsg{} }
}
