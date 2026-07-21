package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

//go:embed index.html
var indexHTML embed.FS

type Task struct {
	File    string   `json:"file"`
	Status  string   `yaml:"status" json:"status"`
	Project string   `yaml:"project" json:"project"`
	Epic    string   `yaml:"epic" json:"epic"`
	Order   *float64 `yaml:"order,omitempty" json:"order,omitempty"`
	Tags    []string `yaml:"tags" json:"tags"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	// ParseError is set only when the file's frontmatter failed to parse; it
	// is never read from or written to YAML. Content still holds the file's
	// real body in this case, so opening the card and saving it back (e.g.
	// via the tag editor) round-trips safely instead of clobbering the body
	// with an error message.
	ParseError string `json:"parseError,omitempty"`
}

type BoardState struct {
	Tasks []Task `json:"tasks"`
	mu    sync.RWMutex
}

type BoardConfig struct {
	Columns []string `yaml:"columns"`
}

const configFileName = ".kanban.yml"

var config = &BoardConfig{}
var configMu sync.RWMutex

var board = &BoardState{}
var kanbanDir = ".kanban"
var boardFile = filepath.Join(kanbanDir, "board.json")

// Set via -ldflags at build time (see .goreleaser.yaml).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// SSE Clients
var clients = make(map[chan string]bool)
var clientsMu sync.Mutex

func main() {
	portFlag := flag.Int("port", 0, "port to listen on (default: 8080 or next available)")
	dirFlag := flag.String("dir", "docs", "directory containing markdown tasks")
	versionFlag := flag.Bool("version", false, "print version information and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("mk %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	docsDir := *dirFlag

	// Ensure directories exist
	if _, err := os.Stat(docsDir); os.IsNotExist(err) {
		os.MkdirAll(docsDir, 0755)
	}
	if _, err := os.Stat(kanbanDir); os.IsNotExist(err) {
		os.Mkdir(kanbanDir, 0755)
	}

	// Initial scan
	if err := scanDocs(docsDir, boardFile); err != nil {
		log.Printf("Initial scan error: %v", err)
	}

	// Bootstrap the board column config if absent, then load it. Both must
	// happen before watcher.Add below so a bootstrap write is never observed
	// by fsnotify and does not trigger a spurious SSE broadcast.
	if err := bootstrapIfAbsent(docsDir); err != nil {
		log.Printf("Config bootstrap error: %v", err)
	}
	if err := loadConfig(docsDir); err != nil {
		log.Printf("Config load error: %v", err)
	}

	// Setup watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write ||
					event.Op&fsnotify.Create == fsnotify.Create ||
					event.Op&fsnotify.Remove == fsnotify.Remove ||
					event.Op&fsnotify.Rename == fsnotify.Rename {
					name := filepath.Base(event.Name)
					if strings.HasSuffix(name, ".md") {
						log.Printf("File changed: %s", event.Name)
						if err := scanDocs(docsDir, boardFile); err != nil {
							log.Printf("Scan error: %v", err)
						}
						broadcast("update")
					} else if name == configFileName {
						log.Printf("Config changed: %s", event.Name)
						if err := loadConfig(docsDir); err != nil {
							log.Printf("Config reload error: %v", err)
						}
						broadcast("update")
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(docsDir)
	if err != nil {
		log.Fatal(err)
	}

	// Find available port
	startPort := *portFlag
	if startPort == 0 {
		startPort = 8080
	}

	listener, err := findListener(startPort)
	if err != nil {
		log.Fatalf("Could not find an available port: %v", err)
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port

	// HTTP Server
	http.HandleFunc("/api/board", func(w http.ResponseWriter, r *http.Request) {
		getBoard(w, r)
	})
	http.HandleFunc("/api/update", func(w http.ResponseWriter, r *http.Request) {
		updateTask(w, r, docsDir)
	})
	http.HandleFunc("/api/events", sseHandler)
	http.HandleFunc("/", serveIndex)

	fmt.Printf("MK // Kanban starting on http://localhost:%d\n", actualPort)
	fmt.Printf("Watching directory: %s\n", docsDir)
	log.Fatal(http.Serve(listener, nil))
}

func findListener(startPort int) (net.Listener, error) {
	for port := startPort; port < startPort+100; port++ {
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			return l, nil
		}
	}
	return nil, fmt.Errorf("no available ports in range %d-%d", startPort, startPort+99)
}

func updateTask(w http.ResponseWriter, r *http.Request, docsDir string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		File    string   `json:"file"`
		Content *string  `json:"content"`
		Status  string   `json:"status"`
		Order   *float64 `json:"order"`
		Tags    []string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Status == "CONFLICT" {
		http.Error(w, "CONFLICT is a derived status and cannot be assigned", http.StatusBadRequest)
		return
	}
	if req.Status == "OTHER" {
		http.Error(w, "OTHER is a diagnostic bucket and cannot be assigned", http.StatusBadRequest)
		return
	}
	if req.Status == "" && req.Order == nil && req.Content == nil && req.Tags == nil {
		http.Error(w, "Nothing to update: at least one of status, order, content, or tags is required", http.StatusBadRequest)
		return
	}

	path, err := taskPath(docsDir, req.File)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	existing, err := ioutil.ReadFile(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	parts := strings.SplitN(string(existing), "---", 3)
	var newContent string
	// Mirrors parseTask's F25 fix: only treat the file as having frontmatter if
	// it actually starts with the delimiter, not merely contains two "---" lines.
	if strings.HasPrefix(string(existing), "---") && len(parts) >= 3 {
		// Parse frontmatter to update status if provided
		var fm map[string]interface{}
		if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
			http.Error(w, "Invalid frontmatter", http.StatusInternalServerError)
			return
		}
		if fm == nil {
			fm = map[string]interface{}{}
		}

		if req.Status != "" {
			fm["status"] = req.Status
		}
		if req.Order != nil {
			fm["order"] = *req.Order
		}
		if req.Tags != nil {
			fm["tags"] = req.Tags
		}

		updatedFM, _ := yaml.Marshal(fm)
		body := parts[2]
		// [P2] Distinguish between omitted and empty body
		if req.Content != nil {
			body = "\n" + *req.Content
		}
		newContent = "---\n" + string(updatedFM) + "---" + body
	} else {
		// No frontmatter, just update content or add frontmatter if status, order, or tags provided
		if req.Status != "" || req.Order != nil || req.Tags != nil {
			fm := map[string]interface{}{}
			if req.Status != "" {
				fm["status"] = req.Status
			}
			if req.Order != nil {
				fm["order"] = *req.Order
			}
			if req.Tags != nil {
				fm["tags"] = req.Tags
			}
			fmBytes, _ := yaml.Marshal(fm)
			// F26: default to the file's existing body, not "" — a drag (status
			// + order, no content key) must not wipe a frontmatter-less file.
			content := string(existing)
			if req.Content != nil {
				content = *req.Content
			}
			newContent = "---\n" + string(fmBytes) + "---\n" + content
		} else if req.Content != nil {
			newContent = *req.Content
		}
	}

	// Belt-and-braces: never let a computed-empty write clobber a non-empty file
	// when nobody actually asked for an empty body (req.Content == nil means the
	// caller sent no content key at all, not that they explicitly emptied it).
	if newContent == "" && len(existing) > 0 && req.Content == nil {
		http.Error(w, "Refusing to write an empty file over existing content", http.StatusConflict)
		return
	}

	if err := ioutil.WriteFile(path, []byte(newContent), 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func taskPath(docsDir, file string) (string, error) {
	if file == "" {
		return "", fmt.Errorf("missing file")
	}
	if file != filepath.Base(file) {
		return "", fmt.Errorf("invalid file")
	}
	return filepath.Join(docsDir, file), nil
}

func broadcast(msg string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for client := range clients {
		// A browser may disconnect between the context cancellation that
		// triggers sseHandler's cleanup and its removal from clients. Never
		// let that stale (or merely slow) connection block the watcher while
		// it holds clientsMu: doing so prevents every later file edit from
		// reaching healthy browsers too. Each SSE client has a one-message
		// buffer; if it is already waiting to refresh, dropping this duplicate
		// is safe because the eventual /api/board fetch reads the latest state.
		select {
		case client <- msg:
		default:
		}
	}
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Buffer one update so the filesystem watcher is never coupled to a
	// browser's network write. broadcast coalesces additional updates while
	// this client is already waiting to refresh.
	messageChan := make(chan string, 1)
	clientsMu.Lock()
	clients[messageChan] = true
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, messageChan)
		clientsMu.Unlock()
	}()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case msg := <-messageChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

// scanDocs re-reads docsDir into board.Tasks and writes a JSON cache of the
// result to cachePath, creating cachePath's parent directory if needed
// (callers, including tests, may point this at a scratch location rather
// than the real .kanban/ directory).
func scanDocs(docsDir, cachePath string) error {
	files, err := ioutil.ReadDir(docsDir)
	if err != nil {
		return err
	}

	var newTasks []Task
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
			task, err := parseTask(filepath.Join(docsDir, f.Name()))
			if err != nil {
				log.Printf("Error parsing %s: %v", f.Name(), err)
				continue
			}
			newTasks = append(newTasks, task)
		}
	}

	board.mu.Lock()
	board.Tasks = newTasks
	board.mu.Unlock()

	data, err := json.MarshalIndent(newTasks, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return err
	}
	return ioutil.WriteFile(cachePath, data, 0644)
}

func parseTask(path string) (Task, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return Task{}, err
	}

	task := Task{
		File:  filepath.Base(path),
		Title: strings.TrimSuffix(filepath.Base(path), ".md"),
	}

	// Check for Git conflict markers
	if strings.Contains(string(content), "<<<<<<<") {
		task.Status = "CONFLICT"
		task.Content = string(content)
		return task, nil
	}

	// Basic frontmatter parsing. Require the file to actually start with the
	// delimiter — SplitN alone would treat any two "---" lines anywhere in the
	// body as a frontmatter block (F25), misparsing plain notes that use "---"
	// as a thematic break.
	parts := strings.SplitN(string(content), "---", 3)
	if strings.HasPrefix(string(content), "---") && len(parts) >= 3 {
		var fm Task
		if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil ||
			(fm.Order != nil && (math.IsNaN(*fm.Order) || math.IsInf(*fm.Order, 0))) {
			// Malformed frontmatter (e.g. `order: soon`, or a scalar where a
			// list is expected) must not make the task disappear. Leave
			// task.Status empty so it lands in OTHER, and surface the error
			// via ParseError (not Epic — Epic feeds the filter bar's chip
			// list, and a real-looking "PARSE ERROR" epic would pollute it)
			// rather than Content — Content must stay the file's real body,
			// or opening the card and saving it back (e.g. via the tag
			// editor, which doesn't touch the textarea) would silently
			// overwrite that body with the error message.
			if err != nil {
				task.ParseError = err.Error()
			} else {
				task.ParseError = "order must be a finite number"
			}
			task.Content = strings.TrimSpace(parts[2])
			return task, nil
		}
		task.Status = fm.Status
		task.Project = fm.Project
		task.Epic = fm.Epic
		task.Order = fm.Order
		task.Tags = fm.Tags
		task.Content = strings.TrimSpace(parts[2])
	} else {
		task.Content = strings.TrimSpace(string(content))
	}

	return task, nil
}

func getBoard(w http.ResponseWriter, r *http.Request) {
	board.mu.RLock()
	tasks := board.Tasks
	board.mu.RUnlock()

	configMu.RLock()
	columns := config.Columns
	configMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Columns []string `json:"columns"`
		Tasks   []Task   `json:"tasks"`
	}{Columns: columns, Tasks: tasks})
}

// bootstrapIfAbsent writes docs/.kanban.yml from the current task set (via
// scanDocs, which must already have run) if the file does not already exist.
// Called exactly once, from main(), before the watcher is attached — this is
// a startup concern, not something loadConfig should redo on every reload.
func bootstrapIfAbsent(docsDir string) error {
	path := filepath.Join(docsDir, configFileName)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return bootstrapConfig(docsDir, path)
		}
		return err
	}
	return nil
}

// loadConfig reads docs/.kanban.yml and swaps it into the in-memory config.
// It is read-only: if the file is transiently absent (an editor's
// atomic-save rename, a git checkout/stash, a sync client), it logs a
// warning and leaves the current in-memory config untouched rather than
// regenerating the file, which would silently discard hand-edited columns.
func loadConfig(docsDir string) error {
	path := filepath.Join(docsDir, configFileName)

	data, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		log.Printf("Config warning: %s is missing; keeping current in-memory columns", path)
		return nil
	}
	if err != nil {
		return err
	}

	var loaded BoardConfig
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		return err
	}

	var cols []string
	seen := map[string]bool{}
	for _, c := range loaded.Columns {
		if c == "" || seen[c] {
			if seen[c] {
				log.Printf("Config warning: duplicate column %q is ignored", c)
			}
			continue
		}
		if c == "CONFLICT" || c == "OTHER" {
			log.Printf("Config warning: %q is a reserved column name and is ignored", c)
			continue
		}
		seen[c] = true
		cols = append(cols, c)
	}

	if len(cols) == 0 {
		// An empty file, an empty `columns:` list, or a list containing only
		// reserved names would otherwise leave the board with zero columns —
		// every card falls into the non-droppable OTHER bucket and the board
		// becomes silently read-only. Keep serving the last good config
		// instead (same treatment as a missing file, above). An editor that
		// truncates before writing can trigger this transiently.
		configMu.RLock()
		hasCurrent := len(config.Columns) > 0
		configMu.RUnlock()
		if !hasCurrent {
			configMu.Lock()
			config = &BoardConfig{Columns: []string{"TODO", "IN PROGRESS", "DONE"}}
			configMu.Unlock()
			log.Printf("Config warning: %s has no usable columns; using default columns", path)
			return nil
		}
		log.Printf("Config warning: %s has no usable columns; keeping current in-memory columns", path)
		return nil
	}

	configMu.Lock()
	config = &BoardConfig{Columns: cols}
	configMu.Unlock()

	return nil
}

// bootstrapConfig writes docs/.kanban.yml from the statuses seen in
// board.Tasks, in first-seen (filename) order, excluding CONFLICT and empty
// statuses. Falls back to TODO/IN PROGRESS/DONE when no statuses are found.
func bootstrapConfig(docsDir, path string) error {
	board.mu.RLock()
	tasks := board.Tasks
	board.mu.RUnlock()

	seen := map[string]bool{}
	var cols []string
	for _, t := range tasks {
		if t.Status == "" || t.Status == "CONFLICT" || seen[t.Status] {
			continue
		}
		seen[t.Status] = true
		cols = append(cols, t.Status)
	}

	if len(cols) == 0 {
		cols = []string{"TODO", "IN PROGRESS", "DONE"}
	}

	body, err := yaml.Marshal(BoardConfig{Columns: cols})
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("# MK board columns. Order here is the order on the board.\n")
	b.WriteString("# A card whose `status` matches none of these lands in OTHER.\n")
	b.Write(body)

	if err := ioutil.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return err
	}

	log.Printf("Bootstrapped %s with columns: %v", path, cols)
	return nil
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	data, err := indexHTML.ReadFile("index.html")
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}
