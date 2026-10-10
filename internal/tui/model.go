package tui

import (
	"cmp"
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/Syntax-Syllogism/mk/internal/board"
	"github.com/charmbracelet/colorprofile"
)

type mode int

const (
	modeBoard mode = iota
	modeTable
	modePreview
	modeFilter
	modeHelp
	modeLogs
)

type cachedBody struct {
	hash   [32]byte
	length int
	lower  string
}
type reloadMsg struct{}
type tickMsg struct{}
type watcherError struct{ err error }
type Options struct {
	Logs    *LogSink
	Version string
	NoMouse bool
}
type Model struct {
	store                       *board.Store
	snap                        board.Snapshot
	deps                        board.DepInfo
	sections                    []board.SectionInfo
	filter                      board.Filter
	mode, prevMode, overlayMode mode
	focusSec                    int
	selected                    map[string]string
	scroll                      map[string]int
	tableCur                    string
	tableScroll                 int
	filterRow, filterChip       int
	preview                     previewState
	overlay                     viewport.Model
	prefs                       Prefs
	prefsFile                   string
	width, height               int
	version                     string
	logs                        *LogSink
	logSequence                 int
	toast                       string
	toastUntil                  time.Time
	dark                        bool
	toastError                  bool
	grab                        *grabState
	tags                        *tagState
	undo                        []undoEntry
	search                      *textinput.Model
	noMouse                     bool
	layout                      layout
	renderX, renderY            int
	bodyCache                   map[string]cachedBody
}

