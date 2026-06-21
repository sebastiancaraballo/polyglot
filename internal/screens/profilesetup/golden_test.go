package profilesetup

import (
	"testing"

	"github.com/charmbracelet/x/exp/golden"

	"github.com/sebastiancaraballo/polyglot/internal/avatar"
	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func TestNameGolden(t *testing.T) {
	m := New(Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES, Tutorial: true})
	golden.RequireEqual(t, []byte(m.View().Content))
}

func TestAvatarGolden(t *testing.T) {
	m := New(Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES, Tutorial: true})
	m.name = "Sebastián"
	m.choices = avatar.Options(m.name)
	m.step = stepAvatar
	golden.RequireEqual(t, []byte(m.View().Content))
}
