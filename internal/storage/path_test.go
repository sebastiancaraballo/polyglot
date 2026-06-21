package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "polyglot.db")
	paths := []string{base, base + "-wal", base + "-shm"}
	for _, p := range paths {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatalf("WriteFile %q: %v", p, err)
		}
	}

	if err := Remove(base); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	for _, p := range paths {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%q still exists after Remove (err=%v)", p, err)
		}
	}
}

func TestRemoveMissingIsNoError(t *testing.T) {
	base := filepath.Join(t.TempDir(), "absent.db")
	if err := Remove(base); err != nil {
		t.Errorf("Remove on missing files = %v, want nil", err)
	}
}
