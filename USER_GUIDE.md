# MK User Guide

This is a guide to actually using the MK board day-to-day. For installation and build
instructions, see [README.md](README.md). For the task file schema and agent-authoring
rules, see [AGENTS.md](AGENTS.md).

## Starting the board

```bash
mk
```

Run this from the directory containing your `docs/` folder (or point `-dir` at a
different one). MK opens a local web server — visit the URL it prints (default
`http://localhost:8080`, or the next free port). Leave it running; the browser tab updates
live as you edit task files, no refresh needed.

## The board

Each column on the board corresponds to one line in `docs/.kanban.yml`, in the order
listed there. Three special sections can also appear:

- **CONFLICT** — appears only when at least one task file contains Git conflict markers
  (`<<<<<<<`), and always renders first. You cannot drag a card into or out of it by
  dropping — resolve the conflict in the file itself (see [Resolving a
  CONFLICT](#resolving-a-conflict) below), and it disappears once the markers are gone.
- **OTHER** — appears only when at least one task has a `status` that doesn't match any
  configured column, or no `status` at all, or frontmatter MK couldn't parse. It renders
  last. You cannot drop a card into it either — fix the task's `status` (or its YAML) and
  it moves itself to the right column on the next save.
- Your **configured columns** — always visible, even when empty, so you always have
  somewhere to drop the last card out of a column.

Filtering by project, epic, or tag never hides or reveals a column — only cards. A column
that's empty because of an active filter is still a valid drop target.

## Moving and reordering cards

Drag a card to:

- **Move it between columns** — drop it in a different column's list; this writes a new
  `status:` to the file.
- **Reorder it within a column** — drop it above or below another card; this writes a new
  `order:` float to the file. Cards sort by `order` ascending, then by filename for cards
  that don't have one yet (so untouched cards keep a stable, predictable position at the
  bottom of the pile).
- **Do both at once** — drag across columns to a specific vertical position; `status` and
  `order` are written together in a single request, so the board file only changes once per
  drag.

You don't need to set `order:` by hand. Leave it out of new task files — the card will sort
to the bottom of its column, which just means "not yet triaged."

## Editing a card

Click any card to open it. The modal has two tabs:

- **PREVIEW** — rendered Markdown (opens by default).
- **EDIT** — the raw Markdown source. Edits autosave about a second after you stop typing;
  watch the status indicator in the modal header (`typing...` → `saving...` → `saved`).

Tags are managed from the footer of the modal — type into the box and press Enter to add
one, click the `×` on a tag to remove it. Tag changes save immediately.

`status` and `order` are **not** editable from the modal — they're set exclusively by
dragging the card on the board.

## Filters

The filter bar (projects / epics / tags) sits above the board and starts expanded. Click
the `filters` header to collapse or expand it — MK remembers your choice across restarts.
When any filter is active, the header shows a count (`filters (2)`) and an `×` that clears
every active filter in one click without needing to expand the bar first.

## Table view

Click `switch_view()` in the header to swap the board for a flat, sortable-by-eye table of
every visible task (file, project, epic, status, tags) — useful for scanning a large
backlog. `switch_theme()` toggles between light and dark.

## Configuring columns

Board columns live in `docs/.kanban.yml`, not in code. On first run against a `docs/`
folder that has no `.kanban.yml` yet, MK generates one automatically from whatever
`status:` values it finds in your existing task files (in the order it encounters them),
or falls back to `TODO` / `IN PROGRESS` / `DONE` if there are no tasks yet. After that,
**MK never rewrites this file on its own** — it's yours to edit like any other file in the
repo:

```yaml
# MK board columns. Order here is the order on the board.
# A card whose `status` matches none of these lands in OTHER.
columns:
  - TODO
  - IN PROGRESS
  - DONE
```

Add, remove, rename, or reorder lines and save — the board reflows within about a second,
no restart needed. Two names are reserved and cannot be used as a column: `CONFLICT` and
`OTHER`. If you accidentally list one, MK logs a warning and drops it from the column list
rather than refusing to start.

Renaming a column in this file does **not** rewrite `status:` in your task files — any task
using the old name will show up in OTHER until you either restore the old column name or
update the task's `status:` to match.

## Resolving a CONFLICT

A task shows as CONFLICT when its file contains unresolved Git merge markers
(`<<<<<<<`/`=======`/`>>>>>>>`). Open the file in your editor, resolve the conflict as you
normally would, save, and the card moves itself into whatever column its (now clean)
`status:` points to — no action needed on the board itself. Conflicted cards can't be
dragged to another column on the board; the conflict has to be resolved in the file first.

## When a card lands in OTHER

Two different problems both route a card to OTHER, and the card tells you which:

- **Wrong or missing status** — the card looks normal, just sitting in OTHER. Fix by
  changing `status:` in the file to match one of the columns in `docs/.kanban.yml` (or add
  the status you want as a new column there).
- **Unparseable frontmatter** — the card's epic line reads `!!! PARSE ERROR`, and opening it
  shows a red banner explaining what YAML couldn't be parsed (for example, `order: soon`
  instead of a number). The card's body is preserved on disk, but the editor and tag
  controls are read-only in the modal until the frontmatter is fixed — MK won't let you
  make changes here that might not save cleanly. Fix the frontmatter block at the top of
  the file directly, then reopen the card.
