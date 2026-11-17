package tui

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Prefs struct {
	Collapsed        []string `json:"collapsed,omitempty"`
	View             string   `json:"view"`
	Compact          bool     `json:"compact"`
	FiltersCollapsed bool     `json:"filtersCollapsed"`
}

func defaultPrefs() Prefs { return Prefs{View: "board", Compact: true} }
func prefsPath(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	var root string
	if runtime.GOOS == "linux" {
		root = os.Getenv("XDG_STATE_HOME")
		if root == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return ""
			}
			root = filepath.Join(home, ".local", "state")
		}
		root = filepath.Join(root, "mk")
	} else {
		config, err := os.UserConfigDir()
		if err != nil {
			return ""
		}
		root = filepath.Join(config, "mk", "state")
	}
	return filepath.Join(root, fmt.Sprintf("%x", sha256.Sum256([]byte(abs)))[:16]+".json")
}
func loadPrefs(path string) Prefs {
	p := defaultPrefs()
	data, err := os.ReadFile(path)
	if err != nil {
		return p
	}
	if json.Unmarshal(data, &p) != nil || (p.View != "board" && p.View != "table") {
		return defaultPrefs()
	}
	return p
}
func savePrefs(path string, p Prefs) {
	if path == "" {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0700) != nil {
		return
	}
	data, err := json.Marshal(p)
	if err != nil {
		return
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".prefs-*")
	if err != nil {
		return
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return
	}
	if f.Close() != nil {
		return
	}
	_ = os.Rename(f.Name(), path)
}
