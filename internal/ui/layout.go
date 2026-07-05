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
	frameHeight = 23
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

// FitFrame renders content like Frame but lets the frame grow taller than the
// shared fixed height to fit its content, for a screen that needs more vertical
// room than the others — currently the kana chart. The frame hugs the content
// (capped to the terminal) instead of filling the whole screen, and its width
// still matches Frame so the border lines up horizontally with every other screen.
func FitFrame(theme Theme, width, height int, content string) string {
	w := frameWidth
	if width > 0 && width-2 < w {
		w = width - 2
	}
	h := lipgloss.Height(content) + theme.Box.GetVerticalFrameSize()
	if height > 0 && height-2 < h {
		h = height - 2
	}
	if h < 1 {
		h = 1
	}
	box := theme.Box.
		Width(w).
		Height(h).
		MaxWidth(w).
		MaxHeight(h)
	return Center(width, height, box.Render(content))
}

// MaxFrameContentHeight returns the content height available inside a frame that
// fills the terminal. The kana chart uses it to budget its row spacing against the
// tallest the frame could ever be.
func MaxFrameContentHeight(theme Theme, height int) int {
	h := frameHeight
	if height > 0 {
		h = height - 2
	}
	contentHeight := h - theme.Box.GetVerticalFrameSize()
	if contentHeight < 0 {
		return 0
	}
	return contentHeight
}

// FrameContentWidth returns the display width available for content inside
// Frame's border and padding.
func FrameContentWidth(theme Theme, width int) int {
	w := frameWidth
	if width > 0 && width-2 < w {
		w = width - 2
	}
	contentWidth := w - theme.Box.GetHorizontalFrameSize()
	if contentWidth < 0 {
		return 0
	}
	return contentWidth
}

// FrameContentHeight returns the display height available for content inside
// Frame's border and padding.
func FrameContentHeight(theme Theme, height int) int {
	h := frameHeight
	if height > 0 && height-2 < h {
		h = height - 2
	}
	contentHeight := h - theme.Box.GetVerticalFrameSize()
	if contentHeight < 0 {
		return 0
	}
	return contentHeight
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
	current := ""
	for _, word := range words {
		// A single token wider than the frame (e.g. space-less Japanese text,
		// which strings.Fields keeps whole) is hard-broken by display width so
		// it wraps instead of overflowing the frame.
		for _, piece := range breakWord(word, width) {
			switch {
			case current == "":
				current = piece
			case lipgloss.Width(current+" "+piece) <= width:
				current += " " + piece
			default:
				lines = append(lines, current)
				current = piece
			}
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

// breakWord splits a single whitespace-free token into chunks that each fit
// width display cells, breaking between runes and honoring wide (CJK) runes.
// A token that already fits is returned unchanged. This is what lets scripts
// without spaces (Japanese) wrap at all.
func breakWord(word string, width int) []string {
	if width < 1 {
		width = 1
	}
	if lipgloss.Width(word) <= width {
		return []string{word}
	}
	var chunks []string
	var cur []rune
	curW := 0
	for _, r := range word {
		rw := lipgloss.Width(string(r))
		if curW+rw > width && len(cur) > 0 {
			chunks = append(chunks, string(cur))
			cur = cur[:0]
			curW = 0
		}
		cur = append(cur, r)
		curW += rw
	}
	if len(cur) > 0 {
		chunks = append(chunks, string(cur))
	}
	return chunks
}
