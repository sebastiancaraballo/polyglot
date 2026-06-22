package art

import (
	"strings"
	"testing"
)

// TestGlobeFramesUniform guards the generated frames: every frame must be a
// rectangle of identical, non-empty dimensions so the header layout stays
// aligned, and every cell must be a braille glyph (U+2800–U+28FF).
func TestGlobeFramesUniform(t *testing.T) {
	if len(GlobeFrames) < 2 {
		t.Fatalf("GlobeFrames = %d, want at least 2 for an animation", len(GlobeFrames))
	}

	rows := strings.Count(GlobeFrames[0], "\n") + 1
	cols := len([]rune(strings.SplitN(GlobeFrames[0], "\n", 2)[0]))
	if rows == 0 || cols == 0 {
		t.Fatalf("frame 0 is empty (%d rows, %d cols)", rows, cols)
	}

	for i, frame := range GlobeFrames {
		lines := strings.Split(frame, "\n")
		if len(lines) != rows {
			t.Errorf("frame %d has %d rows, want %d", i, len(lines), rows)
		}
		for j, line := range lines {
			runes := []rune(line)
			if len(runes) != cols {
				t.Errorf("frame %d line %d has %d cols, want %d", i, j, len(runes), cols)
			}
			for _, r := range runes {
				if r < 0x2800 || r > 0x28FF {
					t.Errorf("frame %d line %d has non-braille rune %q", i, j, r)
					break
				}
			}
		}
	}
}
