// Command polyglot is the entry point for the Polyglot terminal application.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sebastiancaraballo/polyglot/internal/app"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "0.0.1"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("polyglot %s\n", version)
		return
	}

	if err := app.Run(version); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
