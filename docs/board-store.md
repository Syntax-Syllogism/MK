---
title: Board store
description: How the internal/board package opens, reads, writes, and watches task files, and how the web server and terminal board use it.
---

# Board store

`internal/board` owns the task and config state, independent of HTTP. `main.go`
embeds the frontend and dispatches to `runServe` in `serve.go` by default, or to
`runTUI` in `tui.go` for `mk tui`. `internal/tui` consumes the same snapshots and
watcher, and uses the task-writing API for card actions.

Both front ends must keep the same task-file rules. Writes go through
`Store.Apply` or its batch methods, never around them. The one exception is the
archive move, which renames an unchanged file; see [Archive moves](#archive-moves).

## Opening and reading

`board.Open(dir, board.Options{CachePath: ..., Logf: ...})` resolves an absolute
task directory, creates it if needed, scans tasks, bootstraps an absent
`.kanban.yml`, and loads columns. Directory resolution/creation errors are
returned. Initial scan/bootstrap/config errors are logged and opening continues.
Malformed YAML on initial config load can therefore leave columns empty; on a
later failed reload the previous columns remain.

An empty `CachePath` disables cache writes. A relative path is resolved against
the process working directory, not the task directory. The web server explicitly
uses `.kanban/board.json`. `Logf` receives warnings; nil falls back to `log.Printf`.
The cache is write-only derived output, never the source of task state.

`Store.Dir()` returns the absolute directory. `Store.Snapshot()` returns
independent copies of columns, tasks, tags, dependencies, order pointers, and
archived filenames. Consumers can modify them without changing the store. `Scan`, `ReloadConfig`,
and `Apply` serialize their I/O; a separate read/write lock protects state.

`Scan()` reads top-level `.md` files in filename order. Unreadable files are
logged and skipped; malformed typed frontmatter produces a task with `ParseError`
and its real body, so the card remains visible. `ParseTask(path)` can also be
used directly. Conflict detection takes precedence over YAML parsing. After
replacing the in-memory tasks, a scan writes the configured cache, if enabled.
Cache failures are returned even though the in-memory scan has completed.

`Scan()` also lists top-level `.md` names in the archive folder into
`Snapshot.Archived`, without parsing them. `ArchiveDir()` returns the configured
`archive_dir`, or `archive` when unset. A missing archive folder is empty. If
the archive path has a symlink or a non-directory component, `Scan` returns an
error but still publishes and caches the top-level tasks. Archive read errors
behave the same way; archived names are unavailable for that scan. `Open` logs
the error and continues with the scanned tasks.

Config bootstrapping happens only at open. `ReloadConfig()` never rewrites config;
missing or empty/unusable columns retain previous columns, or use
`TODO` / `IN PROGRESS` / `DONE` if none exist. Duplicate, empty, and reserved
(`CONFLICT`, `OTHER`) column names are ignored. Invalid YAML returns an error.

## Writes

`Store.Apply(board.UpdateRequest{...})` reads an existing file, changes its
frontmatter/body, writes it, and re-parses that file into the store immediately.
It does not update the cache or invoke a watcher callback; a later watcher scan
refreshes the cache and notifies consumers. `TaskPath(file)` rejects empty names
and paths whose basename differs from the input; callers use filenames.

| Field | Meaning |
| --- | --- |
| `File` | Existing task filename |
| `Content *string` | Nil preserves the body; a pointer to an empty string explicitly empties it |
| `Status string` | Empty means no change; CONFLICT and OTHER assignments are rejected |
| `Order *float64` | Nil preserves order; a pointer sets it |
| `Tags []string` | Nil preserves tags; an empty non-nil slice writes `tags: []` |
| `Unset []string` | Deletes only `status`, `order`, or `tags`; setting and unsetting the same key is rejected |
| `ExpectedHash *[32]byte` | Rejects the write with `ErrChanged` unless the current bytes match (TUI-only; not exposed by HTTP) |
| `Restore []byte` | TUI-only exact-byte undo; see the paragraph below (not exposed by HTTP) |

An unset-only request on a file without frontmatter preserves its bytes and
does not add a YAML block. Frontmatter writes use a generic map, preserving
unrelated keys while reserializing YAML; comments and original formatting are
not preserved. A syntactically invalid frontmatter map is rejected. Typed
parse errors are not a blanket write guard in `Apply`; the web UI makes these
cards read-only. See [task-file rules](task-files.md) for task semantics.

Callers can use `errors.Is` with `ErrMissingFile`, `ErrInvalidFile`,
`ErrInvalidUnset`, `ErrConflictStatus`, `ErrOtherStatus`, `ErrNothingToUpdate`,
`ErrInvalidFrontmatter`, `ErrChanged`, `ErrInvalidRestore`, and the defensive
`ErrEmptyClobber` sentinel. Read, write, and re-parse I/O errors are wrapped with
operation/path context.

`ApplyBatch` serializes all updates for one TUI action and rolls back completed
writes if a later request fails. `ApplyBatchWithHashes` also returns hashes of
the exact bytes written in request order, so the TUI can record undo without a
post-write disk read. `ExpectedHash` verifies the current bytes under
the store I/O lock. TUI-only `Restore` restores exact original bytes for undo;
it requires `ExpectedHash` and rejects changes to the body or frontmatter keys
other than status/order/tags. These fields are not exposed by HTTP.
`FrontmatterKeys(path)` returns top-level YAML nodes using `ParseTask`'s delimiter
rules, retaining the distinction between missing keys and explicit empty values.

## Archive moves

`Store.Archive(file, expected)` and `Store.Unarchive(file, expected)` are the
non-`Apply` write path. They rename an unchanged `.md` file between the task
folder and the archive folder, and the TUI uses them for `A` and its undo. They
do not change file content, so `Restore` and `ExpectedHash` are not involved.

- Both hold the store I/O lock. `expected` is the SHA-256 of the file's current
  bytes. A mismatch returns `ErrChanged`; a non-regular source returns
  `ErrInvalidFile`.
- `Archive` reparses the file on disk and refuses CONFLICT and parse-error cards,
  even if the caller's snapshot is stale.
- The archive folder is created on first use. A symlink or non-directory in the
  archive path is refused.
- A destination that already exists returns `name taken`. The check is a stat
  followed by `os.Rename`, so an external writer can still race it.
- Neither call updates the in-memory tasks or the cache. Callers run `Scan`
  afterwards.

## Watching

`Store.Watch(ctx, onChange)` attaches fsnotify to the task directory and blocks
until cancellation or watcher termination. Start it after `Open`. It watches
Write/Create/Remove/Rename events for `.md` and `.kanban.yml`, with a 50 ms
trailing debounce. The archive folder is not watched, so changes made only inside
it appear after a rescan (`r` in the TUI). Each flush reloads config first if a
`.kanban.yml` event arrived, then scans once if any `.md` or config event arrived,
then invokes `onChange` synchronously, even if an operation logged an error.
Keep the callback short. A nil callback is allowed; cancellation returns nil
and closes the watcher. Watcher creation/attachment errors are returned;
runtime watcher errors are logged.

## View helpers

These helpers operate on snapshots/tasks without changing files. The web
frontend still implements its own ordering/filter/dependency presentation in
JavaScript; the Go helpers are available for additional consumers.

- `ByOrder` sorts ascending with nil order last, then filenames using byte
  comparison (which can differ from JavaScript `localeCompare`). `ComputeOrder`
  averages two neighbours, steps one past a sole neighbour, or returns 1.
- `DropOrder` expects the full target column, already sorted by `ByOrder` and
  excluding the moving card. Pass the visible card below the drop as
  `beforeFile`, or the last visible card for an end drop. Hidden neighbours
  still participate in the calculation. It returns `(order float64, updates
  []UpdateRequest)`. An end drop following an unordered card assigns orders to
  the unordered tail without modifying the input. Apply the returned updates
  through `Store.Apply` along with the moving card's order; a numeric order alone
  cannot place a card after cards with unset orders.
- `Section` classifies CONFLICT, a configured column, or OTHER.
  `Sections(snapshot)` uses all unfiltered tasks: optional non-droppable
  CONFLICT first, all configured droppable columns, optional non-droppable
  OTHER last. `SortForTable` returns independent task copies in this section
  order, then `ByOrder`; this differs from the web table's status text sorting.
- `Filter` combines true selections with OR within project/epic/tag groups
  and AND across groups. False map values are unselected. `Active` counts
  selected chips plus one for a nonblank query; `Visible` tests a task.
  `Query` ANDs whitespace-separated terms using fuzzy `Title` or `Heading`
  matches, or substrings in metadata/body; the web leaves it empty. `Score` sums
  each term's best `Title` or `Heading` match, and `MatchesQuery` accepts a
  cached lowercase body for the terminal consumer. `NoEpic` (`NO EPIC`) matches
  empty epic. `Facets` returns sorted unique values, omitting empty projects.
- `AnalyzeDeps(tasks, archived...)` returns filename-keyed `Known`, `Cycles`, and
  `Archived` maps. Tarjan's strongly connected components mark only cycle members
  (including self-loops), ignoring missing edges. `DepInfo.Missing(slug)` checks
  `slug + ".md"`. `DepInfo.IsArchived(slug)` is true for an unknown slug whose
  `.md` is in the archive folder, and `Missing` is false for those slugs.

## HTTP and SSE

The web adapter is in `serve.go`. The endpoints are documented in the
[HTTP API](http-api.md) reference.

Each `server` owns a store and its SSE clients; no board/config/client state is
package-global. `/api/board` serves `{columns, tasks, docsDir, archived}` from a
snapshot. `archived` carries the archive-folder filenames. The web board does not
use it yet, so archived dependencies still show as missing there.
`POST /api/update` decodes `file`, `content`, `status`, `order`, and `tags` into
`UpdateRequest`; `unset` is not exposed by HTTP.

| Error/condition | HTTP status |
| --- | --- |
| Non-POST update | 405 |
| JSON decoding error | 400 |
| Missing/invalid filename, reserved status, nothing to update | 400 |
| Empty-clobber guard | 409 |
| Invalid frontmatter or other error | 500 |
| Successful update | 200 |

Error response text comes from `err.Error()`. The watcher callback broadcasts
`update` to `/api/events` clients. Each client has a one-message buffer;
nonblocking broadcasts coalesce duplicate notifications so a slow browser
cannot stall the watcher. Browsers fetch the latest board after notification.
Watcher termination is monitored alongside the HTTP server and causes
`runServe` to return; cancellation closes the watcher on shutdown.
