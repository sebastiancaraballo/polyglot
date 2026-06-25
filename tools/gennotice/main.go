// Command gennotice regenerates the repo-root NOTICE file from the third-party
// asset manifest (internal/license/assets.yaml).
//
// Run it after editing the manifest:
//
//	go run ./tools/gennotice
//
// The license test (internal/license) fails if NOTICE has drifted from the
// manifest, so this tool keeps the human-readable NOTICE in sync with the
// machine-readable source of truth.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sebastiancaraballo/polyglot/internal/license"
)

func main() {
	out := flag.String("o", "NOTICE", "output path for the generated NOTICE file")
	flag.Parse()

	if err := run(*out); err != nil {
		fmt.Fprintf(os.Stderr, "gennotice: %v\n", err)
		os.Exit(1)
	}
}

func run(out string) error {
	assets, err := license.Load()
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, []byte(license.RenderNOTICE(assets)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	fmt.Printf("gennotice: wrote %s (%d assets)\n", out, len(assets))
	return nil
}
