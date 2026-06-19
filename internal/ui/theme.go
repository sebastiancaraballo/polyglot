package ui

import (
	"os"

	"charm.land/lipgloss/v2"
)

// NoColor reports whether color output should be disabled, following the
// NO_COLOR convention (https://no-color.org): the variable is set and non-empty.
func NoColor() bool {
	v, ok := os.LookupEnv("NO_COLOR")
	return ok && v != ""
}

// Theme is the set of styles used across screens. A high-contrast variant avoids
// color and relies on bold/reverse so the UI stays legible without color.
type Theme struct {
	Title    lipgloss.Style
	Subtle   lipgloss.Style
	Normal   lipgloss.Style
	Selected lipgloss.Style
	Accent   lipgloss.Style
	Success  lipgloss.Style
	Error    lipgloss.Style
	Help     lipgloss.Style
	Box      lipgloss.Style
}

// DefaultTheme returns the theme appropriate for the current environment,
// switching to high-contrast when NO_COLOR is set.
func DefaultTheme() Theme {
	return NewTheme(NoColor())
}

// NewTheme builds a theme. When highContrast is true, colors are dropped in
// favor of bold and reverse styling.
func NewTheme(highContrast bool) Theme {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 3)

	if highContrast {
		return Theme{
			Title:    lipgloss.NewStyle().Bold(true),
			Subtle:   lipgloss.NewStyle().Faint(true),
			Normal:   lipgloss.NewStyle(),
			Selected: lipgloss.NewStyle().Bold(true).Reverse(true),
			Accent:   lipgloss.NewStyle().Bold(true),
			Success:  lipgloss.NewStyle().Bold(true),
			Error:    lipgloss.NewStyle().Bold(true),
			Help:     lipgloss.NewStyle().Faint(true),
			Box:      box,
		}
	}

	var (
		accent  = lipgloss.Color("63")  // indigo
		subtle  = lipgloss.Color("245") // grey
		success = lipgloss.Color("42")  // green
		danger  = lipgloss.Color("203") // red
	)
	return Theme{
		Title:    lipgloss.NewStyle().Foreground(accent).Bold(true),
		Subtle:   lipgloss.NewStyle().Foreground(subtle),
		Normal:   lipgloss.NewStyle(),
		Selected: lipgloss.NewStyle().Foreground(accent).Bold(true),
		Accent:   lipgloss.NewStyle().Foreground(accent),
		Success:  lipgloss.NewStyle().Foreground(success),
		Error:    lipgloss.NewStyle().Foreground(danger),
		Help:     lipgloss.NewStyle().Foreground(subtle),
		Box:      box.BorderForeground(accent),
	}
}
