---
title: HTTP API
description: The local endpoints behind the MK web board, with request and response shapes, error codes, and the live update stream.
---

# HTTP API

The web board is a single page that talks to `mk` over a small JSON API. You can
use the same endpoints from scripts. They serve the port that `mk` prints when
it starts, which is `8080` by default.

These endpoints have no authentication. See
[Network access](cli.md#network-access) before you expose the port.

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/board` | Read the columns and tasks. |
| `POST` | `/api/update` | Change one task's status, order, tags, or body. |
| `GET` | `/api/events` | Server-Sent Events stream of changes. |
| `GET` | `/` | The web board page. |

## Read the board

`GET /api/board` returns the current state. The response is built from a
snapshot of the task folder.

```json
{
  "columns": ["TODO", "IN PROGRESS", "DONE"],
  "tasks": [
    {
      "file": "write-install-notes.md",
      "status": "TODO",
      "project": "MK",
      "epic": "SETUP",
      "order": 1,
      "tags": ["docs"],
      "dependsOn": ["design-task"],
      "title": "write-install-notes",
      "heading": "Write install notes",
      "content": "# Write install notes\n\nAdd the install steps...",
      "parseError": ""
    }
  ],
  "docsDir": "/home/you/project/docs",
  "archived": {"old-task.md": true}
}
```

| Field | Meaning |
| --- | --- |
| `columns` | Column names from `.kanban.yml`, in display order. |
| `tasks[].file` | The task's file name, including `.md`. |
| `tasks[].status`, `project`, `epic` | The frontmatter values. Empty strings when not set. |
| `tasks[].tags` | The frontmatter tags. `null` when not set. |
| `tasks[].order` | Present only when the task has an `order` value. |
| `tasks[].dependsOn` | Present only when the task has dependencies. |
| `tasks[].title` | The card title, which is the file name without `.md`. |
| `tasks[].heading` | The first Markdown H1 outside fenced code blocks. Present only when the body has one. |
| `tasks[].content` | The Markdown body, without the frontmatter. For a `CONFLICT` task, the whole raw file. |
| `tasks[].parseError` | Present only when the frontmatter could not be parsed. |
| `docsDir` | The absolute path of the task folder. |
| `archived` | An object whose keys are the `.md` file names directly in the archive folder (`archive_dir`, default `archive`), each set to `true`. Empty when nothing is archived. The web board does not use it yet. |

A file that contains Git conflict markers has `status` set to `CONFLICT`. See
[Special cards](web-ui.md#special-cards).

## Update a task

`POST /api/update` changes one task file. The body is a JSON object.

```bash
curl -X POST http://localhost:8080/api/update \
  -H 'Content-Type: application/json' \
  -d '{"file": "write-install-notes.md", "status": "DONE"}'
```

| Field | Required | Meaning |
| --- | --- | --- |
| `file` | Yes | The task's file name, such as `write-install-notes.md`. Paths are not accepted. |
| `status` | No | New status. Omit it, or send an empty string, to leave it unchanged. |
| `order` | No | New order value, as a number. Omit it to leave it unchanged. |
| `tags` | No | New tag list. Omit it to leave the tags unchanged. An empty array removes all tags. |
| `content` | No | New Markdown body. Omit it to leave the body unchanged. An empty string empties the body. |

Send at least one of `status`, `order`, `tags`, or `content`. The request
updates only the fields you send, and it keeps the rest of the frontmatter.

Don't send `CONFLICT` or `OTHER` as a status. Those are reserved.

On success, the response is `200 OK` with no body. Errors are plain text.

| Status | When |
| --- | --- |
| `200 OK` | The file was updated. |
| `400 Bad Request` | The body is not valid JSON, `file` is missing or invalid, the status is `CONFLICT` or `OTHER`, or no field was sent. |
| `405 Method Not Allowed` | The request was not a `POST`. |
| `409 Conflict` | The write was refused by the empty-file guard. |
| `500 Internal Server Error` | Any other failure, such as a file with invalid frontmatter. |

`POST /api/update` writes the file and reads it back. It does not wait for the
file watcher, so the next `GET /api/board` already shows the change.

## Live updates

`GET /api/events` opens a Server-Sent Events stream. When a task file or
`.kanban.yml` changes, the server sends:

```text
data: update

```

Clients respond by fetching `/api/board` again. Several changes in quick
succession can produce one message, so treat the message as "something changed"
and not as a list of changes.

## Fields the API does not accept

The store has more write options than the API exposes. The API does not accept
the TUI-only fields for removing keys or for undo. Those are described in
[Board store](board-store.md).
