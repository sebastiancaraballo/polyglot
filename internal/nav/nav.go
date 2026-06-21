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
	Onboarding
	Settings
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

// WipeDataMsg requests deletion of all local app data. It is handled by the
// router, which owns the storage connection and application context.
type WipeDataMsg struct{}

// WipeData returns a command that requests deletion of all local app data.
func WipeData() tea.Cmd {
	return func() tea.Msg { return WipeDataMsg{} }
}
