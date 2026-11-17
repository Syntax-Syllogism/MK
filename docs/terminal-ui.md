---
title: Terminal board
description: Use the mk tui terminal board, with its keys, card actions, search, filters, previews, undo, mouse support, and preferences.
---

# Terminal board

`mk tui` opens a full-screen board using `docs/` as the task
directory. Select another directory with `mk tui -dir <path>`.
`mk tui -version` prints build information without opening a store or terminal
UI. Running `mk` without the subcommand still starts the web server; `-port`
is a web-server option and is not accepted by `mk tui`.

The TUI reads the same top-level Markdown files and `.kanban.yml` as the web
board. Card actions write through the shared board store. Opening a store can create the task
directory and bootstrap a missing `.kanban.yml`; scans write the disposable
`.kanban/board.json` cache relative to the process working directory. UI
preferences are stored separately, as described below.

## Board and table

The board shows CONFLICT first when present, every configured column even
when empty, and OTHER last when present. Filters hide cards, not sections.
Cards sort by `order` ascending, with unset orders last, then by filename.
Section headers show visible counts, or visible/total counts while filtering.
CONFLICT and OTHER have a `⊘` diagnostic marker.

Compact cards are the default: filename, tags, and a dependency count when
present. Both compact and full cards show the first Markdown H1 on a dim,
single line directly below the filename, truncated with `…` when needed.
Cards without an H1 omit that line. Only ATX headings (`# Heading`) outside
fenced code blocks are used; Setext headings (`Heading` followed by `===`)
are not detected. The filename remains the task ID. Full cards also show
project, epic (or `NO EPIC`), and dependency slugs. Conflict and parse-error
cards show their diagnostic labels in full mode. Missing dependencies have a
`?` suffix; archived dependencies show `archived` instead and are not missing;
cycle members show `cycle`.
Dependency count badges warn when a target is missing or the task is in a
cycle. See [task-file rules](task-files.md) for the underlying schema.

Narrow terminals show a window of columns around the focused section, with
edge markers counting hidden sections. Each column scrolls independently to
keep its selected card visible, with indicators for additional cards.
Board colours use the terminal's ANSI colours; Markdown previews use the
reported light/dark background. There is no theme-toggle key.

The table lists FILE, TITLE (the H1 heading), PROJECT, EPIC, STATUS, TAGS,
and DEPS. FILE reserves at least 40% of the width; TITLE shares the remaining
space with metadata, with long values truncated using `…`. Empty metadata is
shown as `-`; DEPS is a count followed by `!` when there is a dependency
warning. Rows follow board section order, then card order within each section.
This differs from the web table's sorting by status text.

## Keymap

This is the canonical keymap. `?` generates scrollable help from the keymap for
the current mode plus global bindings. `Esc`, `q`, `?`, or `!` closes help; the
wheel scrolls it. A curated help line appears at widths of 60 columns or more. It prioritises
the current mode’s useful actions and shows more as the terminal gets wider;
`?` stays last, with undo depth shown separately when available.
Text entry consumes ordinary characters, including `q`, `r`, and `!`;
`?` opens help in every mode. Grab mode accepts its listed keys and help only.