func NewModel(store *board.Store, opts Options) *Model {
	if opts.Logs == nil {
		opts.Logs = NewLogSink()
	}
	m := &Model{store: store, selected: map[string]string{}, scroll: map[string]int{}, width: 80, height: 24, logs: opts.Logs, version: opts.Version, dark: true}
	m.noMouse = opts.NoMouse
	m.clearFilters()
	m.prefsFile = prefsPath(store.Dir())
	m.prefs = loadPrefs(m.prefsFile)
	if m.prefs.View == "table" {
		m.mode = modeTable
	}
	m.overlay = viewport.New()
	m.preview.viewport = viewport.New()
	m.reload()
	return m
}
func pulse() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}
func (m *Model) Init() tea.Cmd { return tea.Batch(pulse(), tea.RequestBackgroundColor) }
func Run(store *board.Store, opts Options) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := tea.NewProgram(NewModel(store, opts), tea.WithColorProfile(colorprofile.ANSI))
	done := make(chan error, 1)
	go func() {
		err := store.Watch(ctx, func() { p.Send(reloadMsg{}) })
		done <- err
		if err != nil {
			p.Send(watcherError{err})
		}
	}()
	_, err := p.Run()
	cancel()
	watchErr := <-done
	if err != nil {
		return err
	}
	return watchErr
}
func (m *Model) notify(text string) {
	if m.toastError && time.Now().Before(m.toastUntil) {
		return
	}
	m.toastError = false
	m.toast = text
	m.toastUntil = time.Now().Add(time.Second)
}
func (m *Model) clearFilters() {
	m.filter = board.Filter{Projects: map[string]bool{}, Epics: map[string]bool{}, Tags: map[string]bool{}}
}
func (m *Model) matches(t board.Task) bool {
	chips := m.filter
	chips.Query = ""
	return chips.Visible(t) && m.filter.MatchesQuery(t, m.bodyCache[t.File].lower)
}
func (m *Model) visible(section string) []board.Task {
	var tasks []board.Task
	for _, t := range m.snap.Tasks {
		if board.Section(t, m.snap.Columns) == section && m.matches(t) {
			tasks = append(tasks, t)
		}
	}
	slices.SortFunc(tasks, board.ByOrder)
	return tasks
}
func (m *Model) tableTasks() []board.Task {
	var tasks []board.Task
	for _, t := range m.snap.Tasks {
		if m.matches(t) {
			tasks = append(tasks, t)
		}
	}
	tasks = board.SortForTable(tasks, m.snap.Columns)
	if strings.TrimSpace(m.filter.Query) != "" {
		slices.SortStableFunc(tasks, func(a, b board.Task) int { return cmp.Compare(m.filter.Score(b), m.filter.Score(a)) })
	}
	return tasks
}
func fileIndex(tasks []board.Task, file string) int {
	for i, t := range tasks {
		if t.File == file {
			return i
		}
	}
	return -1
}
func selectedFile(tasks []board.Task, file string, index int) string {
	if len(tasks) == 0 {
		return ""
	}
	if fileIndex(tasks, file) >= 0 {
		return file
	}
	return tasks[max(0, min(index, len(tasks)-1))].File
}
func (m *Model) reload() {
	indices := map[string]int{}
	focusName := ""
	if len(m.sections) > 0 {
		focusName = m.sections[m.focusSec].Name
	}
	for _, s := range m.sections {
		indices[s.Name] = fileIndex(m.visible(s.Name), m.selected[s.Name])
	}
	tableIndex := fileIndex(m.tableTasks(), m.tableCur)
	m.snap = m.store.Snapshot()
	cache := make(map[string]cachedBody, len(m.snap.Tasks))
	for _, t := range m.snap.Tasks {
		hash := sha256.Sum256([]byte(t.Content))
		entry, ok := m.bodyCache[t.File]
		if !ok || entry.hash != hash || entry.length != len(t.Content) {
			entry = cachedBody{hash, len(t.Content), strings.ToLower(t.Content)}
		}
		cache[t.File] = entry
	}
	m.bodyCache = cache
	m.sections = board.Sections(m.snap)
	m.deps = board.AnalyzeDeps(m.snap.Tasks)
	m.pruneFilters()
	m.focusSec = max(0, min(m.focusSec, len(m.sections)-1))
	for i, s := range m.sections {
		if s.Name == focusName {
			m.focusSec = i
		}
		m.selected[s.Name] = selectedFile(m.visible(s.Name), m.selected[s.Name], indices[s.Name])
	}
	m.tableCur = selectedFile(m.tableTasks(), m.tableCur, tableIndex)
	if m.mode == modePreview {
		m.refreshPreview()
	}
	m.refreshGrab()
}
func (m *Model) repairSelection() {
	for _, s := range m.sections {
		m.selected[s.Name] = selectedFile(m.visible(s.Name), m.selected[s.Name], 0)
	}
	m.tableCur = selectedFile(m.tableTasks(), m.tableCur, 0)
	if len(m.sections) > 0 && len(m.visible(m.sections[m.focusSec].Name)) == 0 {
		for i := m.focusSec + 1; i < len(m.sections); i++ {
			if len(m.visible(m.sections[i].Name)) > 0 {
				m.focusSec = i
				return
			}
		}
		for i := m.focusSec - 1; i >= 0; i-- {
			if len(m.visible(m.sections[i].Name)) > 0 {
				m.focusSec = i
				return
			}
		}
	}
}
func (m *Model) currentFile() string {
	if m.mode == modeTable {
		return m.tableCur
	}
	if len(m.sections) == 0 {
		return ""
	}
	return m.selected[m.sections[m.focusSec].Name]
}
func (m *Model) moveCard(key string) {
	var tasks []board.Task
	file := m.currentFile()
	if m.mode == modeTable {
		tasks = m.tableTasks()
	} else if len(m.sections) > 0 {
		tasks = m.visible(m.sections[m.focusSec].Name)
	}
	if len(tasks) == 0 {
		return
	}
	i := max(0, fileIndex(tasks, file))
	step := max(1, m.contentHeight()/2)
	switch key {
	case "j", "down":
		i++
	case "k", "up":
		i--
	case "g":
		i = 0
	case "G":
		i = len(tasks) - 1
	case "ctrl+d":
		i += step
	case "ctrl+u":
		i -= step
	}
	i = max(0, min(i, len(tasks)-1))
	if m.mode == modeTable {
		m.tableCur = tasks[i].File
	} else {
		m.selected[m.sections[m.focusSec].Name] = tasks[i].File
	}
}
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(1, msg.Width)
		m.height = max(1, msg.Height)
		if m.search != nil {
			m.search.SetWidth(max(1, m.width-3))
		}
		if m.preview.task.File != "" {
			m.renderPreview()
		}
		m.sizeOverlay()
	case tea.BackgroundColorMsg:
		m.dark = msg.IsDark()
		m.preview.cacheKey = ""
		if m.preview.task.File != "" {
			m.renderPreview()
		}
	case reloadMsg:
		m.notify("synced")
		m.reload()
	case editorDoneMsg:
		m.editorDone(msg)
	case watcherError:
		m.logs.Logf("Watcher stopped: %v", msg.err)
		m.notify(fmt.Sprintf("Watcher stopped: %v", msg.err))
	case tickMsg:
		entries, seq := m.logs.snapshot()
		if seq != m.logSequence && len(entries) > 0 {
			m.notify(entries[len(entries)-1])
			m.logSequence = seq
			if m.mode == modeLogs {
				m.overlay.SetContent(m.warningText())
			}
		}
		return m, pulse()
	case tea.MouseMsg:
		m.updateMouse(msg)
	case tea.KeyPressMsg:
		k := msg.String()
		if m.mode == modeHelp || m.mode == modeLogs {
			if k == "ctrl+c" {
				if m.grab != nil {
					m.grab = nil
					m.mode = m.overlayMode
					return m, nil
				}
				if m.tags != nil {
					m.tags = nil
					m.mode = m.overlayMode
					return m, nil
				}
				return m, tea.Quit
			}
			if k == "esc" || k == "q" || k == "?" || k == "!" {
				m.mode = m.overlayMode
				return m, nil
			}
			var cmd tea.Cmd
			m.overlay, cmd = m.overlay.Update(msg)
			return m, cmd
		}
		if k == "?" {
			m.overlayMode = m.mode
			m.mode = modeHelp
			m.sizeOverlay()
			m.overlay.SetContent(m.actionHelp())
			m.overlay.GotoTop()
			return m, nil
		}
		if m.search != nil {
			return m, m.updateSearch(msg)
		}
		if m.tags != nil {
			return m, m.updateTags(msg)
		}
		if m.grab != nil {
			m.updateGrab(k)
			return m, nil
		}
		if k == "ctrl+c" {
			return m, tea.Quit
		}

		if k == "!" {
			m.overlayMode = m.mode
			m.mode = modeLogs
			text := m.warningText()
			m.sizeOverlay()
			m.overlay.SetContent(text)
			m.overlay.GotoTop()
			return m, nil
		}
		if k == "r" {
			return m, m.rescan()
		}
		if m.mode == modePreview {
			switch k {
			case "t":
				return m, m.startTags(m.preview.task.File)
			case "e":
				return m, m.editFile(m.preview.task.File)
			case "y":
				return m, m.copyFile(m.preview.task.File)
			}
			return m, m.updatePreview(msg)
		}
		if m.mode == modeBoard || m.mode == modeTable {
			switch k {
			case "/":
				return m, m.startSearch()
			case "esc":
				m.filter.Query = ""
				m.repairSelection()
				return m, nil
			case "n", "N":
				m.nextMatch(k == "N")
				return m, nil
			case "t":
				return m, m.startTags(m.currentFile())
			case "e":
				return m, m.editFile(m.currentFile())
			case "y":
				return m, m.copyFile(m.currentFile())
			case "u":
				m.undoAction()
				return m, nil
			case "space", " ":
				if m.mode == modeBoard {
					m.startGrab()
				}
				return m, nil
			case "H", "L", "J", "K":
				if m.mode == modeBoard {
					m.quickMove(k)
				}
				return m, nil
			}
		}
		switch k {
		case "q":
			return m, tea.Quit
		case "F":
			m.clearFilters()
			m.repairSelection()
		case "ctrl+f":
			m.prefs.FiltersCollapsed = !m.prefs.FiltersCollapsed
			if m.mode == modeFilter && m.prefs.FiltersCollapsed {
				m.mode = m.prevMode
			}
			savePrefs(m.prefsFile, m.prefs)
		case "f":
			if m.mode == modeFilter {
				m.mode = m.prevMode
			} else {
				m.prevMode = m.mode
				m.mode = modeFilter
				m.prefs.FiltersCollapsed = false
				savePrefs(m.prefsFile, m.prefs)
			}
		default:
			if m.mode == modeFilter {
				m.updateFilters(k)
				return m, nil
			}
			switch k {
			case "v":
				if m.mode == modeBoard {
					m.mode = modeTable
					m.tableCur = selectedFile(m.tableTasks(), m.currentBoardFile(), 0)
					m.prefs.View = "table"
				} else {
					m.mode = modeBoard
					m.selectBoard(m.tableCur)
					m.prefs.View = "board"
				}
				savePrefs(m.prefsFile, m.prefs)
			case "z":
				m.prefs.Compact = !m.prefs.Compact
				savePrefs(m.prefsFile, m.prefs)
			case "h", "left":
				m.focusSec = max(0, m.focusSec-1)
			case "l", "right":
				m.focusSec = max(0, min(len(m.sections)-1, m.focusSec+1))
			case "enter":
				if file := m.currentFile(); file != "" {
					m.prevMode = m.mode
					m.mode = modePreview
					m.preview.back = nil
					m.openPreview(file)
				}
			default:
				m.moveCard(k)
			}
		}
	default:
		if m.search != nil {
			return m, m.updateSearchInput(msg)
		}
		if m.tags != nil {
			var cmd tea.Cmd
			m.tags.input, cmd = m.tags.input.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}
func (m *Model) currentBoardFile() string {
	if len(m.sections) == 0 {
		return ""
	}
	return m.selected[m.sections[m.focusSec].Name]
}
func (m *Model) selectBoard(file string) {
	for _, t := range m.snap.Tasks {
		if t.File == file {
			for i, s := range m.sections {
				if board.Section(t, m.snap.Columns) == s.Name {
					m.focusSec = i
					m.selected[s.Name] = file
					return
				}
			}
		}
	}
}
func (m *Model) sizeOverlay() { m.overlay.SetWidth(m.width); m.overlay.SetHeight(max(1, m.height-2)) }

func (m *Model) rescan() tea.Cmd {
	store, logs := m.store, m.logs
	return func() tea.Msg {
		if err := store.Scan(); err != nil {
			logs.Logf("Scan: %v", err)
		}
		if err := store.ReloadConfig(); err != nil {
			logs.Logf("Config: %v", err)
		}
		return reloadMsg{}
	}
}
