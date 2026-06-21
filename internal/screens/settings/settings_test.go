package settings

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func newTestModel() Model {
	return New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES})
}

func enter() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyEnter} }
func esc() tea.KeyPressMsg   { return tea.KeyPressMsg{Code: tea.KeyEscape} }

func TestSelectingDeleteOpensConfirmation(t *testing.T) {
	m := newTestModel()
	next, _ := m.Update(enter())
	got := next.(Model)
	if !got.confirming {
		t.Fatal("confirming the delete item should open the confirmation")
	}
	if got.confirmYes {
		t.Error("confirmation should default to Cancel, not Yes")
	}
}

func TestConfirmYesEmitsWipeData(t *testing.T) {
	m := newTestModel()
	next, _ := m.Update(enter())                                      // open confirmation
	next, _ = next.(Model).Update(tea.KeyPressMsg{Code: tea.KeyDown}) // move to "Yes"
	if !next.(Model).confirmYes {
		t.Fatal("down should move the cursor to Yes")
	}
	_, cmd := next.(Model).Update(enter())
	if cmd == nil {
		t.Fatal("confirming Yes should return a command")
	}
	if _, ok := cmd().(nav.WipeDataMsg); !ok {
		t.Fatalf("expected nav.WipeDataMsg, got %T", cmd())
	}
}

func TestConfirmDefaultCancelDoesNotWipe(t *testing.T) {
	m := newTestModel()
	next, _ := m.Update(enter()) // open confirmation (default Cancel)
	res, cmd := next.(Model).Update(enter())
	if cmd != nil {
		t.Fatalf("selecting Cancel should not emit a command, got %T", cmd())
	}
	if res.(Model).confirming {
		t.Error("selecting Cancel should return to the settings list")
	}
}

func TestEscFromListGoesBack(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(esc())
	if cmd == nil {
		t.Fatal("esc should return a command")
	}
	if _, ok := cmd().(nav.BackMsg); !ok {
		t.Fatalf("expected nav.BackMsg, got %T", cmd())
	}
}

func TestEscFromConfirmCancels(t *testing.T) {
	m := newTestModel()
	next, _ := m.Update(enter()) // open confirmation
	res, cmd := next.(Model).Update(esc())
	if cmd != nil {
		t.Fatalf("esc in confirmation should not emit a command, got %T", cmd())
	}
	if res.(Model).confirming {
		t.Error("esc should cancel the confirmation and return to the list")
	}
}
