# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0/).

## [0.2.0] - 2026-07-21

### Added
- Dynamic columns and card ordering
- Project frontmatter field with filtering
- Version flag and build metadata
- Epic filtering
- Connection loss feedback
- ESC key support
- Strict path validation
- Additional markdown support

### Changed
- Improved header and list styling
- Expanded README with platform-specific setup instructions and project sample

## [Unreleased]

### Added

- Dynamic board columns, defined by a committed `docs/.kanban.yml` (auto-bootstrapped on
  first run from whatever statuses are already in your task files).
- Manual card ordering within and across columns via drag-and-drop, persisted as an
  `order:` float in frontmatter.
- Collapsible filter bar with an active-filter count and clear-all control.

### Fixed

- Dragging a card into the CONFLICT column no longer writes `status: CONFLICT` into its
  frontmatter.

### Changed

- Any task whose `status` is not one of your configured columns (including tasks with no
  `status` at all, or frontmatter that fails to parse) now renders in a diagnostic `OTHER`
  column instead of disappearing. If you upgrade with statuses outside the old
  `TODO / IN PROGRESS / DONE` set, add them to `docs/.kanban.yml` to give them a real
  column. See [USER_GUIDE.md](USER_GUIDE.md) for day-to-day usage of all of the above.

## [0.1.0] - 2026-05-23

### Added

- Epic filtering
- Connection loss feedback
- ESC key support
- Strict path validation
- Header and list styling
- Additional markdown support
- Version flag
- Build metadata

## [0.1.0] - 2026-05-23

### Added

- Version flag and build metadata
- Epic filtering
- Connection loss feedback
- ESC key support
- Strict path validation
- Header and list styling
- Additional markdown format support
