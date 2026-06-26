package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

func TestProfileNameRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	created, err := store.CreateProfile(ctx, "Mei")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if created.Name != "Mei" {
		t.Errorf("created name = %q, want %q", created.Name, "Mei")
	}

	got, err := store.GetProfile(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if got.Name != "Mei" {
		t.Errorf("GetProfile = %+v, want name Mei", got)
	}
}

func TestDeleteProfileCascades(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	p, err := store.CreateProfile(ctx, "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	now := time.Now().UTC()
	if err := store.SaveCardState(ctx, p.ID, model.CardState{
		CardID: "c1", Reps: 1, Ease: model.DefaultEase, DueAt: now, LastReviewedAt: now,
	}); err != nil {
		t.Fatalf("SaveCardState: %v", err)
	}

	if err := store.DeleteProfile(ctx, p.ID); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}
	if _, err := store.GetProfile(ctx, p.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetProfile after delete = %v, want ErrNotFound", err)
	}
	if _, err := store.GetStats(ctx, p.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetStats after delete = %v, want ErrNotFound (cascade)", err)
	}
	if err := store.DeleteProfile(ctx, 9999); !errors.Is(err, ErrNotFound) {
		t.Errorf("DeleteProfile unknown = %v, want ErrNotFound", err)
	}
}

func TestShowRomaji(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	p, err := store.CreateProfile(ctx, "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if !p.ShowRomaji {
		t.Errorf("new profile ShowRomaji = false, want true (default on)")
	}

	if err := store.SetShowRomaji(ctx, p.ID, false); err != nil {
		t.Fatalf("SetShowRomaji: %v", err)
	}
	got, err := store.GetProfile(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if got.ShowRomaji {
		t.Errorf("ShowRomaji after disable = true, want false")
	}

	if err := store.SetShowRomaji(ctx, p.ID, true); err != nil {
		t.Fatalf("SetShowRomaji re-enable: %v", err)
	}
	if got, _ := store.GetProfile(ctx, p.ID); !got.ShowRomaji {
		t.Errorf("ShowRomaji after re-enable = false, want true")
	}

	if err := store.SetShowRomaji(ctx, 9999, false); !errors.Is(err, ErrNotFound) {
		t.Errorf("SetShowRomaji unknown = %v, want ErrNotFound", err)
	}
}

func TestKanaOnboarded(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	p, err := store.CreateProfile(ctx, "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if p.KanaOnboarded {
		t.Errorf("new profile KanaOnboarded = true, want false (default off)")
	}

	if err := store.SetKanaOnboarded(ctx, p.ID); err != nil {
		t.Fatalf("SetKanaOnboarded: %v", err)
	}
	got, err := store.GetProfile(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if !got.KanaOnboarded {
		t.Errorf("KanaOnboarded after set = false, want true")
	}

	if err := store.SetKanaOnboarded(ctx, 9999); !errors.Is(err, ErrNotFound) {
		t.Errorf("SetKanaOnboarded unknown = %v, want ErrNotFound", err)
	}
}

func TestActiveProfileID(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, ok, err := store.GetActiveProfileID(ctx); err != nil || ok {
		t.Fatalf("fresh GetActiveProfileID = ok %v err %v, want ok=false", ok, err)
	}

	p, err := store.CreateProfile(ctx, "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if err := store.SetActiveProfileID(ctx, p.ID); err != nil {
		t.Fatalf("SetActiveProfileID: %v", err)
	}
	id, ok, err := store.GetActiveProfileID(ctx)
	if err != nil || !ok || id != p.ID {
		t.Errorf("GetActiveProfileID = (%d, %v, %v), want (%d, true, nil)", id, ok, err, p.ID)
	}

	// Overwrites on second set.
	if err := store.SetActiveProfileID(ctx, p.ID+1); err != nil {
		t.Fatalf("SetActiveProfileID again: %v", err)
	}
	if id, _, _ := store.GetActiveProfileID(ctx); id != p.ID+1 {
		t.Errorf("active id after overwrite = %d, want %d", id, p.ID+1)
	}
}
