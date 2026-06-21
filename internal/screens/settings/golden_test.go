package settings

import (
	"testing"

	"github.com/charmbracelet/x/exp/golden"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func TestSettingsListGolden(t *testing.T) {
	m := New(Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES})
	golden.RequireEqual(t, []byte(m.View().Content))
}

func TestSettingsConfirmGolden(t *testing.T) {
	m := New(Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES})
	m.confirming = true
	golden.RequireEqual(t, []byte(m.View().Content))
}
