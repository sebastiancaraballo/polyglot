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
