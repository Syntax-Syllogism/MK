package tui

import (
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
	osc52 "github.com/aymanbagabas/go-osc52/v2"
)

func copyPath(path string, env func(string) string, native func(string) error) (string, tea.Cmd) {
	if env("SSH_TTY") == "" && env("SSH_CONNECTION") == "" {
		if err := native(path); err == nil {
			return "copied " + path, nil
		}
	}
	sequence := osc52.New(path)
	if env("TMUX") != "" {
		sequence = sequence.Tmux()
	}
	return "copied via terminal (OSC 52)", tea.Raw(sequence.String())
}
func (m *Model) copyFile(file string) tea.Cmd {
	if _, ok := m.task(file); !ok {
		return nil
	}
	text, cmd := copyPath(filepath.Join(m.snap.Dir, file), os.Getenv, clipboard.WriteAll)
	m.notify(text)
	return cmd
}
