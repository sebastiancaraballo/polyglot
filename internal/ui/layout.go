package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Frame outer dimensions. They are sized to fit the tallest screen so the frame
// is identical in every section and in every state within a section, instead of
// growing and shrinking with its content. They are upper bounds: Frame shrinks
// them to fit a smaller terminal.
const (
	frameWidth  = 64
	frameHeight = 22
)

// Frame renders content inside the app's border at a fixed size and centers it
// in the terminal. The frame's outer size depends only on the terminal size —
// never on the content — so it stays put as the user moves between screens or a
// screen swaps its content (e.g. revealing a quiz answer). Content is anchored
// top-left and clamped to the frame.
func Frame(theme Theme, width, height int, content string) string {
	w, h := frameWidth, frameHeight
	if width > 0 && width-2 < w {
		w = width - 2
	}
	if height > 0 && height-2 < h {
		h = height - 2
	}
	box := theme.Box.
		Width(w).
		Height(h).
		MaxWidth(w).
		MaxHeight(h)
	return Center(width, height, box.Render(content))
}

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