| Mode | Key | Action |
| --- | --- | --- |
| Global | `q`, `ctrl+c` | Quit; `q` leaves preview; `ctrl+c` first cancels grab/tags |
| Global | `?`, `!`, `r` | Help, warnings, rescan (ordinary text while typing, except `?`) |
| Board | `h/l`, Left/Right | Previous/next section |
| Board, table | `j/k`, Down/Up | Next/previous card or row |
| Board, table | `g/G`, `ctrl+d/u` | First/last, half page |
| Board, table | `Enter` | Open preview |
| Board, table | `v`, `z` | Switch view, toggle compact/full board cards |
| Board, table, filters | `f`, `F`, `p` | Focus/leave filters, clear chips and query, collapse/expand panel |
| Board, table | `/`, `ctrl+f` | Open live search; table ranks filename/heading matches by score |
| Board, table | `n/N` | Next/previous match in section reading order |
| Board, table | `Esc` | Clear kept search |
| Board | `Space` | Grab selected card |
| Board | `c` | Collapse/expand focused column |
| Board, table, preview | `A` | Archive selected card; `u` undoes |
| Board | `H/L`, `J/K` | Move to previous/next column bottom, reorder below/above visible neighbour |
| Board, table, preview | `t`, `e`, `y` | Tags, external editor, copy absolute path |
| Board, table, preview | `u` | Guarded undo, up to 50 actions |
| Grab | `h/l`, Left/Right | Target configured column |
| Grab | `j/k`, Down/Up, `g/G` | Insertion position, top/bottom |
| Grab | `Enter`, `Esc`, `ctrl+c` | Drop, cancel, cancel |
| Preview | `j/k`, Down/Up, `g/G` | Scroll, top/bottom |
| Preview | `ctrl+d/u` | Half page down/up |
| Preview | `Space`, PageDown, `b`, PageUp | Page down, page up |
| Preview | `h/l`, Left/Right | Horizontal scroll |
| Preview | `ctrl+f`, `/` | Find literal text in the displayed body |
| Preview find | Type, `Enter`, `Shift+Enter`, `Esc` | Find live, next occurrence, previous occurrence, clear and close |
| Preview | `Tab`, `Shift+Tab`, `Enter` | Select next/previous dependency, follow it |
| Preview | `Backspace`, `Esc`, `q` | Dependency history/back, history/back, leave directly |
| Filters | `h/l`, Left/Right, `j/k`, Down/Up | Chip, group |
| Filters | `Space`, `Enter`, `Esc` | Toggle chip, toggle chip, return |
| Filters | `g/G`, `ctrl+d/u`, PageDown/PageUp | First/last chip, page through chips |
| Filters | `/`, `ctrl+f` | Find chips in the focused group |
| Chip find | Type, `Space`/`Enter`, `Esc` | Narrow chips, toggle while finding, clear and close |
| Chip find | `h/l`, Left/Right, `j/k`, Down/Up | Chip, change group and close find |
| Chip find | `ctrl+d/u`, PageDown/PageUp | Page through chips and keep finding |
| Tag popup | `Enter`, `Tab`, `Shift+Tab` | Add comma-separated tags, switch input/list |
| Tag popup list | `j/k`, Down/Up, `d/x`, Delete | Select tag, remove tag |
| Tag popup | `Esc`, `ctrl+c` | Save once, discard |
| Board, table, preview, filters | `:` | Open command palette |
| Palette | Type, Up/Down, `ctrl+n/p`, `Enter`, `Esc`, `ctrl+c` | Fuzzy filter, select action, run, close, quit |
| Search | Type, `Enter`, `Esc`, `ctrl+c` | Filter live, keep query, clear and close, quit |
| Help/warnings | `j/k`, Up/Down, `h/l`, Left/Right | Scroll vertically/horizontally |
| Help/warnings | `Space`, PageDown, `b`, PageUp, `ctrl+d/u` | Page down/up, half page down/up |
| Help/warnings | `Esc`, `q`, `?`, `!` | Close overlay |

## Command palette

Press `:` on the board, table, preview, or filter panel to find an action by
name or key. The palette uses the same keymap as `?` help and lists the current
mode’s actions plus global actions, excluding scrolling and ordinary navigation.
Type a fuzzy query (for example `tag`), select with Up/Down or `ctrl+n/p`, and
press `Enter` to run the selected action. `j/k` are query text. `Esc` closes
without running anything, and `ctrl+c` quits. `?` opens help for the palette;
closing help returns to the palette with its query kept. Up to 12 results
are visible; moving down reveals more.

The palette is unavailable during grab or text entry. `:` remains ordinary text
in search, preview find, and the tag input. Mouse input is ignored in the palette.

## Search

`/` or `ctrl+f` opens a one-line prompt in the status bar. Search is case-insensitive:
each whitespace-separated term must match either a fuzzy filename (without
`.md`), a fuzzy H1 heading, or a substring in project, epic, tags, dependency
slugs, or body.
Body matching is substring-only. Queries combine with chip filters using AND.
`Enter` keeps the query and shows `search: "<query>"` in the top bar; `Esc`
in the prompt clears it. On the board/table, `Esc` clears a kept query and
`F` clears both chips and query. Queries are session-only.

