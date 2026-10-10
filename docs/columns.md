---
title: Columns
description: Set board columns in .kanban.yml, rename or reorder them, and update task statuses to match.
---

# Columns

The board's columns come from `.kanban.yml`, in your task folder. Edit that file
by hand to add, remove, rename, or reorder columns. MK does not change the file
after it creates it.

```yaml
# MK board columns. Order here is the order on the board.
# A card whose `status` matches none of these lands in OTHER.
columns:
  - TODO
  - IN PROGRESS
  - DONE
```

## First run

When MK opens a task folder that has no `.kanban.yml`, it creates one. The
columns come from the `status:` values in your existing tasks, in the order MK
finds them. If there are no tasks yet, MK uses `TODO`, `IN PROGRESS`, and `DONE`.

After that, the file is yours. MK does not rewrite it.

## Change columns

Save the file and the board updates within about a second. You don't need to
restart MK. The terminal board reloads the same way.

- **Add** a column by adding a line.
- **Remove** a column by deleting its line. Cards with that status move to OTHER.
- **Reorder** columns by moving lines.

Column names are matched exactly, including case and spaces.

Some names are not allowed as columns:

- `CONFLICT` and `OTHER` are reserved for MK's own sections. If you list one, MK
  logs a warning and ignores it.
- Duplicate and empty names are ignored.

If the file cannot be read as YAML, MK keeps the columns it last loaded and logs
the error. The terminal board lists it under its warnings (`!`).

## Rename a column

Renaming a column in `.kanban.yml` does not change the tasks. A task that still
uses the old name moves to OTHER. You have two choices:

- Restore the old column name, or
- Update each task's `status:` to the new name.

To find the tasks that need updating, search your task folder. For example, with
a column that was renamed from `REVIEW` to `IN REVIEW`:

```bash
grep -l '^status: REVIEW$' docs/*.md
```

Then edit those files, or drag the cards to the new column on the web board.

To update many files at once, use your editor's search and replace on the
`status:` lines.
