---
title: Development
description: Build and run MK from source, run the Go checks, smoke-test against a scratch copy of the fixtures, and find your way around the repository.
---

# Development

This page is for people who change MK. For installing and using it, see
[Getting started](getting-started.md).

## Build and run

Run the whole package from the repository root:

```bash
go run . -dir /tmp/mk-smoke
go run . tui -dir /tmp/mk-smoke
go build -o mk .
```

`go run .` takes the same flags as the binary. See the [command line reference](cli.md).

Without `-dir`, MK uses `docs/`. This repository stores its own documentation
in `docs/`, so use a scratch task folder while you develop. Otherwise MK will
show these pages as cards.

There is no frontend build step. `index.html` holds the web board's CSS and
JavaScript, and [`main.go`](https://github.com/Syntax-Syllogism/MK/blob/master/main.go)
embeds it at compile time. Restart `go run .` after you edit it. Only task files
and `.kanban.yml` are watched live.

`main.go` declares the `version`, `commit`, and `date` variables that GoReleaser
sets with `-ldflags`. The [GoReleaser config](https://github.com/Syntax-Syllogism/MK/blob/master/.goreleaser.yaml)
builds the package from `.`, which includes `serve.go` and `tui.go`.

## Automated checks

```bash
go test ./...
go vet ./...
```

The test files cover these areas:

- `internal/board/*_test.go`: parsing, config, writes, batches, guarded restores,
  snapshots, watching, ordering, filters, and dependencies.
- `main_test.go`: the HTTP handlers and SSE, including error mapping and the
  CONFLICT and OTHER write guards.
- `tui_test.go`: TUI flag errors, and that `-version` does not open a store.
- `internal/tui/*_test.go`: layout, navigation, reloads, dependencies,
  preferences, terminal sizes, card actions, undo, and a check that keys which
  don't write never modify task files.

Run the search and mouse checks alone with:

```bash
go test -count=1 ./internal/tui -run 'Test(Search|Mouse|NoMouse|HelpBinding)'
```

Run the filter checks alone with:

```bash
go test -count=1 ./internal/board -run TestFilter
```

Board and table snapshots live in `internal/tui/testdata/`. After an intentional
rendering change, regenerate them and review the diff:

```bash
UPDATE_GOLDEN=1 go test ./internal/tui -run TestSampleScreens
```

Run `go test -race ./internal/...` when you change the store or the TUI's
concurrency.

There is no frontend test harness. Test the web board by hand, as described below.

## Manual smoke testing

[`test-work-items/`](https://github.com/Syntax-Syllogism/MK/blob/master/test-work-items)
is the committed sample board and QA fixture set. Don't run MK against it
directly. Dragging, tagging, TUI card actions, undo, and saving all rewrite
files. Copy it to a new scratch folder first:

```bash
cp -r test-work-items /tmp/mk-smoke
go run . -dir /tmp/mk-smoke
```

Pick a path that does not exist yet. If the folder already exists, the copy nests
the fixtures inside it instead of replacing them. To start over, delete the
scratch copy and copy the fixtures again.

Follow the [smoke-test plan](https://github.com/Syntax-Syllogism/MK/blob/master/test-work-items/document_smoke_test_plan.md)
on the scratch board. The plan is also a card there, in the `QA` epic.

For the terminal board, follow the checklist in
[Implementation and smoke testing](terminal-ui.md#implementation-and-smoke-testing).

Change `test-work-items/` only to add or change a QA scenario on purpose.

## Repository layout

- `main.go`: the CLI entry point, the embedded frontend, and the build version
  variables.
- `serve.go`: the web flags, the HTTP handlers, and the SSE clients. Each server
  owns its store and its clients.
- `tui.go` and `internal/tui/`: the terminal flags and the terminal board.
- `internal/board/`: the shared task store, file writes, config, watching, and
  view helpers. See [Board store](board-store.md).
- `index.html`: the web board, in one file.
- `docs/`: this documentation. It is also the site source for
  [the docs site](docs-site.md).
- `docs-site/`: the Astro project that builds the docs site. It only renders
  `docs/`.
- `test-work-items/`: the QA fixtures. Copy them before you run MK.
- `.github/workflows/`: `ci.yml` for the Go checks, `release.yml` for releases,
  and `docs.yml` for the docs site.
