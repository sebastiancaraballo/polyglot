package profiles

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/x/exp/golden"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func TestProfilesGolden(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	first, err := store.CreateProfile(ctx, "Ana", "initials:0")
	if err != nil {
		t.Fatalf("CreateProfile first: %v", err)
	}
	if _, err := store.CreateProfile(ctx, "Mei", "identicon:1"); err != nil {
		t.Fatalf("CreateProfile second: %v", err)
	}

	m := New(Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES, Store: store, ActiveID: first.ID})
	golden.RequireEqual(t, []byte(m.View().Content))
}