Matched characters in the filename and heading are bold and underlined,
only on the lines that match. Board cards retain their spatial order. The
table sums the better filename or heading score for each term and sorts by
that total descending, breaking ties with section/card order. `n/N` cycles
visible matches in section order, top to bottom, wrapping at either end. When
filtering hides a selection, the first
visible card in that section is selected; empty sections move focus to the
next visible section to the right, then to the left.

## Mouse

Mouse capture is enabled by default. `mk tui -no-mouse` disables it so native
terminal text selection works. Click a card or table row to select it; click
that selected card again to open its preview. Clicking a column, including its
header or empty space, focuses its section; cards take precedence. Chips toggle
on click, and the top filter indicator expands a collapsed panel. Preview
dependency slugs jump to their targets.

The vertical wheel over a filter group scrolls its chips by one line without
changing focus or selection. Over the board, it scrolls the column under the
pointer by one card, keeping selection
unless it leaves the viewport, when it is clamped. In the table, preview,
warnings, and help it scrolls three lines. Horizontal wheel or Shift+wheel
moves focus to the previous or next section, shifting the board's section
window. Mouse input is ignored during grab, tags, search, chip find, preview find, and the palette, except help
scrolling. Release, motion, and right/middle clicks are ignored. There is no
mouse drag-and-drop.

## Filters

The panel contains project, epic, and tag chips drawn from all tasks, including
`NO EPIC` for tasks without an epic. Multiple selections in one group combine
with OR; selections across groups combine with AND.

Press `f` to show and focus the panel. Use `h/l` or Left/Right between chips,
and `j/k` or Down/Up between groups. `Space` or `Enter` toggles a chip;
`F` clears all selections. `f` or `Esc` returns to the board/table while
leaving the panel visible. `p` collapses it and returns focus if needed.
Each group remembers its focused chip and scroll position. Chips wrap into
lines and scroll just far enough to reveal keyboard focus. The focused group
gets extra lines; the board shrinks to make room. `g/G` jumps to the first/last
chip, and `ctrl+d/u` or PageDown/PageUp moves by a window of chip lines.
Dim `↑N` and `↓N` markers count chips hidden above and below. Selected chips
sort alphabetically at the front, in a pinned block above the scrolling chips;
toggling retains focus on the same value. If selected chips alone exceed the
group's line budget, the leading lines stay pinned and the remainder scrolls,
so every chip remains reachable.

Press `/` or `ctrl+f` in the panel to find chips in the focused group by
case-insensitive substring. Selected chips remain visible even if they do not
match. `Space` or `Enter` toggles the focused chip and keeps the prompt open.
`h/l` or Left/Right navigates chips; `ctrl+d/u` or PageDown/PageUp pages through
them; `j/k` or Down/Up changes group and closes find. Those navigation keys are
reserved while finding; other ordinary keys,
including `q`, `f`, `F`, `r`, `!`, and `:`, are query text. `Esc` clears and
closes find, `?` opens help and restores find on return, and `ctrl+c` quits.
Chip find only narrows the displayed chips; it does not search cards.
Scroll positions and chip-find text are session-only.

Active filters continue to apply when the panel is collapsed. A selected
facet that disappears from the task set is removed on reload.

## Task preview and dependencies

