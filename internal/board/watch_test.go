package board

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchDebouncesRapidWrites(t *testing.T) {
	s := testStore(t)
	// Attach synchronously to avoid guessing when a goroutine has called Add.
	// This exercises the same event loop as Watch with real fsnotify events.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()
	if err := watcher.Add(s.Dir()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes := make(chan Snapshot, 20)
	done := make(chan error, 1)
	go func() { done <- s.watchEvents(ctx, watcher, func() { changes <- s.Snapshot() }) }()
	for i := 0; i < 10; i++ {
		writeTaskFile(t, s.Dir(), "card.md", "---\nstatus: TODO\n---\nbody")
	}
	// Config and task changes in the same burst must both flush once.
	writeTaskFile(t, s.Dir(), ConfigFileName, "columns: [REVIEW, DONE]\n")
	select {
	case snapshot := <-changes:
		if len(snapshot.Tasks) != 1 || !equalStrings(snapshot.Columns, []string{"REVIEW", "DONE"}) {
			t.Fatalf("incomplete flush: %+v", snapshot)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no change notification")
	}
	select {
	case <-changes:
		t.Fatal("rapid writes were not coalesced")
	case <-time.After(200 * time.Millisecond):
	}
	if err := os.Remove(filepath.Join(s.Dir(), "card.md")); err != nil {
		t.Fatal(err)
	}
	select {
	case snapshot := <-changes:
		if len(snapshot.Tasks) != 0 {
			t.Fatal("remove not scanned")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no remove notification")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("watch did not stop")
	}
}
func TestWatchCancellationAndSetupError(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Watch(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(s.Dir()); err != nil {
		t.Fatal(err)
	}
	if err := s.Watch(context.Background(), nil); err == nil {
		t.Fatal("missing directory accepted")
	}
}
