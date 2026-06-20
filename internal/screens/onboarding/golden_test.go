package onboarding

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func goldenModel() Model {
	return New(Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES})
}

func TestOnboardingWelcomeGolden(t *testing.T) {
	golden.RequireEqual(t, []byte(goldenModel().View().Content))
}

func TestOnboardingExerciseGolden(t *testing.T) {
	next, _ := goldenModel().Update(enter()) // welcome -> exercise
	m := next.(Model)
	golden.RequireEqual(t, []byte(m.View().Content))
}

func TestOnboardingDoneGolden(t *testing.T) {
	next, _ := goldenModel().Update(enter()) // -> exercise
	m := next.(Model)

	correctKey := string(rune('1' + i18n.ES.SampleCorrect))
	next, _ = m.Update(key(correctKey)) // answer correctly
	m = next.(Model)

	next, _ = m.Update(enter()) // -> done
	m = next.(Model)
	golden.RequireEqual(t, []byte(m.View().Content))
}

func key(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
}
