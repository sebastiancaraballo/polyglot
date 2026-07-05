package menu

import (
	"testing"

	"github.com/charmbracelet/x/exp/golden"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func TestMenuGolden(t *testing.T) {
	summary := Summary{Name: "Sebastián", XP: 1240, Streak: 5, Learned: 8, Total: 20}
	m := New(ui.PlainTheme(), i18n.ES, summary, "test")
	golden.RequireEqual(t, []byte(m.View().Content))
}

func TestMenuSubmenuGolden(t *testing.T) {
	summary := Summary{Name: "Sebastián", XP: 1240, Streak: 5, Learned: 8, Total: 20}
	m := New(ui.PlainTheme(), i18n.ES, summary, "test")
	m.section = 0 // the first category (Aprender) open
	m.cursor = 0
	golden.RequireEqual(t, []byte(m.View().Content))
}
