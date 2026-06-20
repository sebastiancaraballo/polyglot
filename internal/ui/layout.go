package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Center places content in the middle of a width×height area. When the size is
// unknown (zero), it returns the content unchanged so screens still render.
func Center(width, height int, content string) string {
	if width <= 0 || height <= 0 {
		return content
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

// WrapText wraps text to the requested display width, preserving existing line
// breaks. It is intended for short UI copy, not full prose layout.
func WrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, wrapLine(line, width)...)
	}
	return strings.Join(wrapped, "\n")
}

func wrapLine(line string, width int) []string {
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		next := current + " " + word
		if lipgloss.Width(next) <= width {
			current = next
			continue
		}
		lines = append(lines, current)
		current = word
	}
	return append(lines, current)
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
