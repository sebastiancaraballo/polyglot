package flashcards

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/review"
	"github.com/sebastiancaraballo/polyglot/internal/srs"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
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

// TestNewSurfacesHeldBackCount seeds more new cards than the pacing budget
// admits and checks the screen both caps the session and says why.
func TestNewSurfacesHeldBackCount(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	profile, err := store.CreateProfile(context.Background(), "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	var items []review.Item
	for i := 0; i < 15; i++ {
		id := fmt.Sprintf("v:%d", i)
		items = append(items, review.Item{CardID: id, Strand: review.Vocab, Prompt: id, Answer: id})
	}
	m := New(Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES, Store: store, ProfileID: profile.ID, Items: items})
	if len(m.queue) != 10 {
		t.Fatalf("queue = %d, want 10 (paced new-card intake)", len(m.queue))
	}
	if m.heldBackNew != 5 {
		t.Fatalf("heldBackNew = %d, want 5", m.heldBackNew)
	}
	if !strings.Contains(m.cardView(), "5 tarjetas nuevas en espera") {
		t.Fatal("card view should state how many new cards were held back and why")
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
