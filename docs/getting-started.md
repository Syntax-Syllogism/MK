---
title: Getting started
description: Install MK on macOS, Linux, or Windows, start the web or terminal board, and create your first task file.
---

# Getting started

This page takes you from nothing installed to a working board with one task.

## Install

MK is a single binary. It needs no runtime, database, or container.

### Download a release

Each release on the [GitHub releases page](https://github.com/Syntax-Syllogism/MK/releases)
includes archives for Linux, macOS, and Windows, on amd64 and arm64. Archive
names look like `mk_0.3.4_Linux_x86_64.tar.gz`. Windows archives are `.zip`
files. Each archive contains the `mk` binary and the README.

Unpack the archive and move `mk` (or `mk.exe`) onto your `PATH`:

```bash
# macOS / Linux
sudo mv mk /usr/local/bin/mk
```

On Windows, move `mk.exe` into a folder that is already on your `PATH`, or add a
folder under **System Properties → Environment Variables**.

### Build from source

You need Go 1.26 or newer, the version in `go.mod`.

```bash
git clone https://github.com/Syntax-Syllogism/MK.git
cd MK
go build -o mk .
```

On Windows, use `go build -o mk.exe .`. Then move the binary onto your `PATH`
as above.

### Check the install

```bash
mk -version
```

## Start the board

Change into a project directory and run:

```bash
mk
```

MK uses the `docs/` folder in that directory as the task folder. It starts a
local web server and prints the address, usually `http://localhost:8080`, or the
next free port. Open it in your browser. Leave MK running: the board updates as
you edit files.

To use a different folder, pass `-dir`:

```bash
mk -dir my-tasks
```

If the folder does not exist, MK creates it. If it has no `.kanban.yml` file,
MK creates one with the columns it finds in your task statuses, or with `TODO`,
`IN PROGRESS`, and `DONE` if there are no tasks yet. See [Columns](columns.md).

To use the terminal board instead, run:

```bash
mk tui
```

The terminal board uses the same folder and files. Press `?` for the key list.
See the [terminal board guide](terminal-ui.md).

## Write your first task

A task is a Markdown file in the task folder. The file name, without `.md`, is
the card title. Create `docs/write-install-notes.md`:

```markdown
---
status: TODO
project: MK
epic: SETUP
tags: [docs]
---

# Write install notes

Add the install steps for our team to the project README.
```

Save the file. The card appears in the `TODO` column within a second. Your
column names come from `.kanban.yml`, so use a status that matches one of them.

Now try the board:

- Drag the card to `IN PROGRESS` in the web board. MK writes the new `status:`
  to the file for you.
- Or open the file and change `status:` by hand. The card moves when you save.

## Next steps

- [Web board](web-ui.md) covers editing, filters, and the table view.
- [Task files](task-files.md) lists every frontmatter field.
- [Using MK with Git](git-workflow.md) explains what to commit.
