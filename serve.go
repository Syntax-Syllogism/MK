package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/Syntax-Syllogism/mk/internal/board"
)

type server struct {
	store     *board.Store
	clientsMu sync.Mutex
	clients   map[chan string]bool
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("mk", flag.ExitOnError)
	portFlag := fs.Int("port", 0, "port to listen on (default: 8080 or next available)")
	dirFlag := fs.String("dir", "docs", "directory containing markdown tasks")
	versionFlag := fs.Bool("version", false, "print version information and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *versionFlag {
		fmt.Printf("mk %s (commit %s, built %s)\n", version, commit, date)
		return nil
	}
	store, err := board.Open(*dirFlag, board.Options{CachePath: filepath.Join(".kanban", "board.json")})
	if err != nil {
		return err
	}
	srv := &server{store: store, clients: make(map[chan string]bool)}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watchResult := make(chan error, 1)
	go func() { watchResult <- store.Watch(ctx, func() { srv.broadcast("update") }) }()
	startPort := *portFlag
	if startPort == 0 {
		startPort = 8080
	}
	listener, err := findListener(startPort)
	if err != nil {
		return fmt.Errorf("Could not find an available port: %w", err)
	}
	defer listener.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/board", srv.handleBoard)
	mux.HandleFunc("/api/update", srv.handleUpdate)
	mux.HandleFunc("/api/events", srv.handleEvents)
	mux.HandleFunc("/", serveIndex)
	fmt.Printf("MK // Kanban starting on http://localhost:%d\n", listener.Addr().(*net.TCPAddr).Port)
	fmt.Printf("Watching directory: %s\n", *dirFlag)
	serveResult := make(chan error, 1)
	go func() { serveResult <- http.Serve(listener, mux) }()
	select {
	case err := <-watchResult:
		if err != nil {
			return fmt.Errorf("Watcher error: %w", err)
		}
		return fmt.Errorf("watcher stopped unexpectedly")
	case err := <-serveResult:
		return err
	}
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

func (s *server) handleUpdate(w http.ResponseWriter, r *http.Request) {
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

	err := s.store.Apply(board.UpdateRequest{File: req.File, Content: req.Content, Status: req.Status, Order: req.Order, Tags: req.Tags})
	if err != nil {
		code := http.StatusInternalServerError
		switch {
		case errors.Is(err, board.ErrMissingFile), errors.Is(err, board.ErrInvalidFile), errors.Is(err, board.ErrConflictStatus), errors.Is(err, board.ErrOtherStatus), errors.Is(err, board.ErrNothingToUpdate):
			code = http.StatusBadRequest
		case errors.Is(err, board.ErrEmptyClobber):
			code = http.StatusConflict
		}
		http.Error(w, err.Error(), code)
		return
	}
	w.WriteHeader(http.StatusOK)
}
func (s *server) broadcast(msg string) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	for client := range s.clients {
		// A browser may disconnect between the context cancellation that
		// triggers handleEvents's cleanup and its removal from clients. Never
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

func (s *server) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Buffer one update so the filesystem watcher is never coupled to a
	// browser's network write. broadcast coalesces additional updates while
	// this client is already waiting to refresh.
	messageChan := make(chan string, 1)
	s.clientsMu.Lock()
	s.clients[messageChan] = true
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, messageChan)
		s.clientsMu.Unlock()
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

func (s *server) handleBoard(w http.ResponseWriter, r *http.Request) {
	snapshot := s.store.Snapshot()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Columns []string     `json:"columns"`
		Tasks   []board.Task `json:"tasks"`
		DocsDir string       `json:"docsDir"`
	}{snapshot.Columns, snapshot.Tasks, snapshot.Dir})
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
