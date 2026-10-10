---
epic: QA
order: 8
project: MK
status: DONE
tags:
  - qa
  - smoke-test
  - docs
---

# Document the Smoke Test Plan for the Dynamic-Columns Release

Automated tests don't cover drag, tag, and save gestures, and several of the bugs found
in `updateTask` and `parseTask` were only reachable through those gestures. Run this
checklist by hand in a browser before a release that touches the board's write paths.

The fixtures in `test-work-items/` are real tickets as well as test cases. Each one is a real
task in a real shape, but its frontmatter or body is deliberately built to land on one of
the tricky code paths (malformed YAML, no frontmatter at all, an empty frontmatter block,
and so on). Run through the checklist below against them.

Don't run MK against `test-work-items/` itself. Dragging, tagging, and saving rewrite the
files, and a "malformed frontmatter" fixture that a successful save has repaired isn't
testing malformed frontmatter anymore. Copy the folder to a scratch directory and test the
copy. Edit `test-work-items/` only when you mean to change or add a fixture.

## Before you start

1. From the repo root, copy the fixtures to a scratch directory:
   `rm -rf /tmp/mk-smoke && cp -r test-work-items /tmp/mk-smoke`. Make a fresh copy any time a
   fixture looks like it has been repaired by an earlier pass.
2. Run the board against the copy: `go run . -dir /tmp/mk-smoke` (or
   `mk -dir /tmp/mk-smoke` if you've built or installed it; see
   [development instructions](../docs/development.md)).
3. Open it in a browser. You should see six columns, in this order: `CONFLICT`,
   `TODO`, `IN PROGRESS`, `UNDER REVIEW`, `DONE`, and `OTHER`. Only the onboarding card
   starts in `CONFLICT`. `OTHER` holds the fixtures with malformed or missing
   frontmatter or status, or a valid status outside the configured columns.
4. Have this list open somewhere else so you can tick items off as you go. Items start
   unticked. There's no in-app way to track them yet.

## Column and status handling

- [ ] **Existing conflicts cannot be dragged, and resolving them removes CONFLICT.**
      `resolve_merge_conflict_in_onboarding.md` already contains a Git conflict block
      around Step 3 (seven `<` and a branch name, a line of seven `=`, then seven `>`
      and a branch name). Its card starts in `CONFLICT` with a red "!!! GIT CONFLICT"
      marker. Confirm you **cannot drag it**; it shouldn't pick up under your cursor.
      In the copy, choose one version of Step 3 and remove all three marker lines and
      the rejected version. Save and watch the card move to `TODO` and the `CONFLICT`
      column disappear without refreshing.
- [ ] **A valid status that isn't a configured column routes to OTHER.**
      `verify_status_after_board_rename.md` has the old `status: REVIEW`. It starts in
      `OTHER` with no error banner and looks like a normal card because its frontmatter
      parses. Drag it into `UNDER REVIEW` and confirm the new status persists.
- [ ] **Board wraps instead of scrolling.** On a fresh copy, with all six columns present, resize the
      window down to a typical laptop width. Columns should wrap onto a second row.
      Confirm there is no horizontal scrollbar at any width you'd actually use.

## Malformed and missing frontmatter

- [ ] **Type-mismatched frontmatter lands in OTHER with a clear error.**
      `migrate_legacy_priority_field.md` has `order: asap`, which isn't a number. It should
      land in `OTHER` with a "!!! PARSE ERROR" marker on the card. Open it: the body should
      be the real ticket content, **not** an error message. The modal's editor and tag input
      should both be disabled (read-only), with a banner explaining the file needs fixing
      outside MK.
- [ ] **Empty frontmatter block doesn't crash the server.** `spike_notification_service.md`
      has an empty `---` / `---` block and no `status`, so it starts in `OTHER`. Drag it
      into any real column. The drop should succeed, because empty frontmatter is valid,
      just sparse. If the server falls over or the request hangs, the frontmatter
      writer's nil-map bug has come back.
