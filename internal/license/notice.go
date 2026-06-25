package license

import (
	"fmt"
	"sort"
	"strings"
)

const noticeHeader = `Polyglot
Copyright (c) 2026 Sebastián Caraballo

Polyglot is MIT-licensed (see LICENSE). This NOTICE lists the third-party assets
bundled in or derived into this repository, with their licenses and the
attributions those licenses require. In-house authored content is covered by the
repository's MIT license and is not listed here.

This file is GENERATED from internal/license/assets.yaml.
Regenerate it with: go run ./tools/gennotice
`

// RenderNOTICE returns the full, deterministic contents of the repo-root NOTICE
// file describing every third-party asset in the manifest (sorted by id).
func RenderNOTICE(assets []Asset) string {
	sorted := append([]Asset(nil), assets...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })

	var b strings.Builder
	b.WriteString(noticeHeader)
	for _, a := range sorted {
		b.WriteString("\n")
		b.WriteString(strings.Repeat("-", 76))
		b.WriteString("\n")
		fmt.Fprintf(&b, "%s\n", a.Name)
		fmt.Fprintf(&b, "  License:     %s\n", a.License)
		fmt.Fprintf(&b, "  Source:      %s\n", a.Source)
		fmt.Fprintf(&b, "  Attribution: %s\n", a.Attribution)
		if a.Notes != "" {
			fmt.Fprintf(&b, "  Notes:       %s\n", a.Notes)
		}
	}
	return b.String()
}
