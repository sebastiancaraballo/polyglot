package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/sebastiancaraballo/polyglot/internal/content"
	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/screens/menu"
	"github.com/sebastiancaraballo/polyglot/internal/screens/profilesetup"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func newTestAppContext(t *testing.T, store storage.Storage) appContext {
	t.Helper()
	course, err := content.LoadEmbedded(content.DefaultPair)
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	return appContext{store: store, course: course, theme: ui.PlainTheme(), msgs: i18n.ES}
}

func TestWipeAndResetClearsData(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "polyglot.db")
	store, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Seed an onboarded profile with progress.
	profile, err := store.CreateProfile(ctx, "tester", "")
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

	if root.ctx.profile.ID != 0 {
		t.Errorf("profile after wipe id = %d, want zero profile before setup", root.ctx.profile.ID)
	}
	if !root.setupTutorial {
		t.Error("wipe should return to first-run profile setup")
	}
	if _, ok := root.screen.(profilesetup.Model); !ok {
		t.Fatalf("screen after wipe = %T, want profilesetup.Model", root.screen)
	}
	profiles, err := root.ctx.store.ListProfiles(ctx)
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	if len(profiles) != 0 {
		t.Errorf("profiles after wipe = %d, want 0 before setup", len(profiles))
	}
}

func TestNewRootWithoutProfileStartsProfileSetup(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "polyglot.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	root := newRoot(newTestAppContext(t, store))
	if !root.setupTutorial {
		t.Error("first run should mark profile setup as tutorial")
	}
	if _, ok := root.screen.(profilesetup.Model); !ok {
		t.Fatalf("initial screen = %T, want profilesetup.Model", root.screen)
	}
}

func TestProfileCreatedWithoutTutorialSetsOnboardedAndMenu(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(filepath.Join(t.TempDir(), "polyglot.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	profile, err := store.CreateProfile(ctx, "Mei", "identicon:0")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	root := newRoot(newTestAppContext(t, store))

	updated, _ := root.profileCreated(nav.ProfileCreatedMsg{ID: profile.ID, Tutorial: false})
	got := updated.(rootModel)
	if !got.ctx.profile.Onboarded {
		t.Error("additional profiles should be marked onboarded immediately")
	}
	if _, ok := got.screen.(menu.Model); !ok {
		t.Fatalf("screen = %T, want menu.Model", got.screen)
	}
	id, ok, err := store.GetActiveProfileID(ctx)
	if err != nil || !ok || id != profile.ID {
		t.Fatalf("active profile = (%d, %v, %v), want (%d, true, nil)", id, ok, err, profile.ID)
	}
}

func TestDeleteActiveProfileSwitchesToRemainingProfile(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(filepath.Join(t.TempDir(), "polyglot.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	first, err := store.CreateProfile(ctx, "Ana", "initials:0")
	if err != nil {
		t.Fatalf("CreateProfile first: %v", err)
	}
	second, err := store.CreateProfile(ctx, "Mei", "identicon:1")
	if err != nil {
		t.Fatalf("CreateProfile second: %v", err)
	}
	if err := store.SetOnboarded(ctx, first.ID); err != nil {
		t.Fatalf("SetOnboarded first: %v", err)
	}
	if err := store.SetOnboarded(ctx, second.ID); err != nil {
		t.Fatalf("SetOnboarded second: %v", err)
	}
	appCtx := newTestAppContext(t, store)
	appCtx.profile = first
	root := newRoot(appCtx)

	updated, _ := root.deleteActiveProfile()
	got := updated.(rootModel)
	if got.ctx.profile.ID != second.ID {
		t.Fatalf("active profile after delete = %d, want %d", got.ctx.profile.ID, second.ID)
	}
	if _, err := store.GetProfile(ctx, first.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("deleted profile lookup = %v, want ErrNotFound", err)
	}
	if _, ok := got.screen.(menu.Model); !ok {
		t.Fatalf("screen = %T, want menu.Model", got.screen)
	}
}

func TestDeleteLastProfileReturnsToProfileSetup(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(filepath.Join(t.TempDir(), "polyglot.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	profile, err := store.CreateProfile(ctx, "Ana", "initials:0")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	appCtx := newTestAppContext(t, store)
	appCtx.profile = profile
	root := newRoot(appCtx)

	updated, _ := root.deleteActiveProfile()
	got := updated.(rootModel)
	if got.ctx.profile.ID != 0 {
		t.Fatalf("active profile after deleting last = %d, want zero", got.ctx.profile.ID)
	}
	if !got.setupTutorial {
		t.Error("deleting the last profile should return to first-run setup")
	}
	if _, ok := got.screen.(profilesetup.Model); !ok {
		t.Fatalf("screen = %T, want profilesetup.Model", got.screen)
	}
}
