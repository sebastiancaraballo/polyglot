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
	Review
	Quiz
	Stats
	Onboarding
	Settings
	ProfileSetup
	Profiles
	KanaChart
	Rikai
	Story
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

// ProfileCreatedMsg reports that a profile setup flow created a new profile.
// Tutorial indicates whether first-run onboarding should run next.
type ProfileCreatedMsg struct {
	ID       int64
	Tutorial bool
}

// ProfileCreated returns a command that reports a newly-created profile.
func ProfileCreated(id int64, tutorial bool) tea.Cmd {
	return func() tea.Msg { return ProfileCreatedMsg{ID: id, Tutorial: tutorial} }
}

// SwitchProfileMsg requests changing the active local profile.
type SwitchProfileMsg struct{ ID int64 }

// SwitchProfile returns a command that requests switching to profile id.
func SwitchProfile(id int64) tea.Cmd {
	return func() tea.Msg { return SwitchProfileMsg{ID: id} }
}

// CreateProfileMsg requests navigation to the add-profile setup flow.
type CreateProfileMsg struct{}

// CreateProfile returns a command that requests profile creation.
func CreateProfile() tea.Cmd {
	return func() tea.Msg { return CreateProfileMsg{} }
}

// DeleteProfileMsg requests deletion of the active profile. It is handled by the
// router, which owns the active profile and storage context.
type DeleteProfileMsg struct{}

// DeleteProfile returns a command that requests deletion of the active profile.
func DeleteProfile() tea.Cmd {
	return func() tea.Msg { return DeleteProfileMsg{} }
}

// WipeDataMsg requests deletion of all local app data. It is handled by the
// router, which owns the storage connection and application context.
type WipeDataMsg struct{}

// WipeData returns a command that requests deletion of all local app data.
func WipeData() tea.Cmd {
	return func() tea.Msg { return WipeDataMsg{} }
}

// SetShowRomajiMsg requests persisting the active profile's romaji preference. It
// is handled by the router, which owns the active profile and storage context.
type SetShowRomajiMsg struct{ Enabled bool }

// SetShowRomaji returns a command that requests persisting the romaji preference.
func SetShowRomaji(enabled bool) tea.Cmd {
	return func() tea.Msg { return SetShowRomajiMsg{Enabled: enabled} }
}
