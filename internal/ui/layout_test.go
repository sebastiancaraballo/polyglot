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
