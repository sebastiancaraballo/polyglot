// Package license records and validates the provenance of every third-party
// asset bundled into or derived into the repository, so the project's MIT
// promise holds for assets as well as code (see CLAUDE.md "Hard constraints").
//
// The machine-readable manifest (assets.yaml) is the single source of truth:
// the repo-root NOTICE file is generated from it (tools/gennotice), and a test
// (license_test.go) fails CI if any asset is missing provenance, carries a
// non-permissive license, or if NOTICE has drifted from the manifest.
package license

import (
	_ "embed"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed assets.yaml
var manifestYAML []byte

// Asset is one third-party resource and its provenance. In-house authored
// content is not represented here — it is covered by the repo's MIT LICENSE.
type Asset struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Kind        string   `yaml:"kind"` // data | font | image | audio | generated
	License     string   `yaml:"license"`
	Source      string   `yaml:"source"`
	Attribution string   `yaml:"attribution"`
	UsedBy      []string `yaml:"used_by"`
	Notes       string   `yaml:"notes"`
}

type manifest struct {
	Assets []Asset `yaml:"assets"`
}

var validKinds = map[string]bool{
	"data":      true,
	"font":      true,
	"image":     true,
	"audio":     true,
	"generated": true,
}

// Load parses and returns the embedded third-party asset manifest.
func Load() ([]Asset, error) {
	var m manifest
	if err := yaml.Unmarshal(manifestYAML, &m); err != nil {
		return nil, fmt.Errorf("parse asset manifest: %w", err)
	}
	return m.Assets, nil
}

// Validate reports whether the asset records complete provenance and carries a
// license on Polyglot's permissive allowlist.
func (a Asset) Validate() error {
	switch {
	case a.ID == "":
		return fmt.Errorf("asset is missing id")
	case a.Name == "":
		return fmt.Errorf("asset %q is missing name", a.ID)
	case !validKinds[a.Kind]:
		return fmt.Errorf("asset %q has invalid kind %q", a.ID, a.Kind)
	case a.License == "":
		return fmt.Errorf("asset %q is missing license", a.ID)
	case a.Source == "":
		return fmt.Errorf("asset %q is missing source", a.ID)
	case a.Attribution == "":
		return fmt.Errorf("asset %q is missing attribution", a.ID)
	case len(a.UsedBy) == 0:
		return fmt.Errorf("asset %q is missing used_by", a.ID)
	}
	if !PermittedLicense(a.License) {
		return fmt.Errorf("asset %q: license %q is not on the permissive allowlist", a.ID, a.License)
	}
	return nil
}

// permitted lists the exact license identifiers always allowed.
var permitted = map[string]bool{
	"Public Domain": true,
	"CC0-1.0":       true,
	"MIT":           true,
	"BSD-2-Clause":  true,
	"BSD-3-Clause":  true,
	"Apache-2.0":    true,
}

// PermittedLicense reports whether a license identifier is on Polyglot's
// permissive allowlist (see CLAUDE.md "Hard constraints"). The CC-BY family is
// allowed only when it carries neither a NonCommercial (-NC) nor a ShareAlike
// (-SA) restriction; copyleft (GPL/LGPL/AGPL) and unknown licenses are rejected.
func PermittedLicense(id string) bool {
	if permitted[id] {
		return true
	}
	up := strings.ToUpper(id)
	if strings.Contains(up, "-NC") || strings.Contains(up, "-SA") {
		return false
	}
	if strings.HasPrefix(up, "GPL") || strings.HasPrefix(up, "LGPL") || strings.HasPrefix(up, "AGPL") {
		return false
	}
	return strings.HasPrefix(up, "CC-BY-")
}
