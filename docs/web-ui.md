---
title: Web board
description: Use the browser board to move and reorder cards, edit tasks, filter, switch to the table view, and read special cards.
---

# Web board

The web board is the browser front end, started with `mk`. It shows the task
files in your task folder as cards in columns. Every change you make on the
board is written to a task file, and every change to a file appears on the board
without a refresh.

For the terminal front end, see the [terminal board guide](terminal-ui.md).

## Columns

Each column matches one line in `.kanban.yml`, in the order listed there. See
[Columns](columns.md) to change them.

Two more sections can appear:

- **CONFLICT** appears first, when a task file contains Git conflict markers.
- **OTHER** appears last, when a task has a `status` that matches no column, has
  no `status`, or has frontmatter MK cannot parse.

Your configured columns are always visible, even when empty. Filters hide cards,
not columns.

## Move and reorder cards

Drag a card to:

- **Another column** to change its `status:`.
- **A new position in its column** to change its `order:`. Cards are sorted by
  `order`, and cards without one sort to the bottom by filename.
- **Both at once**, by dropping it in another column at a specific height. MK
  writes both fields in one save.

You don't need to set `order:` yourself. Leave it out of new task files.

You cannot drop a card into CONFLICT or OTHER. Fix the file instead, as described
below.

## Edit a card

Click a card to open it. The modal has two tabs:

- **PREVIEW** shows the rendered Markdown. It opens first.
- **EDIT** shows the raw Markdown. Edits save about a second after you stop
  typing. The header shows `typing...`, `saving...`, and `saved` as this happens.

Manage tags in the footer of the modal. Type a tag and press Enter to add it.
Click the `×` on a tag to remove it. Tag changes save at once.

`status` and `order` cannot be edited in the modal. Drag the card to change them.

Close the modal with the `close()` button, the Escape key, or a click outside the
card. The `copy path to file` button copies the card's file path.

## Dependencies

Cards with `depends_on` show a badge for each dependency. Select a badge to open
its target. For the rules, see [Dependencies](dependencies.md).

## Filters

The filter bar above the board lists projects, epics, and tags. It starts
expanded. Click the `filters` header to collapse or expand it. MK remembers the
choice in your browser.

When a filter is active, the header shows a count such as `filters (2)`. Click
its `×` to clear every filter at once.

Filters combine as follows: selections within one group match any of them, and
selections across groups must all match.

## Table view

Click `switch_view()` in the header to replace the board with a flat table of the
visible tasks. The table shows file, project, epic, status, and tags. Rows are
sorted by status, then by `order`. Column headers do not sort.

Click `switch_theme()` to switch between light and dark.

## Special cards

Some cards are not in a configured column. The card says why.

### CONFLICT

A task is a CONFLICT card when its file contains unresolved Git conflict
markers: the start, separator, and end lines that Git writes into a file after a
failed merge or rebase. CONFLICT always renders first.

Conflict cards cannot be dragged. To fix one:

1. Open the file in your editor.
2. Resolve the conflict the way you normally would, and remove the markers.
3. Check that `status:` holds the value you want.
4. Save.

The card moves to its column once the markers are gone. See
[Using MK with Git](git-workflow.md) for a longer walkthrough.

### OTHER

A task is an OTHER card for one of two reasons. The card shows which one.

- **Wrong or missing status.** The card looks normal and sits in OTHER. Change
  `status:` to a column name in `.kanban.yml`, or add the status as a new column.
  The card moves on the next save.
- **Unparseable frontmatter.** The card's epic line reads `!!! PARSE ERROR`, and
  the card opens with a red banner that explains the problem. An example is
  `order: soon`, which should be a number. The body is kept on disk, but the
  editor and tag controls are read-only until the frontmatter is fixed, so MK
  cannot overwrite it with a bad save. Fix the YAML block at the top of the file
  in your editor, then reopen the card.

OTHER cards cannot be dropped into either. Fix the file, as above.

For the frontmatter fields, see [Task files](task-files.md).
