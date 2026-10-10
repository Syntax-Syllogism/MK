package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	tea "charm.land/bubbletea/v2"
	"github.com/google/shlex"
)

type editorDoneMsg struct {
	file string
	err  error
}

func editorCmd(env func(string) string, goos, path string) (*exec.Cmd, error) {
	value := env("VISUAL")
	if value == "" {
		value = env("EDITOR")
	}
	if value == "" {
		value = "vi"
		if goos == "windows" {
			value = "notepad"
		}
	}
	args, err := shlex.Split(value)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("empty editor command")
	}
	return exec.Command(args[0], append(args[1:], path)...), nil
}
func (m *Model) editFile(file string) tea.Cmd {
	if _, ok := m.task(file); !ok {
		return nil
	}
	cmd, err := editorCmd(os.Getenv, runtime.GOOS, filepath.Join(m.snap.Dir, file))
	if err != nil {
		m.actionError(err)
		return nil
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return editorDoneMsg{file, err} })
}
func (m *Model) editorDone(msg editorDoneMsg) {
	scanErr := m.store.Scan()
	m.reload()
	m.focusFile(msg.file)
	if m.mode == modePreview {
		m.refreshPreview()
	}
	if msg.err != nil {
		m.actionError(fmt.Errorf("editor exited: %w", msg.err))
	} else if scanErr != nil {
		m.actionError(scanErr)
	}
}
