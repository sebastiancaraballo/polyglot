package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sebastiancaraballo/polyglot/internal/content"
	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func TestWipeAndResetClearsData(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "polyglot.db")
	store, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Seed an onboarded profile with progress.
	profile, err := store.CreateProfile(ctx, "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if err := store.SetOnboarded(ctx, profile.ID); err != nil {
		t.Fatalf("SetOnboarded: %v", err)
	}
	if err := store.AddXP(ctx, profile.ID, 100); err != nil {
		t.Fatalf("AddXP: %v", err)
	}

	course, err := content.LoadEmbedded(content.DefaultPair)
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}

	m := rootModel{ctx: appContext{
		store: store, course: course, profile: profile,
		theme: ui.PlainTheme(), msgs: i18n.ES, dbPath: dbPath,
	}}

	updated, _ := m.wipeAndReset()
	root := updated.(rootModel)
	t.Cleanup(func() { _ = root.ctx.store.Close() })

	if root.ctx.profile.Onboarded {
		t.Error("fresh profile after wipe should not be onboarded")
	}
	stats, err := root.ctx.store.GetStats(ctx, root.ctx.profile.ID)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.XP != 0 {
		t.Errorf("XP after wipe = %d, want 0", stats.XP)
	}
	profiles, err := root.ctx.store.ListProfiles(ctx)
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	if len(profiles) != 1 {
		t.Errorf("profiles after wipe = %d, want 1 fresh default", len(profiles))
	}
}
