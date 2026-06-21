package avatar

import (
	"strings"
	"testing"

	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func TestOptionsDeterministic(t *testing.T) {
	a := Options("José")
	b := Options("José")
	if len(a) != 1+identiconVariants {
		t.Fatalf("Options returned %d choices, want %d", len(a), 1+identiconVariants)
	}
	wantFirst := "initials:0"
	if a[0].Spec != wantFirst {
		t.Errorf("first spec = %q, want %q", a[0].Spec, wantFirst)
	}
	for i := range a {
		if a[i].Spec != b[i].Spec || a[i].Tile != b[i].Tile {
			t.Errorf("Options not deterministic at %d", i)
		}
	}
}

func TestInline(t *testing.T) {
	cases := map[string]string{
		"Ann Bob":   "AB",
		"李":         "李",
		"josé niño": "JN",
		"":          "?",
	}
	for in, want := range cases {
		if got := Inline(in); got != want {
			t.Errorf("Inline(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIdenticonShapeAndSymmetry(t *testing.T) {
	tile := tile("identicon:1", "Mei")
	lines := strings.Split(tile, "\n")
	if len(lines) != gridSize {
		t.Fatalf("identicon has %d lines, want %d", len(lines), gridSize)
	}
	for i, line := range lines {
		cols := []rune(line)
		if len(cols) != tileWidth {
			t.Errorf("line %d width = %d, want %d", i, len(cols), tileWidth)
		}
		// Cells are 2 runes wide; the grid is horizontally symmetric.
		for c := 0; c < gridSize/2; c++ {
			l := string(cols[c*2 : c*2+2])
			r := string(cols[(gridSize-1-c)*2 : (gridSize-1-c)*2+2])
			if l != r {
				t.Errorf("line %d not symmetric: col %d %q vs %q", i, c, l, r)
			}
		}
	}
}

func TestRenderInitialsTileContainsInitials(t *testing.T) {
	out := Render(ui.PlainTheme(), "initials:0", "Ann Bob")
	if !strings.Contains(out, "AB") {
		t.Errorf("initials tile should contain %q:\n%s", "AB", out)
	}
}
