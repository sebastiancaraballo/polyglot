package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestWrapTextRespectsDisplayWidth(t *testing.T) {
	got := WrapText("Pronunciación: ohayō. Entrada kana: ohayou. Cortés: おはようございます.", 28)
	for _, line := range strings.Split(got, "\n") {
		if width := lipgloss.Width(line); width > 28 {
			t.Fatalf("line %q width = %d, want <= 28", line, width)
		}
	}
}

func TestWrapTextPreservesExistingLineBreaks(t *testing.T) {
	got := WrapText("Pronunciación: ohayō.\nEntrada kana: ohayou.", 80)
	if got != "Pronunciación: ohayō.\nEntrada kana: ohayou." {
		t.Fatalf("WrapText() = %q", got)
	}
}

func TestWrapTextBreaksSpacelessJapanese(t *testing.T) {
	// Japanese prose has no spaces, so strings.Fields keeps a whole sentence as
	// one token; it must still be hard-broken by display width (wide runes count
	// as two cells) so it wraps inside the frame instead of overflowing.
	jp := "これが基本の疑問詞です。どこ、どう、どちら。よく聞いてくださいね。"
	got := WrapText(jp, 28)
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected the sentence to wrap onto multiple lines, got %d line(s)", len(lines))
	}
	for _, line := range lines {
		if width := lipgloss.Width(line); width > 28 {
			t.Fatalf("line %q width = %d, want <= 28", line, width)
		}
	}
}

func TestFrameContentWidthMatchesFrameInterior(t *testing.T) {
	if got := FrameContentWidth(PlainTheme(), 80); got != 56 {
		t.Fatalf("FrameContentWidth() = %d, want 56", got)
	}
}

func TestFrameContentHeightMatchesFrameInterior(t *testing.T) {
	if got := FrameContentHeight(PlainTheme(), 40); got != 19 {
		t.Fatalf("FrameContentHeight() = %d, want 19", got)
	}
}
