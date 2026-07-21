# MK // Markdown Kanban

**MK** is a localhost Kanban board that uses your filesystem as its database.
It's designed for developers who want the visualization of a Kanban board with
the portability and version-control benefits of Markdown files.

This README covers installation and setup. For day-to-day usage — dragging
cards, filters, table view, configuring columns, resolving CONFLICT/OTHER — see
the [User Guide](USER_GUIDE.md).

## 🚀 Key Features

- **Single Binary**: No Node.js, no Docker, no external database. Just one file.
- **Filesystem Database**: Your tasks are stored as `.md` files in a `docs/`
  folder.
- **Real-time Sync**: Uses SSE (Server-Sent Events) to reflect file changes in
  the browser instantly (<200ms).
- **Edit & Preview**: Integrated Markdown editor and live preview within the web
  app.
- **Dynamic Columns**: Board columns come from a committed `docs/.kanban.yml`,
  editable by hand; unrecognized or missing statuses collect in a diagnostic
  `OTHER` column so no task is ever invisible.
- **Drag-and-Drop**: Move cards between columns and reorder them within a
  column; both the status and position are saved automatically.
- **Tag Filtering**: Quickly focus on specific categories with a dynamic filter
  bar.
- **Git Ready**: Built-in detection for Git conflict markers.
- **Auto-Port**: Automatically finds an available port starting from `8080`.

## 🛠️ Setup & Installation

### 1. Download/Build

If you have Go installed, you can build the binary yourself. First, fetch the
dependencies (`fsnotify` for filesystem watching and `yaml.v3` for parsing task
frontmatter):

```bash
go mod tidy
```

Then build:

```bash
# macOS / Linux
go build -o mk main.go
```

```powershell
# Windows
go build -o mk.exe main.go
```

### 2. Install Globally (Optional)

Move the binary to your system path to use it in any project:

```bash
# macOS / Linux
sudo mv mk /usr/local/bin/mk
```

```powershell
# Windows — run PowerShell as Administrator
Move-Item mk.exe C:\Windows\System32\mk.exe
```

Alternatively on Windows, move it to any folder already in your `%PATH%`, or add
a new folder via **System Properties → Environment Variables**.

### 3. Usage

Navigate to any project directory and run:

```bash
# macOS / Linux
mk
```

```powershell
# Windows — if installed globally
mk

# Windows — if running from the build directory
.\mk.exe
```

### Options

- **`-dir`**: Specify the directory containing your markdown tasks (default:
  `docs`).

  ```bash
  mk -dir my-tasks
  ```

- **`-port`**: Manually specify a port. If not provided, MK will try `8080` and
  then search for the next available port automatically.

  ```bash
  mk -port 9000
  ```

## 🧑‍💻 Developing MK

Building a binary (`go build`) is **not** a required step for local development. `go run`
compiles and runs in one command, and every flag works the same way:

```bash
go run main.go
go run main.go -dir my-tasks -port 9000
```

Use `go build` only when you want a standalone binary to keep around, install globally
(see "Install Globally" above), or hand to someone else who doesn't have Go installed.

There's no separate frontend build step, either. `index.html` is a single file — inline
`<style>` and `<script>`, no bundler, no npm, no node_modules — embedded directly into the
binary at compile time via `//go:embed index.html` (see the top of `main.go`). Edit it and
re-run `go run main.go`; there's no asset pipeline to run first. Note that this embedding
means a running `mk`/`go run` process serves whatever `index.html` looked like when it
started — editing it while the server is running doesn't hot-reload the page like task
file edits do (which use a real filesystem watcher, not `go:embed`). Restart the process to
pick up frontend changes.

`.kanban/board.json` (the generated task cache) and `docs/.kanban.yml` (the board's column
config, bootstrapped on first run if missing — see "Project Structure" below) are both
created automatically; you don't need to set up anything by hand before running against a
scratch `docs/` folder for testing.

### Tests

```bash
go test ./...
go vet ./...
```

`main_test.go` covers the Go backend: config loading/bootstrap, `/api/update` request
handling (including the CONFLICT/OTHER write guards), and frontmatter parsing. There is no
frontend test harness by design — UI behavior (drag-and-drop, filters, the modal) is
verified manually in a browser; see [USER_GUIDE.md](USER_GUIDE.md) for what "working"
looks like from that side.

### Manual smoke testing

`test-docs/` holds a set of QA fixture tickets — each one deliberately shaped to exercise
an edge case (a live Git conflict, malformed frontmatter, no frontmatter at all, and so
on). Run `scripts/reset-smoke-fixtures.sh` to copy them into `docs/`, then follow
`docs/document_smoke_test_plan.md` (it shows up as a ticket on the board itself). Because
using the fixtures — dragging, tagging, saving — is the point, it also tends to repair
them; `test-docs/` is the untouched source, `docs/`'s copies are disposable, and rerunning
the script resets them. Don't run MK against `test-docs/` directly, and don't hand-edit the
fixtures in `docs/` expecting the fix to stick.

## 📂 Project Structure

- **`<tasks_dir>/*.md`**: Your task files. Each file should have a YAML
  frontmatter.
- **`<tasks_dir>/.kanban.yml`**: Committed config listing board columns in
  display order. Bootstrapped automatically on first run if absent.
- **`.kanban/board.json`**: A generated index file used for fast UI rendering.
- **`test-docs/`**: Pristine QA smoke-test fixtures — see "Manual smoke testing" above.
- **`scripts/reset-smoke-fixtures.sh`**: Copies `test-docs/` fixtures into `docs/`.

### Task Schema (Example)

Each `.md` file in the `docs/` folder should follow this format:

```markdown
---
project: MK
status: TODO
epic: SETUP
tags: [backend, go]
order: 1
---

# Task Title

Task description goes here...
```

`order` is optional and written automatically by the board when you drag a card
— you don't need to set it by hand.

## 🎨 Philosophy

1. **Plain Files First**: Your tasks are markdown you own, legible to any
   editor; the board is one view of them.
2. **Zero Dependency**: The tool should work on any machine without installing a
   runtime.
3. **Transparent State**: No hidden database. If you want to move a task, you
   can move the file or edit the text.

---

Built with Go and Vanilla JS.
