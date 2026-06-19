package onboarding

import (
	"context"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func newTestModel(t *testing.T) (Model, *storage.SQLiteStore, int64) {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	profile, err := store.CreateProfile(context.Background(), "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	m := New(Deps{Theme: ui.NewTheme(true), Msgs: i18n.ES, Store: store, ProfileID: profile.ID})
	return m, store, profile.ID
}

func enter() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyEnter} }

func space() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeySpace} }

func TestOnboardingFlowCompletes(t *testing.T) {
	m, store, profileID := newTestModel(t)

	// Welcome -> exercise.
	next, _ := m.Update(enter())
	m = next.(Model)
	if m.step != stepExercise {
		t.Fatalf("step = %d, want stepExercise", m.step)
	}

	// Choose the correct option (index SampleCorrect, 1-based key).
	correctKey := string(rune('1' + i18n.ES.SampleCorrect))
	next, _ = m.Update(tea.KeyPressMsg{Code: rune(correctKey[0]), Text: correctKey})
	m = next.(Model)
	if !m.answered || !m.correct {
		t.Fatalf("expected a correct answer, got answered=%v correct=%v", m.answered, m.correct)
	}

	// Confirm -> done.
	next, _ = m.Update(enter())
	m = next.(Model)
	if m.step != stepDone {
		t.Fatalf("step = %d, want stepDone", m.step)
	}

	// Confirm -> finish: persists onboarded and navigates back.
	_, cmd := m.Update(enter())
	if cmd == nil {
		t.Fatal("expected a command on finish")
	}
	if _, ok := cmd().(nav.BackMsg); !ok {
		t.Fatal("finish should navigate back to the menu")
	}

	profile, err := store.GetProfile(context.Background(), profileID)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if !profile.Onboarded {
		t.Error("profile should be marked onboarded after finishing")
	}
}

func TestOnboardingWrongAnswerStays(t *testing.T) {
	m, _, _ := newTestModel(t)
	next, _ := m.Update(enter()) // to exercise
	m = next.(Model)

	wrong := 0
	if i18n.ES.SampleCorrect == 0 {
		wrong = 1
	}
	key := string(rune('1' + wrong))
	next, _ = m.Update(tea.KeyPressMsg{Code: rune(key[0]), Text: key})
	m = next.(Model)
	if m.correct {
		t.Fatal("selecting a wrong option should not be marked correct")
	}
	if m.step != stepExercise {
		t.Fatal("a wrong answer should keep the learner on the exercise step")
	}
}

func TestOnboardingSpaceAdvancesAndAnswers(t *testing.T) {
	m, _, _ := newTestModel(t)

	next, _ := m.Update(space())
	m = next.(Model)
	if m.step != stepExercise {
		t.Fatalf("step = %d, want stepExercise", m.step)
	}

	next, _ = m.Update(space())
	m = next.(Model)
	if !m.answered {
		t.Fatal("space should answer the sample exercise")
	}
}
