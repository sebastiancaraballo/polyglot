package flashcards

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/review"
	"github.com/sebastiancaraballo/polyglot/internal/srs"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func scheduled(it review.Item) review.Scheduled {
	return review.Scheduled{Item: it, State: srs.NewCard(it.CardID)}
}

func revealedModel(showRomaji bool) Model {
	it := review.Item{CardID: "test:1", Strand: review.Vocab, Prompt: "Gracias", Answer: "ありがとう", Secondary: "arigatou"}
	return Model{
		deps:     Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES, ShowRomaji: showRomaji},
		queue:    []review.Scheduled{scheduled(it)},
		revealed: true,
	}
}

func TestRevealShowsRomajiWhenEnabled(t *testing.T) {
	got := revealedModel(true).cardView()
	if !strings.Contains(got, "arigatou") {
		t.Fatalf("expected romaji on the reveal, got %q", got)
	}
}

func TestRevealHidesRomajiWhenDisabled(t *testing.T) {
	got := revealedModel(false).cardView()
	if strings.Contains(got, "arigatou") {
		t.Fatalf("romaji should be hidden when disabled, got %q", got)
	}
	if !strings.Contains(got, "ありがとう") {
		t.Fatalf("japanese word should still show, got %q", got)
	}
}

func TestSpaceRevealsFlashcard(t *testing.T) {
	it := review.Item{CardID: "test:1", Prompt: "Gracias", Answer: "ありがとう"}
	m := Model{queue: []review.Scheduled{scheduled(it)}}

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	got := next.(Model)
	if !got.revealed {
		t.Fatal("space should reveal the current flashcard")
	}
}

func TestGradeOptionsRenderOnePerLine(t *testing.T) {
	m := Model{deps: Deps{Msgs: i18n.ES}}

	got := m.gradeOptions(model.CardState{Ease: model.DefaultEase})
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
