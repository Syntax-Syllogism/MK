---
title: Task files
description: The frontmatter fields MK reads, how file names become card titles, and the rules for people and agents who write task files.
---

# Task files

A task is a Markdown file in the task folder. MK reads the top-level `.md` files
in that folder. Subfolders are ignored, and so are files whose names do not end
in lowercase `.md`.

A task starts with a YAML frontmatter block, followed by a Markdown body:

```markdown
---
status: TODO
project: MK
epic: SETUP
tags: [backend, go]
depends_on: [design-task, api-task]
---

# Task title

Task description goes here...
```

## Fields

| Field | Required | Type | Meaning |
| --- | --- | --- | --- |
| `status` | For a card in a column | Text | Must match a column in `.kanban.yml` exactly, with the same case. Otherwise the card goes to OTHER. |
| `project` | No | Text | Shown on the card. Used as a filter. |
| `epic` | No | Text | Shown on the card. Used as a filter. A task with no epic is filtered as `NO EPIC`. |
| `tags` | No | List | Shown on the card. Used as filters. |
| `depends_on` | No | List of slugs | Other tasks this one depends on. A slug is a file name without `.md`. See [Dependencies](dependencies.md). |
| `order` | No | Number | Position within the column. MK writes it when you drag a card. Don't write it by hand. |

Other keys are kept as they are. MK does not use them, but it preserves them when
it writes the file.

## Card titles

MK uses the file name, without `.md`, as the card title. A `title:` key in the
frontmatter does not change the title. Name the file to match the title you want.

## Special cards

A card may appear outside your configured columns. There are three cases:

- **CONFLICT**: the file contains Git conflict markers. Resolve the conflict in
  the file.
- **OTHER**: the status is missing or matches no column. Set a valid status.
- **Parse error**: the frontmatter is not valid YAML, or a field has the wrong
  type, such as `order: soon`. Fix the YAML by hand.

The web board and the terminal board both show these cards. For how to recognize
each one and fix it, see [Special cards](web-ui.md#special-cards).

## Rules for people and agents

If you write task files, follow these rules:

- Create a task with a `status`, and leave out `order:`.
- Don't write or renumber `order:`. MK manages it.
- To change a task's state, change its `status:`.
- Make `status` exactly match a column name in `.kanban.yml`.
- Keep the frontmatter block at the top of the file, delimited by `---` lines.
- Keep tags short and lowercase.
- Don't add `CONFLICT` or `OTHER` as a status. Those are MK's own labels.
- Don't write Git conflict markers into a file on purpose. Resolve them first.
