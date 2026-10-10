---
title: Command line
description: Reference for the mk command, the mk tui subcommand, their flags, default ports, and where MK stores its cache and preferences.
---

# Command line

MK has one binary with two modes:

- `mk [flags]` starts the web board.
- `mk tui [flags]` opens the terminal board.

## mk

| Flag | Default | Meaning |
| --- | --- | --- |
| `-dir <path>` | `docs` | The folder that holds the task files. Relative paths are resolved from the current directory. |
| `-port <n>` | `8080` | The first port to try. If it is in use, MK tries the next ports, up to 100 in all. |
| `-version` | | Print the version, commit, and build date, then exit. |

Starting `mk` prints the address it is serving:

```text
MK // Kanban starting on http://localhost:8080
Watching directory: docs
```

If `-dir` does not exist, MK creates it. If `.kanban.yml` is missing, MK creates
one. See [Columns](columns.md).

```bash
mk -dir my-tasks -port 9000
```

## mk tui

| Flag | Default | Meaning |
| --- | --- | --- |
| `-dir <path>` | `docs` | The folder that holds the task files. |
| `-no-mouse` | off | Turn off mouse capture, so the terminal can select text natively. |
| `-version` | | Print the build version and exit. It does not open the task folder. |

`-port` is a web option. `mk tui` rejects it.

```bash
mk tui -dir my-tasks -no-mouse
```

For what the terminal board does, see the [terminal board guide](terminal-ui.md).

## Network access

The web server listens on every network interface, not only `localhost`. It has
no login. Anyone who can reach its port can read and change your task files.
Run `mk` on a trusted network, or block the port with a firewall. Don't expose
it to the internet.

## Files MK writes

- **`.kanban/board.json`** is a cache of the board. It is written relative to the
  directory you started MK from, not to the task folder. It is rebuilt on each
  scan and is safe to delete.
- **`.kanban.yml`** is created in the task folder if it is missing. MK does not
  change it after that.
- **Task files** change only when you move a card, edit tags, or save a file in
  the web or terminal board.
- **Terminal preferences** are stored outside the task folder, per task folder.
  On Linux they are in `$XDG_STATE_HOME/mk/`, which defaults to
  `~/.local/state/mk/`. On macOS and Windows they are in the user config folder,
  under `mk/state/`.

## Exit status

If MK cannot start, for example because it cannot create the task folder or no
port is free, it prints the error to standard error and exits with a non-zero
status. Invalid flags also stop MK before it starts.

## Development runs

When you run MK from source, `go run .` takes the same flags as `mk`, and
`go run . tui` takes the same flags as `mk tui`. See [Development](development.md).
