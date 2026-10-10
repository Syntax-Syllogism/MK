package tui

import (
	"fmt"
	"sync"
)

type LogSink struct {
	mu       sync.Mutex
	entries  []string
	sequence int
}

func NewLogSink() *LogSink { return &LogSink{} }
func (l *LogSink) Logf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, fmt.Sprintf(format, args...))
	l.sequence++
	if len(l.entries) > 50 {
		l.entries = l.entries[len(l.entries)-50:]
	}
}
func (l *LogSink) snapshot() ([]string, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.entries...), l.sequence
}
