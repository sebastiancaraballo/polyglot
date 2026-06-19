package ui

import "charm.land/lipgloss/v2"

// Center places content in the middle of a width×height area. When the size is
// unknown (zero), it returns the content unchanged so screens still render.
func Center(width, height int, content string) string {
	if width <= 0 || height <= 0 {
		return content
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

// ProgressBar renders a fixed-width bar for a percentage in [0, 100] using block
// characters. The caller is responsible for styling/coloring the result.
func ProgressBar(percent, width int) string {
	if width <= 0 {
		return ""
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := percent * width / 100
	bar := make([]rune, 0, width)
	for i := 0; i < width; i++ {
		if i < filled {
			bar = append(bar, '█')
		} else {
			bar = append(bar, '░')
		}
	}
	return string(bar)
}
