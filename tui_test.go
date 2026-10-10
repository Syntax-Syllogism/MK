package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTUIVersionDoesNotOpenStore(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	if err := run([]string{"tui", "-version", "-no-mouse", "-dir", dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("version must not open task directory")
	}
}
func TestTUIArgumentErrors(t *testing.T) {
	if err := runTUI([]string{"-unknown"}); err == nil {
		t.Fatal("unknown flags must fail")
	}
	if err := runTUI([]string{"unexpected"}); err == nil {
		t.Fatal("positional arguments must fail")
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runTUI([]string{"-dir", file}); err == nil {
		t.Fatal("directory errors must be returned before terminal startup")
	}
}