The preview header shows project, epic, filename, status, tags, dependencies,
and the absolute file path. OTHER tasks retain their actual status followed
by `(OTHER)`. Parse errors show a banner directing you to press `e` and fix the
YAML in an external editor. The body renders Markdown; CONFLICT bodies remain raw
so merge markers are visible. `t`, `e`, and `y` act on the open task; see
[card actions](#card-actions).

Use `j/k`, Up/Down, `ctrl+d/ctrl+u`, and `Space` / `b` to scroll.
`Tab` / `Shift+Tab` selects a dependency, and `Enter` follows it. Missing
targets show `unknown card: <slug>` without leaving the current preview.
Archived targets show `archived: <slug>` and do not open.
`Backspace` or `Esc` returns through dependency history, then leaves the
preview when the history is empty. `q` leaves the preview directly.

Press `ctrl+f` or `/` to find case-insensitive literal text in the displayed
body. Metadata and hidden Markdown syntax are excluded. Phrases remain
searchable across soft wraps. All occurrences are highlighted; the active
occurrence is underlined with a distinct background. The status prompt shows
its current/total count or `no matches`. `Enter` / `Shift+Enter` move forward /
backward with wraparound, keeping the prompt focused and revealing the match.
Typing selects the first occurrence at or below the current scroll position,
wrapping to the first occurrence when none is below it. An empty query removes
highlights.

`Esc` clears and closes find while keeping your scroll position; a subsequent
`Esc` follows the usual dependency history/back behavior. `?` suspends find for
help and restores it when help closes. Ordinary action keys are query text
while finding, and `ctrl+c` quits. Find survives resizing and content refresh,
but clears when opening another task or leaving preview. Queries are
session-only and independent of board/table search and chip filters.

Returning to the board/table selects the last viewed task. If filters hide
that task, they are cleared with a notification. External content changes
refresh the preview while retaining its scroll offset where possible. If
the file disappears, its last content remains with a `file removed` banner.

## Card actions

`A` archives the selected card from the board, table, or preview without a
prompt. Cards of any status can be archived; CONFLICT and parse-error cards are
refused. The file moves unchanged into `archive_dir` (default `archive`), and
preview returns to the previous view. A name collision refuses the action.
`u` moves it back only if the archived bytes are unchanged and the original
name is free. Archive shares the 50-action undo stack with moves and tags.

On the board, `c` collapses the focused column to a five-cell strip showing
its abbreviated name and count. Focus moves to an expanded column, preferring
the right; the last expanded column cannot be collapsed. `h/l` skips strips.
Click a strip to expand and focus it; its mouse wheel does nothing. `H/L`
and grab still target collapsed configured columns, dropping at the real
bottom even under filters. CONFLICT and OTHER can also collapse. Collapse is
remembered per column name and board, including names temporarily removed
from configuration. The table is unaffected.

On the board, `Space` grabs a card. Its dimmed origin stays in place and a
highlighted double-border ghost shows the destination. Use `h/l` or Left/Right
to select a configured column, `j/k` or Down/Up for the insertion position,
and `g/G` for top/bottom. `Enter` drops; `Esc` or `ctrl+c` cancels. Other keys
are ignored except `?` for help. A second `ctrl+c` quits after cancellation.

`H/L` sends a card to the real bottom of the previous/next configured column;
`J/K` reorders it relative to its visible neighbour. Reordering under filters
uses hidden cards when computing the new order. CONFLICT and parse-error cards
cannot move or change tags; use `e` to fix them. Valid OTHER cards can move to a
configured column. Grab and quick-move keys apply only to the board.

From the board, table, or preview, `t` opens the tag popup. Enter comma-separated
tags and press `Enter` to add them, normalised to lowercase with surrounding
spaces removed. Duplicates are skipped with an `already tagged` notice. `Tab`
switches to the tag list, `j/k` selects, and `d`, `x`, or Delete removes. `Esc` saves once; `ctrl+c` discards. An external change while the
popup is open cancels saving rather than overwriting the changed file.

`e` releases the terminal for `$VISUAL`, then `$EDITOR`, defaulting to `vi`
(`notepad` on Windows). Quoted executable paths and arguments are supported.
GUI editors such as VS Code should use `EDITOR="code --wait"`. Tasks rescan on
return, including after an editor error; later GUI saves also trigger the watcher.
Editor changes are not undoable.

`y` copies the absolute task path using the native clipboard, falling back to
OSC 52 through the terminal. SSH uses OSC 52 directly. The terminal must support
OSC 52; tmux requires `set -g set-clipboard on` and `set -g allow-passthrough on`.

`u` on the board, table, or preview undoes the last move, tag, or archive action. The in-memory
stack holds up to 50 actions and is lost on quit. Undo restores exact original
bytes, including absent keys and YAML formatting, and includes any unordered
tail cards assigned orders by a bottom drop. It refuses and discards an entry
if any affected file changed since the action. Errors appear in red for five
seconds. The status bar shows undo depth.

## Reloads, warnings, and preferences

External task and `.kanban.yml` changes reload through the shared filesystem
watcher. The TUI keeps selections by filename within each section. If a
selected task moves or disappears, focus stays in the current section and
selection falls back to its previous position, clamped to the remaining
cards. `r` requests a manual scan and config reload. Config edits never
rewrite task statuses; see [Columns](columns.md) for column rules.

New warnings appear briefly in the status bar. `!` opens a scrollable list
of the last 50 warnings, including scan, config, and watcher errors.
Help and warning overlays close with `Esc`, `q`, `?`, or `!`.

View (board/table), card density, column collapse, and filter-panel collapse state persist per
absolute task-directory path. Active filter selections and search queries are session-only.
Defaults are board view, compact cards, and an expanded panel.

- Linux: `$XDG_STATE_HOME/mk/<key>.json`, defaulting to
  `~/.local/state/mk/<key>.json`.
- macOS/Windows: the user config directory's `mk/state/<key>.json`.

`<key>` is the first 16 hex characters of SHA-256 of the absolute task
directory. Unreadable or invalid preferences fall back to defaults. Writes
use a temporary file and rename; storage errors are silently ignored.

## Implementation and smoke testing

`tui.go` parses flags and opens the shared store with a warning sink.
`internal/tui/model.go` owns snapshots, selection, modes, and the watcher
lifecycle. Board/table rendering, filters, previews, logs, layout,
search (`search.go`), mouse hit regions (`mouse.go`),
and preferences live in their corresponding files under `internal/tui/`.
The keymap in `keys.go` is the single source for the table above, the `?` help
text, the curated hint line, and the command palette: change a binding there and
update this page. `palette.go` holds the palette; it dispatches a chosen action
by replaying its key message through `Update`, so it takes the same code paths
as the key.
Card actions are split across `grab.go` (grab mode, drops, quick moves),
`tags.go` (tag popup), `editor.go` (`$EDITOR` handoff), `clipboard.go` (copy
path), and `undo.go` (undo stack and action application). Archive actions live in
`archive.go`, which calls `Store.Archive` and `Store.Unarchive`, and column
strips live in `collapse.go`. The TUI consumes snapshots and view helpers, and
writes through `Store.ApplyBatch` and the TUI-only `Restore` path.
See [board store](board-store.md) for the shared API and
[development](development.md) for automated checks and fixture safety.

For a manual smoke check, copy `test-work-items/` to an unused scratch directory
and run `go run . tui -dir <scratch>`. Check board/table switching, both card
densities, filters, previews, dependency jump/back, help, and warnings. Check
card actions: grab and move a card, `H/L/J/K` quick moves, tag add/remove, `e`
with an editor, `y`, `/` search combined with chips, `n/N`, mouse selection/scrolling,
`-no-mouse`, and `u` back to byte-identical files (`git diff` is empty
in a `git init` copy). Also confirm that `u` refuses after an external edit.
In a long scratch task, open preview find with both `ctrl+f` and `/`; check
case-insensitive literal phrases across soft wraps, all/active highlights,
Enter/Shift+Enter wraparound, no matches, help restoration, and Esc followed
by normal dependency navigation. Resize with find active and edit the body
externally to check that matches refresh.
Check `A` archives from board, table, and preview, that `u` restores the file
byte-identical, and that a dependency on the archived card shows `archived`.
Check `c` collapses and expands board columns, that `h/l` skips strips, that a
card can be dropped into a strip with grab or `H/L`, and that clicking a strip
expands it. Restart to check collapsed columns persist.
Check `p` collapses the filter panel and persists after restart; `ctrl+f`
opens search on board/table and find in preview. After changing a tag, press
`u` in preview and confirm the restored tags appear. Confirm `d/u/f` no longer
scroll preview/help while `ctrl+d/u`, Space/PageDown, and b/PageUp still do.
Open `:` in board/table/preview/filters, filter for an action, run it with Enter,
and cancel with Esc. Check colon stays text in search/tags/find and is ignored
in grab. Resize to widths 60, 80, 120, and 200 and check the hint line grows,
fits, and keeps help and undo depth visible.
In a scratch board with 300+ tags, focus filters and traverse all tags at
80×24 and a larger terminal. Check hidden-chip counts, paging, per-group focus
restoration, pinned selections, wheel scrolling, and clicks after scrolling.
Find chips by substring with `/` and `ctrl+f`; toggle with Space/Enter, type
action keys as query text, open/close help, clear with Esc, and change groups.
Remove tags externally and confirm focus and scroll clamp after reload.
Resize to 80×24 and a larger terminal. Edit task content/status and column
config externally, then check reload and selection behavior; remove an open
task and check the retained preview banner. Restart against the same scratch
directory to check preferences. Quit and verify terminal restoration.
Keep all testing changes in the scratch copy.
