# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0/).

## [Unreleased]

### Changed

- Shared store extracted into `internal/board`; README and GoReleaser build
  the whole package (`.`).
- Renamed the pristine QA fixture folder to `test-work-items/`.

### Added

- Task dependencies: an optional `depends_on` frontmatter list renders clickable dependency
  badges, with warnings for missing targets and dependency cycles.
- `mk tui` terminal board with filters, Markdown previews, and card actions: grab-and-move,
  quick column/neighbour moves, tag editing, `$EDITOR` handoff, copy path (OSC 52 over SSH),
  hash-guarded undo of TUI moves and tag edits, board/table views, live fuzzy
  search combined with chips, mouse selection and scrolling (`-no-mouse`),
  and generated mode-specific help.

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
