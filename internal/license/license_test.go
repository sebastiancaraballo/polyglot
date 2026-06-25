package license

import (
	"os"
	"slices"
	"testing"
)

func TestManifestValid(t *testing.T) {
	assets, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(assets) == 0 {
		t.Fatal("expected at least one asset in the manifest")
	}

	seen := make(map[string]bool, len(assets))
	for _, a := range assets {
		if err := a.Validate(); err != nil {
			t.Errorf("invalid asset: %v", err)
		}
		if seen[a.ID] {
			t.Errorf("duplicate asset id %q", a.ID)
		}
		seen[a.ID] = true
	}
}

// TestExternalAssetsAreRegistered guards that every shipped third-party asset
// keeps its provenance entry. If someone removes or renames an entry, this fails
// in CI alongside the asset it documents.
func TestExternalAssetsAreRegistered(t *testing.T) {
	assets, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// (consumer-path that must appear in some asset's used_by).
	required := []string{
		"tools/genglobe",           // public-domain Natural Earth coastlines
		"content/ja/frequency.tsv", // Tatoeba-derived frequency list
	}
	for _, want := range required {
		if !slices.ContainsFunc(assets, func(a Asset) bool { return slices.Contains(a.UsedBy, want) }) {
			t.Errorf("no manifest asset is registered as used by %q", want)
		}
	}
}

// TestNoticeInSync is the drift guard: the committed repo-root NOTICE must equal
// what the manifest renders. Regenerate with: go run ./tools/gennotice
func TestNoticeInSync(t *testing.T) {
	assets, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := RenderNOTICE(assets)

	got, err := os.ReadFile("../../NOTICE")
	if err != nil {
		t.Fatalf("read NOTICE: %v", err)
	}
	if string(got) != want {
		t.Errorf("NOTICE is out of sync with internal/license/assets.yaml; run `go run ./tools/gennotice`")
	}
}

func TestPermittedLicense(t *testing.T) {
	tests := map[string]bool{
		"Public Domain": true,
		"CC0-1.0":       true,
		"MIT":           true,
		"BSD-2-Clause":  true,
		"BSD-3-Clause":  true,
		"Apache-2.0":    true,
		"CC-BY-4.0":     true,
		"CC-BY-2.0-FR":  true,
		"CC-BY-SA-4.0":  false, // share-alike
		"CC-BY-NC-4.0":  false, // non-commercial
		"GPL-3.0":       false,
		"LGPL-2.1":      false,
		"AGPL-3.0":      false,
		"":              false,
		"Proprietary":   false,
	}
	for id, want := range tests {
		if got := PermittedLicense(id); got != want {
			t.Errorf("PermittedLicense(%q) = %v, want %v", id, got, want)
		}
	}
}
