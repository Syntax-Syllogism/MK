---
title: MK documentation
description: Install MK, run its web and terminal boards, and find the guides and reference for task files, columns, and the HTTP API.
---

**MK** is a Kanban board for the Markdown files in your project. Each task is a
file, the board is one view of those files, and nothing else stores state. You
can read, edit, diff, and commit your tasks like any other text.

MK has two front ends that read the same files and can run at the same time:

- `mk` serves a board in your browser.
- `mk tui` opens a board in your terminal.

## Start here

- [Getting started](getting-started.md): install MK, run it, and write your first task.

## Guides

- [Web board](web-ui.md): columns, moving and editing cards, filters, the table view, and special cards.
- [Terminal board](terminal-ui.md): keys, card actions, search, previews, undo, and preferences.
- [Columns](columns.md): configure board columns in `.kanban.yml`, and rename or migrate statuses.
- [Dependencies](dependencies.md): link tasks with `depends_on`, and read missing-target and cycle warnings.
- [Using MK with Git](git-workflow.md): what to commit, what to ignore, and how to resolve conflicted task files.

## Reference

- [Command line](cli.md): flags, subcommands, ports, and the cache location.
- [Task files](task-files.md): frontmatter fields, filename rules, and rules for people and agents who write tasks.
- [HTTP API](http-api.md): the endpoints behind the web board, with request and response shapes.

## Contributing

- [Development](development.md): build, test, and smoke-test MK, and the repository layout.
- [Board store](board-store.md): the `internal/board` package that both front ends share.
- [Documentation site](docs-site.md): preview and build this site, and the writing rules for these pages.

<!-- Screenshot or GIF of the web board and the terminal board goes here. -->
