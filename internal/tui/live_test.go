package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherReloadKeepsSelection(t *testing.T) {
	m := newTestModel(t)
	press(m, "G")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	messages := make(chan reloadMsg, 1)
	done := make(chan error, 1)
	go func() {
		done <- m.store.Watch(ctx, func() {
			select {
			case messages <- reloadMsg{}:
			default:
			}
		})
	}()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(2 * time.Second)
	defer timeout.Stop()
	// Retry writes until Watch has attached; this avoids assuming goroutine scheduling.
	for {
		select {
		case <-ticker.C:
			if err := os.WriteFile(filepath.Join(m.store.Dir(), "external.md"), []byte("---\nstatus: TODO\n---\nExternal change"), 0600); err != nil {
				t.Fatal(err)
			}
			// Stop after the first write long enough for the trailing debounce to fire.
			ticker.Reset(150 * time.Millisecond)
		case msg := <-messages:
			m.Update(msg)
			if _, ok := m.task("external.md"); !ok {
				t.Fatal("watcher update did not refresh TUI snapshot")
			}
			if m.currentFile() != "c.md" {
				t.Fatal("watcher update lost filename selection")
			}
			cancel()
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			return
		case err := <-done:
			t.Fatalf("watcher exited early: %v", err)
		case <-timeout.C:
			t.Fatal("external change did not reach TUI")
		}
	}
}
