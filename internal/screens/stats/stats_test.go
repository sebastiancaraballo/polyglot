package stats

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/x/exp/golden"

	"github.com/sebastiancaraballo/polyglot/internal/content"
	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func TestStatsGolden(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	profile, err := store.CreateProfile(ctx, "tester", "")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	if err := store.SaveCardState(ctx, profile.ID, model.CardState{
		CardID: "greetings:1", Reps: 1, Ease: model.DefaultEase, DueAt: now, LastReviewedAt: now,
	}); err != nil {
		t.Fatalf("SaveCardState: %v", err)
	}
	if err := store.SaveStats(ctx, profile.ID, model.Stats{Streak: 5, BestStreak: 12, LastStudiedAt: now, XP: 1240}); err != nil {
		t.Fatalf("SaveStats: %v", err)
	}

	course := &content.Course{
		Pair: "es-ja",
		Lessons: []model.Lesson{{
			ID: "greetings", Title: "Saludos", JLPT: model.N5,
			Cards: []model.Card{{ID: "greetings:1"}, {ID: "greetings:2"}},
		}},
		Kana: []model.KanaItem{
			{Char: "あ", Romaji: "a", Type: model.Hiragana},
			{Char: "ア", Romaji: "a", Type: model.Katakana},
		},
	}

	m := New(Deps{
		Theme: ui.PlainTheme(), Msgs: i18n.ES, Store: store,
		ProfileID: profile.ID, Course: course,
	})
	golden.RequireEqual(t, []byte(m.View().Content))
}
