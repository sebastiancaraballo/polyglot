package avatar

import (
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

const (
	gridSize  = 5 // identicon is gridSize×gridSize, horizontally symmetric
	tileWidth = gridSize * 2
	filled    = "██"
	empty     = "  "
)

// identiconVariants is how many distinct block identicons Options offers.
const identiconVariants = 4

// Choice is a generated avatar offered to the user at profile creation.
type Choice struct {
	Spec string // stored on the profile, e.g. "initials:0" or "identicon:2"
	Tile string // monochrome multi-line preview; callers apply color
}

// Options returns a deterministic set of avatars for name: one initials tile plus a
// few block identicons. The result depends only on name, so it is stable.
func Options(name string) []Choice {
	choices := []Choice{{Spec: "initials:0", Tile: tile("initials:0", name)}}
	for i := 0; i < identiconVariants; i++ {
		spec := "identicon:" + strconv.Itoa(i)
		choices = append(choices, Choice{Spec: spec, Tile: tile(spec, name)})
	}
	return choices
}

// Render returns the avatar tile for spec/name, tinted with the theme accent. With
// NO_COLOR the accent carries no color but the shape stays visible.
func Render(t ui.Theme, spec, name string) string {
	return t.Accent.Render(tile(spec, name))
}

// Inline returns a compact one-line token (the initials) for list rows.
func Inline(name string) string { return initials(name) }

func tile(spec, name string) string {
	kind, variant := parseSpec(spec)
	if kind == "identicon" {
		return identicon(name, variant)
	}
	return initialsTile(name)
}

func parseSpec(spec string) (kind string, variant int) {
	kind, rest, found := strings.Cut(spec, ":")
	if !found {
		return spec, 0
	}
	variant, _ = strconv.Atoi(rest)
	return kind, variant
}

// identicon renders a gridSize×gridSize horizontally-symmetric block grid seeded by
// name and variant.
func identicon(name string, variant int) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name + "#" + strconv.Itoa(variant)))
	bits := h.Sum64()

	var b strings.Builder
	half := gridSize/2 + 1 // columns we decide; the rest mirror
	bit := 0
	for row := 0; row < gridSize; row++ {
		cells := make([]string, gridSize)
		for col := 0; col < half; col++ {
			on := bits&(1<<uint(bit%64)) != 0
			bit++
			glyph := empty
			if on {
				glyph = filled
			}
			cells[col] = glyph
			cells[gridSize-1-col] = glyph // mirror
		}
		if row > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(strings.Join(cells, ""))
	}
	return b.String()
}

// initialsTile renders the name's initials centered in a gridSize-tall box matching
// the identicon width.
func initialsTile(name string) string {
	inner := tileWidth - 2 // space between the side borders
	top := "┌" + strings.Repeat("─", inner) + "┐"
	bottom := "└" + strings.Repeat("─", inner) + "┘"
	blankRow := "│" + strings.Repeat(" ", inner) + "│"
	mid := "│" + center(initials(name), inner) + "│"

	rows := []string{top}
	for len(rows) < gridSize-1 {
		rows = append(rows, blankRow)
	}
	rows[gridSize/2] = mid
	rows = append(rows, bottom)
	return strings.Join(rows, "\n")
}

// initials returns 1–2 uppercase initials from name (first and last word).
func initials(name string) string {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return "?"
	}
	out := firstRune(fields[0])
	if len(fields) > 1 {
		out += firstRune(fields[len(fields)-1])
	}
	return strings.ToUpper(out)
}

func firstRune(s string) string {
	for _, r := range s {
		return string(r)
	}
	return ""
}

func center(s string, width int) string {
	w := 0
	for range s {
		w++
	}
	if w >= width {
		return s
	}
	left := (width - w) / 2
	right := width - w - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}
