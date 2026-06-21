package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return store
}

func TestOpenAppliesMigrations(t *testing.T) {
	store := newTestStore(t)

	want := []string{"profiles", "card_states", "stats"}
	for _, table := range want {
		var name string
		err := store.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table,
		).Scan(&name)
		if err != nil {
			t.Fatalf("expected table %q to exist: %v", table, err)
		}
	}
}

func TestProfileLifecycle(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	created, err := store.CreateProfile(ctx, "Sebastián")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected a non-zero profile id")
	}
	if created.Onboarded {
		t.Error("new profile should not be onboarded")
	}

	got, err := store.GetProfile(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if got.Name != "Sebastián" {
		t.Errorf("name = %q, want %q", got.Name, "Sebastián")
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, created.CreatedAt)
	}

	if err := store.SetOnboarded(ctx, created.ID); err != nil {
		t.Fatalf("SetOnboarded: %v", err)
	}
	got, err = store.GetProfile(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetProfile after onboarding: %v", err)
	}
	if !got.Onboarded {
		t.Error("profile should be onboarded after SetOnboarded")
	}

	profiles, err := store.ListProfiles(ctx)
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("len(profiles) = %d, want 1", len(profiles))
	}
}

func TestGetProfileNotFound(t *testing.T) {
	store := newTestStore(t)
	_, err := store.GetProfile(context.Background(), 999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestSetOnboardedNotFound(t *testing.T) {
	store := newTestStore(t)
	err := store.SetOnboarded(context.Background(), 999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestCardStateRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	profile, err := store.CreateProfile(ctx, "learner")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	if _, err := store.GetCardState(ctx, profile.ID, "card-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetCardState (missing) err = %v, want ErrNotFound", err)
	}

	due := time.Date(2026, 6, 20, 9, 0, 0, 0, time.UTC)
	reviewed := time.Date(2026, 6, 19, 9, 0, 0, 0, time.UTC)
	state := model.CardState{
		CardID:         "card-1",
		Interval:       1,
		Ease:           model.DefaultEase,
		Reps:           1,
		Lapses:         0,
		DueAt:          due,
		LastReviewedAt: reviewed,
	}
	if err := store.SaveCardState(ctx, profile.ID, state); err != nil {
		t.Fatalf("SaveCardState: %v", err)
	}

	got, err := store.GetCardState(ctx, profile.ID, "card-1")
	if err != nil {
		t.Fatalf("GetCardState: %v", err)
	}
	if got.Interval != 1 || got.Reps != 1 || got.Ease != model.DefaultEase {
		t.Errorf("got %+v, want interval=1 reps=1 ease=%v", got, model.DefaultEase)
	}
	if !got.DueAt.Equal(due) || !got.LastReviewedAt.Equal(reviewed) {
		t.Errorf("times not preserved: due=%v reviewed=%v", got.DueAt, got.LastReviewedAt)
	}

	// Update via upsert.
	state.Interval = 3
	state.Reps = 2
	if err := store.SaveCardState(ctx, profile.ID, state); err != nil {
		t.Fatalf("SaveCardState (update): %v", err)
	}
	got, err = store.GetCardState(ctx, profile.ID, "card-1")
	if err != nil {
		t.Fatalf("GetCardState (after update): %v", err)
	}
	if got.Interval != 3 || got.Reps != 2 {
		t.Errorf("after update got interval=%d reps=%d, want 3 and 2", got.Interval, got.Reps)
	}
}

func TestStatsRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	profile, err := store.CreateProfile(ctx, "learner")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	// A fresh profile has zero-valued stats with no last-studied time.
	stats, err := store.GetStats(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.Streak != 0 || stats.BestStreak != 0 || !stats.LastStudiedAt.IsZero() || stats.XP != 0 {
		t.Errorf("fresh stats = %+v, want zero values", stats)
	}

	studied := time.Date(2026, 6, 19, 20, 0, 0, 0, time.UTC)
	if err := store.SaveStats(ctx, profile.ID, model.Stats{Streak: 5, BestStreak: 12, LastStudiedAt: studied, XP: 150}); err != nil {
		t.Fatalf("SaveStats: %v", err)
	}

	stats, err = store.GetStats(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetStats after save: %v", err)
	}
	if stats.Streak != 5 || stats.BestStreak != 12 || stats.XP != 150 {
		t.Errorf("got streak=%d best=%d xp=%d, want 5, 12 and 150", stats.Streak, stats.BestStreak, stats.XP)
	}
	if !stats.LastStudiedAt.Equal(studied) {
		t.Errorf("LastStudiedAt = %v, want %v", stats.LastStudiedAt, studied)
	}
}

func TestAddXP(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	profile, err := store.CreateProfile(ctx, "learner")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	if err := store.AddXP(ctx, profile.ID, 10); err != nil {
		t.Fatalf("AddXP: %v", err)
	}
	if err := store.AddXP(ctx, profile.ID, 5); err != nil {
		t.Fatalf("AddXP: %v", err)
	}

	stats, err := store.GetStats(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.XP != 15 {
		t.Errorf("XP after adding 10 then 5 = %d, want 15", stats.XP)
	}

	if err := store.AddXP(ctx, 9999, 10); !errors.Is(err, ErrNotFound) {
		t.Errorf("AddXP for unknown profile = %v, want ErrNotFound", err)
	}
}

func TestCountLearnedCards(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	profile, err := store.CreateProfile(ctx, "learner")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	if n, err := store.CountLearnedCards(ctx, profile.ID); err != nil || n != 0 {
		t.Fatalf("CountLearnedCards = (%d, %v), want (0, nil)", n, err)
	}

	now := time.Now().UTC()
	// One learned card (reps > 0) and one brand-new card (reps == 0).
	if err := store.SaveCardState(ctx, profile.ID, model.CardState{CardID: "a", Ease: model.DefaultEase, Reps: 1, DueAt: now, LastReviewedAt: now}); err != nil {
		t.Fatalf("SaveCardState a: %v", err)
	}
	if err := store.SaveCardState(ctx, profile.ID, model.CardState{CardID: "b", Ease: model.DefaultEase, Reps: 0, DueAt: now, LastReviewedAt: now}); err != nil {
		t.Fatalf("SaveCardState b: %v", err)
	}

	if n, err := store.CountLearnedCards(ctx, profile.ID); err != nil || n != 1 {
		t.Fatalf("CountLearnedCards = (%d, %v), want (1, nil)", n, err)
	}
}

func TestCardStateCascadeIsolation(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	a, err := store.CreateProfile(ctx, "a")
	if err != nil {
		t.Fatalf("CreateProfile a: %v", err)
	}
	b, err := store.CreateProfile(ctx, "b")
	if err != nil {
		t.Fatalf("CreateProfile b: %v", err)
	}

	now := time.Now().UTC()
	state := model.CardState{CardID: "shared", Ease: model.DefaultEase, DueAt: now, LastReviewedAt: now}
	if err := store.SaveCardState(ctx, a.ID, state); err != nil {
		t.Fatalf("SaveCardState a: %v", err)
	}

	// Profile b must not see profile a's card state.
	if _, err := store.GetCardState(ctx, b.ID, "shared"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("profile b GetCardState err = %v, want ErrNotFound", err)
	}
}
