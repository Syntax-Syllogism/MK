# Draft Changelog for the v2.0 Release

Pulling this together from the commit log, split into the usual sections. No
frontmatter on this one yet — want to get sign-off on the wording before I file it
properly and assign it a status.

---

Added dynamic board columns driven by `docs/.kanban.yml` instead of a hardcoded list.

---

Added drag-to-reorder within a column, backed by a sparse `order` field.

---

Fixed a bug where dropping a card into a Git-conflicted file's column would silently
overwrite its status. Conflicted files are no longer draggable at all.
