package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPrefs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "prefs.json")
	want := Prefs{View: "table", Compact: false, FiltersCollapsed: true}
	savePrefs(path, want)
	if got := loadPrefs(path); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if err := os.WriteFile(path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := loadPrefs(path); !reflect.DeepEqual(got, defaultPrefs()) {
		t.Fatalf("corrupt prefs: %+v", got)
	}
	// A regular file cannot be used as the parent directory, including as root.
	blocked := filepath.Join(path, "prefs.json")
	savePrefs(blocked, want)
	if got := loadPrefs(blocked); !reflect.DeepEqual(got, defaultPrefs()) {
		t.Fatalf("unwritable prefs: %+v", got)
	}
}
func TestPrefsPerDirectory(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	a := prefsPath("/tmp/a")
	if a == prefsPath("/tmp/b") || a != prefsPath("/tmp/a") {
		t.Fatal("state keys must be stable and per directory")
	}
	if len(filepath.Base(a)) != 21 {
		t.Fatalf("key must have 16 hex characters: %s", a)
	}
}
