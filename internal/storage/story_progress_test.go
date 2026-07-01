package storage

import (
	"context"
	"testing"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

func TestStoryProgressRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	profile, err := store.CreateProfile(ctx, "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	// An untouched profile has no story progress.
	got, err := store.GetStoryProgress(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetStoryProgress: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("fresh profile progress = %v, want empty", got)
	}

	want := model.StoryProgress{ChapterID: "capitulo-1-asakusa", BeatIndex: 2, Completed: false}
	if err := store.SaveStoryProgress(ctx, profile.ID, want); err != nil {
		t.Fatalf("SaveStoryProgress: %v", err)
	}

	got, err = store.GetStoryProgress(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetStoryProgress: %v", err)
	}
	if got["capitulo-1-asakusa"] != want {
		t.Fatalf("round-trip = %+v, want %+v", got["capitulo-1-asakusa"], want)
	}

	// Saving again upserts rather than duplicating.
	want.BeatIndex, want.Completed = 5, true
	if err := store.SaveStoryProgress(ctx, profile.ID, want); err != nil {
		t.Fatalf("SaveStoryProgress (update): %v", err)
	}
	got, err = store.GetStoryProgress(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetStoryProgress: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("progress entries = %d, want 1 (upsert)", len(got))
	}
	if !got["capitulo-1-asakusa"].Completed {
		t.Error("chapter should be marked completed after update")
	}
}

func TestStoryProgressDeletedWithProfile(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	profile, err := store.CreateProfile(ctx, "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if err := store.SaveStoryProgress(ctx, profile.ID,
		model.StoryProgress{ChapterID: "capitulo-1-asakusa", BeatIndex: 1}); err != nil {
		t.Fatalf("SaveStoryProgress: %v", err)
	}
	if err := store.DeleteProfile(ctx, profile.ID); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}

	var n int
	if err := store.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM story_progress WHERE profile_id = ?`, profile.ID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("story_progress rows after profile delete = %d, want 0 (cascade)", n)
	}
}
