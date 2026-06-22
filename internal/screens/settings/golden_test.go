package settings

import (
	"testing"

	"github.com/charmbracelet/x/exp/golden"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func TestSettingsListGolden(t *testing.T) {
	m := New(Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES, ShowRomaji: true})
	golden.RequireEqual(t, []byte(m.View().Content))
}

func TestSettingsListRomajiOffGolden(t *testing.T) {
	m := New(Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES, ShowRomaji: false})
	golden.RequireEqual(t, []byte(m.View().Content))
}

func TestSettingsConfirmGolden(t *testing.T) {
	m := New(Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES, ShowRomaji: true})
	m.cursor = 1 // delete profile row
	m.confirming = true
	golden.RequireEqual(t, []byte(m.View().Content))
}
