# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0/).

## [Unreleased]

## [0.4.1] - 2026-11-17

### Added

- TUI `A` archives cards into configurable `archive_dir` (default `archive`), with guarded undo. Archived dependencies remain known and display as archived
- TUI `c` collapses board columns into persisted name/count strips. Navigation skips strips; quick moves and grab can still drop into them
- TUI cards show the first Markdown H1 below the filename in both densities; the table adds a TITLE column. Search fuzzy-matches and highlights headings, ranking table results by the better filename or heading score. Task JSON exposes the optional `heading` field
- TUI chip find (`/` or `ctrl+f` in filters), first/last chip navigation, page navigation, and mouse-wheel scrolling over each filter group
- Command palette in the TUI (`:`): fuzzy-search current-mode and global actions from the shared help keymap, then press Enter to run one
- Task dependencies: an optional `depends_on` frontmatter list renders clickable dependency badges, with warnings for missing targets and dependency cycles
- Terminal board via `mk tui` with filters, Markdown previews, and card actions: grab-and-move, quick column/neighbour moves, tag editing, `$EDITOR` handoff, copy path (OSC 52 over SSH), hash-guarded undo of TUI moves and tag edits, board/table views, live fuzzy search combined with chips, mouse selection and scrolling (`-no-mouse`), and generated mode-specific help
- Find within a `mk tui` preview (`ctrl+f` or `/`): live case-insensitive literal matches in the displayed body, all occurrences highlighted, Enter/Shift+Enter navigation with wraparound, and current/total or no-match feedback

### Changed

- TUI filter groups now scroll by wrapped lines, show hidden-chip counts, remember focus per group, and give the focused group more space. Selected chips are pinned first while keyboard focus stays on the toggled value
- TUI keys: `p` replaces `ctrl+f` for filter-panel collapse; `ctrl+f` opens board/table search or preview find. Preview `u` now undoes actions; plain `d`, `u`, and `f` no longer scroll previews or overlays. Use `ctrl+d/u` for half pages and Space/PageDown or b/PageUp for full pages
- The TUI hint line prioritises actions per mode and expands with terminal width
- Shared store extracted into `internal/board`; README and GoReleaser build the whole package (`.`)
- Renamed the pristine QA fixture folder to `test-work-items/`

### Fixed

- Corrected the J/K reorder label in the TUI help

## [0.4.0] - 2026-10-10

### Added

- Read-only terminal board with filters and Markdown previews
- Card actions in the terminal UI
- Guarded undo for card actions in the terminal UI
- Fuzzy search in the terminal UI
- Mouse controls in the terminal UI
- Generated help in the terminal UI
- Task dependencies are now rendered
- Restructured documentation with a new Starlight docs site

### Fixed

- Stale documentation corrected

## [0.3.4] - 2026-08-17

### Removed

- `scripts/reset-smoke-fixtures.sh`, `scripts/reset-smoke-fixtures.ps1`, and
  `scripts/test-reset-smoke-fixtures.ps1`.

## [0.3.3] - 2026-08-17

### Removed

- `test-docs/` smoke-test fixture copies.

## [0.3.2] - 2026-08-17

### Changed

- Sample task fixtures moved from `docs/` to `sample-docs/`.

## [0.3.1] - 2026-08-17

### Changed

- Internal maintenance and tooling updates

## [0.3.0] - 2026-08-17

### Added

- Copy path to file button in the card modal.

### Fixed

- Dragging a card into the CONFLICT column no longer writes `status: CONFLICT` into its
  frontmatter.

## [0.2.0] - 2026-07-21

### Added

- Dynamic board columns, defined by a committed `docs/.kanban.yml` (auto-bootstrapped on
  first run from whatever statuses are already in your task files).
- Manual card ordering within and across columns via drag-and-drop, persisted as an
  `order:` float in frontmatter.
- Collapsible filter bar with an active-filter count and clear-all control.
- Project frontmatter field with filtering.

### Changed

- Any task whose `status` is not one of your configured columns (including tasks with no
  `status` at all, or frontmatter that fails to parse) now renders in a diagnostic `OTHER`
  column instead of disappearing. If you upgrade with statuses outside the old
  `TODO / IN PROGRESS / DONE` set, add them to `docs/.kanban.yml` to give them a real
  column. See [USER_GUIDE.md](USER_GUIDE.md) for day-to-day usage of all of the above.
- Expanded README with platform-specific setup instructions and project sample

## [0.1.0] - 2026-05-23

### Added

- Version flag and build metadata
- Epic filtering
- Connection loss feedback
- ESC key support
- Strict path validation
- Header and list styling
- Additional markdown format support
