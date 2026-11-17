---
title: Dependencies
description: Link tasks with depends_on, follow dependency badges, and read the warnings for missing targets and cycles.
---

# Dependencies

A task can name the tasks it depends on. Add a `depends_on` list to its
frontmatter:

```markdown
---
status: TODO
depends_on: [design-task, api-task]
---
```

Each entry is a task slug: the file name without `.md`. In the example, the
tasks are `design-task.md` and `api-task.md`.

## What MK shows

- **Web board:** each dependency is a badge on the card. Select a badge to open
  the target card.
- **Terminal board:** the table and cards show a dependency count. In the
  preview, press `Tab` to select a dependency, then `Enter` to follow it. See
  [Task preview and dependencies](terminal-ui.md#task-preview-and-dependencies).

## Warnings

MK flags two problems. Both are warnings only. A flagged card still appears, and
you can still move it.

- **Missing target.** The slug does not match a task file. The badge is styled
  as a warning. In the terminal, the target shows as `unknown card: <slug>`.
- **Cycle.** Two or more tasks depend on each other, directly or through a
  chain. Every task in the cycle is flagged.

To fix a missing target, check the slug for typos, and check that the file
exists in the task folder. To fix a cycle, remove one of the `depends_on`
entries.

## Notes

- Dependencies do not affect columns or moves. A task can move to `DONE` while
  its dependencies are still open.
- MK does not write `depends_on` for you. You maintain it by hand.

## Archived dependencies

The terminal board treats Markdown filenames in the configured archive folder
as known dependencies. They appear dimmed with `archived`; following one shows
`archived: <slug>`. Archived files do not participate in dependency cycles.
Only the archive folder's top level is checked, without parsing its files.
Changes made solely inside the archive folder appear after `r` or another
board rescan.

The web board still marks archived dependencies as missing; matching its
behavior to the terminal board is a follow-up.
