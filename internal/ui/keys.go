package ui

import tea "charm.land/bubbletea/v2"

// IsConfirmKey reports whether a key press should activate the current item.
func IsConfirmKey(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "enter", "space":
		return true
	default:
		return false
	}
}
