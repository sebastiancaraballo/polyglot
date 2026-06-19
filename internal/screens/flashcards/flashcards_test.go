package flashcards

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/srs"
)

func TestSpaceRevealsFlashcard(t *testing.T) {
	card := model.Card{ID: "test:1", Source: "Gracias", JP: "ありがとう", Romaji: "arigatou"}
	m := Model{
		queue:  []model.Card{card},
		states: map[string]model.CardState{card.ID: srs.NewCard(card.ID)},
	}

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	got := next.(Model)
	if !got.revealed {
		t.Fatal("space should reveal the current flashcard")
	}
}
