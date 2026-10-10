package main

import (
	"embed"
	"fmt"
	"os"
)

//go:embed index.html
var indexHTML embed.FS

// Set via -ldflags at build time (see .goreleaser.yaml).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) > 0 && args[0] == "tui" {
		return runTUI(args[1:])
	}
	return runServe(args)
}
