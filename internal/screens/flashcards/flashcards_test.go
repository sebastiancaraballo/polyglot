package flashcards

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
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

func TestGradeOptionsRenderOnePerLine(t *testing.T) {
	card := model.Card{ID: "test:1"}
	m := Model{
		deps:   Deps{Msgs: i18n.ES},
		states: map[string]model.CardState{card.ID: srs.NewCard(card.ID)},
	}

	got := m.gradeOptions(card)
	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Fatalf("grade options should render one option per line, got %d lines: %q", len(lines), got)
	}

	wantPrefixes := []string{"[1] Otra vez", "[2] Difícil", "[3] Bien", "[4] Fácil"}
	for i, want := range wantPrefixes {
		if !strings.HasPrefix(lines[i], want) {
			t.Fatalf("line %d should start with %q, got %q", i+1, want, lines[i])
		}
	}
}
