package storage

import (
	"context"
	"testing"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

func TestPatternProgressRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	profile, err := store.CreateProfile(ctx, "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	// An untouched profile has no pattern progress.
	got, err := store.GetPatternProgress(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetPatternProgress: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("fresh profile progress = %v, want empty", got)
	}

	want := model.PatternProgress{PatternID: "x-wa-n-desu", Slot: "X", Streak: 2, Attempts: 5, Mastered: true}
	if err := store.SavePatternProgress(ctx, profile.ID, want); err != nil {
		t.Fatalf("SavePatternProgress: %v", err)
	}

	got, err = store.GetPatternProgress(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetPatternProgress: %v", err)
	}
	if got["x-wa-n-desu:X"] != want {
		t.Fatalf("round-trip = %+v, want %+v", got["x-wa-n-desu:X"], want)
	}

	// Saving again upserts rather than duplicating.
	want.Streak, want.Attempts = 3, 6
	if err := store.SavePatternProgress(ctx, profile.ID, want); err != nil {
		t.Fatalf("SavePatternProgress (update): %v", err)
	}
	got, err = store.GetPatternProgress(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetPatternProgress: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("progress entries = %d, want 1 (upsert)", len(got))
	}
	if got["x-wa-n-desu:X"].Streak != 3 {
		t.Errorf("Streak after update = %d, want 3", got["x-wa-n-desu:X"].Streak)
	}
}

func TestPatternProgressTracksSlotsIndependently(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	profile, err := store.CreateProfile(ctx, "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	if err := store.SavePatternProgress(ctx, profile.ID,
		model.PatternProgress{PatternID: "x-wa-n-desu", Slot: "X", Mastered: true}); err != nil {
		t.Fatalf("SavePatternProgress: %v", err)
	}
	if err := store.SavePatternProgress(ctx, profile.ID,
		model.PatternProgress{PatternID: "x-wa-n-desu", Slot: "N", Mastered: false}); err != nil {
		t.Fatalf("SavePatternProgress: %v", err)
	}

	got, err := store.GetPatternProgress(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetPatternProgress: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("progress entries = %d, want 2", len(got))
	}
	if !got["x-wa-n-desu:X"].Mastered {
		t.Error("slot X should be mastered")
	}
	if got["x-wa-n-desu:N"].Mastered {
		t.Error("slot N should not be mastered")
	}
}

func TestPatternProgressDeletedWithProfile(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	profile, err := store.CreateProfile(ctx, "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if err := store.SavePatternProgress(ctx, profile.ID,
		model.PatternProgress{PatternID: "x-wa-n-desu", Slot: "X", Mastered: true}); err != nil {
		t.Fatalf("SavePatternProgress: %v", err)
	}
	if err := store.DeleteProfile(ctx, profile.ID); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}

	var n int
	if err := store.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pattern_progress WHERE profile_id = ?`, profile.ID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("pattern_progress rows after profile delete = %d, want 0 (cascade)", n)
	}
}

func TestGetCardStates(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	profile, err := store.CreateProfile(ctx, "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	got, err := store.GetCardStates(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetCardStates: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("fresh profile card states = %v, want empty", got)
	}

	a := model.CardState{CardID: "self-intro:1", Interval: 1, Ease: model.DefaultEase, Reps: 1}
	b := model.CardState{CardID: "self-intro:2", Interval: 2, Ease: model.DefaultEase, Reps: 3}
	if err := store.SaveCardState(ctx, profile.ID, a); err != nil {
		t.Fatalf("SaveCardState: %v", err)
	}
	if err := store.SaveCardState(ctx, profile.ID, b); err != nil {
		t.Fatalf("SaveCardState: %v", err)
	}

	got, err = store.GetCardStates(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetCardStates: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("card states = %d, want 2", len(got))
	}
	if got["self-intro:1"].Reps != 1 {
		t.Errorf("self-intro:1 Reps = %d, want 1", got["self-intro:1"].Reps)
	}
	if got["self-intro:2"].Reps != 3 {
		t.Errorf("self-intro:2 Reps = %d, want 3", got["self-intro:2"].Reps)
	}
}
