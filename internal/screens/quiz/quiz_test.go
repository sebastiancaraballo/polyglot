package quiz

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func romajiTestModel(showRomaji bool) Model {
	card := model.Card{ID: "test:1", Source: "Gracias", JP: "ありがとう", Romaji: "arigatou"}
	return Model{
		deps:    Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES, ShowRomaji: showRomaji},
		deck:    []model.Card{card},
		options: []string{card.JP},
		romaji:  map[string]string{card.JP: card.Romaji},
		correct: 0,
	}
}

func TestQuestionViewShowsRomajiWhenEnabled(t *testing.T) {
	got := romajiTestModel(true).questionView()
	if !strings.Contains(got, "arigatou") {
		t.Fatalf("expected romaji alongside options, got %q", got)
	}
}

func TestQuestionViewHidesRomajiWhenDisabled(t *testing.T) {
	got := romajiTestModel(false).questionView()
	if strings.Contains(got, "arigatou") {
		t.Fatalf("romaji should be hidden when disabled, got %q", got)
	}
	if !strings.Contains(got, "ありがとう") {
		t.Fatalf("japanese option should still show, got %q", got)
	}
}

func TestSpaceAnswersQuizQuestion(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	profile, err := store.CreateProfile(context.Background(), "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	card := model.Card{ID: "test:1", Source: "Gracias", JP: "ありがとう", Romaji: "arigatou"}
	m := Model{
		deps: Deps{
			Theme:     ui.NewTheme(true),
			Msgs:      i18n.ES,
			Store:     store,
			ProfileID: profile.ID,
			Cards:     []model.Card{card},
		},
		deck:    []model.Card{card},
		options: []string{card.JP},
		correct: 0,
	}

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	got := next.(Model)
	if !got.answered {
		t.Fatal("space should answer the current quiz question")
	}
}
