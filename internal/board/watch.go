package board

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"path/filepath"
	"strings"
	"time"
)

// Watch blocks until cancellation, coalescing relevant changes with a trailing debounce.
func (s *Store) Watch(ctx context.Context, onChange func()) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	if err := watcher.Add(s.dir); err != nil {
		return err
	}
	return s.watchEvents(ctx, watcher, onChange)
}
func (s *Store) watchEvents(ctx context.Context, watcher *fsnotify.Watcher, onChange func()) error {
	var timer *time.Timer
	var tick <-chan time.Time
	var scan, config bool
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			name := filepath.Base(event.Name)
			if strings.HasSuffix(name, ".md") {
				scan = true
			} else if name == ConfigFileName {
				config = true
			} else {
				continue
			}
			if timer == nil {
				timer = time.NewTimer(50 * time.Millisecond)
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(50 * time.Millisecond)
			}
			tick = timer.C
		case <-tick:
			tick = nil
			if scan {
				if err := s.Scan(); err != nil {
					s.logf("Scan error: %v", err)
				}
			}
			if config {
				if err := s.ReloadConfig(); err != nil {
					s.logf("Config reload error: %v", err)
				}
			}
			scan, config = false, false
			if onChange != nil {
				onChange()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			s.logf("Watcher error: %v", err)
		}
	}
}
