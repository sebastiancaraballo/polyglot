package storage

import (
	"context"
	"testing"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

func TestKanaProgressRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	profile, err := store.CreateProfile(ctx, "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	// An untouched profile has no kana progress.
	got, err := store.GetKanaProgress(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetKanaProgress: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("fresh profile progress = %v, want empty", got)
	}

	want := model.KanaProgress{Char: "あ", Streak: 2, Attempts: 5, Mastered: true, BestMs: 1200}
	if err := store.SaveKanaProgress(ctx, profile.ID, want); err != nil {
		t.Fatalf("SaveKanaProgress: %v", err)
	}

	got, err = store.GetKanaProgress(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetKanaProgress: %v", err)
	}
	if got["あ"] != want {
		t.Fatalf("round-trip = %+v, want %+v", got["あ"], want)
	}

	// Saving again upserts rather than duplicating.
	want.Streak, want.Attempts = 3, 6
	if err := store.SaveKanaProgress(ctx, profile.ID, want); err != nil {
		t.Fatalf("SaveKanaProgress (update): %v", err)
	}
	got, err = store.GetKanaProgress(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetKanaProgress: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("progress entries = %d, want 1 (upsert)", len(got))
	}
	if got["あ"].Streak != 3 {
		t.Errorf("Streak after update = %d, want 3", got["あ"].Streak)
	}
}

func TestKanaProgressDeletedWithProfile(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	profile, err := store.CreateProfile(ctx, "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if err := store.SaveKanaProgress(ctx, profile.ID,
		model.KanaProgress{Char: "あ", Mastered: true}); err != nil {
		t.Fatalf("SaveKanaProgress: %v", err)
	}
	if err := store.DeleteProfile(ctx, profile.ID); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}

	var n int
	if err := store.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM kana_progress WHERE profile_id = ?`, profile.ID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("kana_progress rows after profile delete = %d, want 0 (cascade)", n)
	}
}
