package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"path/filepath"

	"github.com/Syntax-Syllogism/mk/internal/board"
	"github.com/Syntax-Syllogism/mk/internal/tui"
)

func runTUI(args []string) error {
	fs := flag.NewFlagSet("mk tui", flag.ContinueOnError)
	dir := fs.String("dir", "docs", "directory containing markdown tasks")
	noMouse := fs.Bool("no-mouse", false, "disable mouse capture for native terminal selection")
	ver := fs.Bool("version", false, "print version information and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *ver {
		fmt.Printf("mk %s (commit %s, built %s)\n", version, commit, date)
		return nil
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("mk tui: unexpected argument %q", fs.Arg(0))
	}
	logs := tui.NewLogSink()
	store, err := board.Open(*dir, board.Options{CachePath: filepath.Join(".kanban", "board.json"), Logf: logs.Logf})
	if err != nil {
		return err
	}
	output := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(output)
	return tui.Run(store, tui.Options{Logs: logs, Version: version, NoMouse: *noMouse})
}
