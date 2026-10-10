# MK // Markdown Kanban

**MK** is a localhost Kanban board that uses your filesystem as its database.
You get a board view of your tasks, and the tasks stay as plain Markdown files
that work with Git and any editor.

- **Plain files**: each task is a `.md` file in a folder, such as `docs/`.
- **Two front ends**: `mk` opens a browser board, and `mk tui` opens a terminal
  board. Both read the same files and can run at the same time.
- **Live sync**: changes to files show up on the board without a refresh.
- **Drag to move**: moving or reordering a card updates the task file.
- **Configurable columns**: columns come from a committed `.kanban.yml`.
- **Git-aware**: Git conflict markers show as a CONFLICT card instead of
  breaking the board.

## Install

MK is a single Go binary with no runtime dependencies. Requires Go 1.26 or
newer.

```bash
go build -o mk .
```

Move `mk` (or `mk.exe` on Windows) somewhere on your `PATH`.

## Quick start

From the project that holds your tasks:

```bash
mk               # browser board; uses docs/ by default
mk tui           # terminal board
mk -dir my-tasks # use another task folder
```

The first run creates `.kanban.yml` if it is missing. Each task file starts
with a YAML frontmatter block:

```markdown
---
status: TODO
project: MK
tags: [backend]
---

# Write install notes

Task description goes here.
```

## Documentation

Full guides and the command reference are on the docs site:
**<https://syntax-syllogism.com/mk/docs/>**

- [Getting started](https://syntax-syllogism.com/mk/docs/getting-started/)
- [CLI reference](https://syntax-syllogism.com/mk/docs/cli/)
- [Task files](https://syntax-syllogism.com/mk/docs/task-files/)

To work on MK itself, see [Development and smoke testing](docs/development.md).
The documentation source is in [`docs/`](docs/README.md).
