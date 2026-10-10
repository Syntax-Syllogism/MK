---
title: Using MK with Git
description: Decide what to commit, ignore MK's cache, and resolve a merge conflict in a task file.
---

# Using MK with Git

MK keeps its state in files, so Git can track it. The board is a view, and
the task files are the record.

## What to commit

Commit these files in your task folder:

- The task files, such as `docs/*.md`.
- `.kanban.yml`, which sets the board columns. See [Columns](columns.md).

Ignore MK's cache. MK writes `.kanban/board.json` in the directory it was started
from. It is rebuilt on every scan and is not the source of any task. Add this to
`.gitignore`:

```text
.kanban/
```

When you drag a card, MK changes the `status:` and `order:` lines in the task's
file. A `git diff` shows those as ordinary text changes, so a move is easy to
review and easy to revert.

## Resolve a conflict in a task file

When two branches change the same task, a merge or rebase can leave Git conflict
markers in the file. MK shows that card in the **CONFLICT** column and does not
allow it to be dragged.

To resolve it:

1. Open the file in your editor. Git marks the conflicting sections with start,
   separator, and end lines.
2. Choose the text you want, and delete the markers.
3. Check the frontmatter. Make sure `status:` holds the column you intend. A
   status that matches no column shows the card in OTHER.
4. Save the file. The card moves to its column without any action on the board.
5. Stage the file and finish the merge or rebase.

The terminal board can open the file for you. Select the card and press `e`.
See [Card actions](terminal-ui.md#card-actions).

For what CONFLICT, OTHER, and parse-error cards mean, see
[Special cards](web-ui.md#special-cards).

## Revert a move

To revert a card move, or any other edit, use Git as you would for any text
file. The terminal board's `u` key also undoes its own moves and tag changes, but
only while the files are unchanged since the action. See
[Card actions](terminal-ui.md#card-actions).