- [ ] **A file with no frontmatter at all is still a normal, editable card.**
      `capture_customer_feedback_notes.md` has no frontmatter block, so it starts in `OTHER`
      with no `status:`. It should open normally and be fully editable. Add a tag and
      confirm the tag persists after the card re-renders. (This used to silently no-op.)
- [ ] **Dragging a frontmatter-less card preserves its body.** Drag
      `capture_customer_feedback_notes.md` into `TODO`. Then `diff` the copy against
      `test-work-items/capture_customer_feedback_notes.md`. Apart from the frontmatter MK added
      at the top (and any tags you added above), every line of the original body should
      still be there. This is the one that used to wipe the file entirely. If you see a
      mostly-empty card after the drag, stop and file a blocker.
- [ ] **A body that just happens to contain `---` isn't mistaken for frontmatter.**
      `draft_changelog_for_v2_release.md` starts with a heading and has no frontmatter.
      It starts in `OTHER` without a parse error. Open it and confirm the heading, intro,
      and all three `---`-separated release notes are visible and editable.
- [ ] **A real ticket with `---` dividers after real frontmatter still works normally.**
      `writeup_deploy_incident_postmortem.md` has proper frontmatter followed by three
      `---`-separated sections in the body. Confirm the whole postmortem renders, not just
      the first section.
- [ ] **An empty body is valid, and clearing a body is allowed.**
      `draft_pricing_page_copy.md` starts with placeholder pricing copy. Open it, select
      the entire body, delete it, and let it autosave. Close and reopen the card to
      confirm the body remains empty. The empty save
      should succeed; it should **not** come back as a save error. Don't bother restoring
      the text, since this is a scratch copy.

## Ordering and filtering

Use a fresh scratch copy before **each** ordering check below so earlier drags, status
changes, and conflict resolution don't change the starting order.

- [ ] **Drag-to-reorder within a column shows the "make room" animation.** Before you drag,
      the `TODO` column should read, top to bottom: `test_frontend.md` (order 1),
      `read_prd.md` (order 1.5), `wire_up_analytics_events.md` (order 2), and
      `polish_mobile_nav_transitions.md` (order 7). Drag the Polish card to the top,
      above `test_frontend.md`. The cards it passes should visibly slide down to
      make room rather than jump.
- [ ] **Reordering behind an active filter still computes the right position.** Select
      both the `docs` and `mobile` tags (tags combine with OR). The visible `TODO` cards
      are `read_prd.md` and `polish_mobile_nav_transitions.md`.
      `wire_up_analytics_events.md` is hidden but sits between them in the full order.
      Drag `read_prd.md` below `polish_mobile_nav_transitions.md`. Clear the filter:
      the full order should be `test_frontend.md`, `wire_up_analytics_events.md`,
      `polish_mobile_nav_transitions.md`, then `read_prd.md`. Analytics should still
      be above Polish, and Read PRD should sit directly below Polish.
- [ ] **Filter collapse button is visible and works.** Confirm the `filters` control near
      the top reads as an actual button (bordered, not just faint text), toggles the filter
      chips open and closed, and the chevron flips direction accordingly.

## Live sync

- [ ] **External edits show up without a refresh.** With the board open, edit any card's
      `status:` directly in the copy and save. The card should move columns in the browser
      within about a second, with no manual refresh.
- [ ] **Editing `.kanban.yml` updates the columns live.** In `/tmp/mk-smoke/.kanban.yml`,
      add another configured column name and save. A new empty column should appear on the board
      without restarting the server. Remove it again afterward.

## Wrapping up

If everything above checks out, this ticket is done and the branch is ready to merge. If
something doesn't check out, don't fix it quietly. File it as its own ticket in the `QA`
epic (or reopen this one), so there's a record of what broke and what the repro was. This
is how the `mk-dynamic-columns-and-card-order` work item has tracked review findings.

When you're done, delete the copy with `rm -rf /tmp/mk-smoke`, and start the next pass from
a fresh `cp`. If a fixture needs to change on purpose, such as for a new scenario or a
wording fix, edit it in `test-work-items/`, not the copy.
