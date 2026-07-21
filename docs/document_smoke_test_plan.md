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

We've been finding real bugs in `updateTask` and `parseTask` in every review
pass of the dynamic-columns work — most of them only reachable through an actual
drag or save gesture, not from reading the code. Before this ships, someone
needs to click through the board by hand and confirm the fixes actually hold up
outside of `go test`.

Seeded the `QA` epic with a set of tickets that double as fixtures: each one is
a real task in a real shape, but its frontmatter or body is deliberately built
to land on one of the tricky code paths (malformed YAML, no frontmatter at all,
a live merge conflict, etc.). Run through the checklist below against them.

Learned the hard way (round 6 of code review) that committing these directly to
`docs/` doesn't work: the whole point of a smoke test is to drag, tag, and save
the fixtures, and doing that repairs them — a "malformed frontmatter" fixture
that just got its frontmatter rewritten by a successful save isn't testing
malformed frontmatter anymore. The pristine fixtures now live in `test-docs/`
instead, which nothing in the app ever reads or writes. `docs/` only gets a
_copy_, made by the reset script below, and that copy is what you're free to
beat up.

## Before you start

1. From the repo root, run `scripts/reset-smoke-fixtures.sh` to copy fresh
   fixtures from `test-docs/` into `docs/`. Run this again any time you suspect
   a fixture got "repaired" by a previous smoke-test pass — it's idempotent and
   safe to rerun.
2. Run the board against this repo's `docs/` folder: `go run main.go` (or `mk`
   if you've built/installed it — see the README's "Developing MK" section if
   you're not sure which).
3. Open it in a browser. You should see six columns: `CONFLICT`, `TODO`,
   `IN PROGRESS`, `UNDER REVIEW`, `DONE`, and `OTHER` — the `QA` epic tickets
   alone are enough to populate `CONFLICT` and `OTHER` without touching anything
   else on the board.
4. Have this list open somewhere else so you can check items off as you go.
   There's no in-app way to do that yet (maybe that's the next ticket).

## Column and status handling

- [x] **CONFLICT column appears and is read-only.** "Resolve Leftover Merge
      Conflict in Onboarding Doc" should show up in a `CONFLICT` column, pinned
      first, with a red "!!! GIT CONFLICT" marker. Confirm you **cannot drag
      it** to another column — the card shouldn't even pick up under your
      cursor.
- [x] **Unrecognized status routes to OTHER.** "Verify Ticket Status After the
      Board Rename" has `status: Needs QA`, which isn't one of the four real
      columns. It should land in `OTHER`, look like a completely normal card (no
      error banner — this is a valid, parseable status, just not a configured
      one), and be freely draggable into a real column.
- [x] **Board wraps instead of scrolling.** With all six columns present, resize
      the window down to a typical laptop width. Columns should wrap onto a
      second row. Confirm there is no horizontal scrollbar at any width you'd
      actually use.

## Malformed and missing frontmatter

- [x] **Type-mismatched frontmatter lands in OTHER with a clear error.**
      "Migrate Legacy `priority` Field Into `order`" has `order: asap` — not a
      number. It should land in `OTHER` with a "!!! PARSE ERROR" marker on the
      card face. Open it: the body text should be the real ticket content,
      **not** an error message, and the modal's editor + tag input should both
      be disabled (read-only) with a banner explaining the file needs fixing
      outside MK.
- [x] **Empty frontmatter block doesn't crash the server.** "Spike: Notification
      Service Options" has an empty `---\n---\n` block. Drag it into any real
      column. This should succeed normally (empty frontmatter is valid, just
      sparse) — if the whole server falls over or the request hangs, that's
      `main.go`'s nil-map bug back from the dead.
- [x] **A file with no frontmatter at all is still a normal, editable card.**
      "Capture Customer Feedback From the Q3 Call" has no frontmatter block. It
      should land in `OTHER` (no `status:` at all), open normally, and be fully
      editable — add a tag to it and confirm the tag actually persists after the
      card re-renders (this used to silently no-op).
- [x] **Dragging a frontmatter-less card preserves its body.** Same ticket: drag
      it into `TODO`. Reopen it afterward and confirm the full body — all three
      bullet points — is still there. This is the one that used to wipe the file
      entirely; if you see a mostly-empty card after the drag, stop and file a
      blocker.
- [x] **A body that just happens to contain `---` isn't mistaken for
      frontmatter.** "Draft Changelog for the v2.0 Release" has no frontmatter
      but uses `---` as a section divider three times in the body. Open it and
      confirm all three sections (dynamic columns, drag-to-reorder, the
      conflict-drag fix) are visible — not just the last one.
- [x] **A real ticket with `---` dividers _after_ real frontmatter still works
      normally.** "Write Up the Tuesday Deploy Incident Postmortem" has proper
      frontmatter followed by three `---`-separated sections in the body.
      Confirm the whole postmortem — What Happened / Impact / Follow-ups —
      renders, not just the first section.
- [x] **Clearing a card's body on purpose is allowed.** Open "Draft Pricing Page
      Copy", select all the text in the editor, delete it, and let it autosave.
      This should succeed (the card's body becomes empty) — it should **not**
      come back as a save error. No need to put the text back afterward; that's
      what the reset script is for.

## Ordering and filtering

- [x] **Drag-to-reorder within a column shows the "make room" animation.** The
      `QA` epic's three `TODO` tickets ("Design the Empty-State Illustration",
      "Wire Up Board Analytics Events", "Polish Mobile Nav Transitions") start
      in that order. Drag the third one to the top and confirm the other two
      visibly slide down to make room rather than jump.
- [x] **Reordering behind an active filter still computes the right position.**
      Filter the `design` tag chip so "Wire Up Board Analytics Events" (tagged
      `analytics`, not `design`) is hidden. With it hidden, drag "Polish Mobile
      Nav Transitions" so it visually sits directly above "Design the
      Empty-State Illustration". Clear the filter and confirm the hidden
      analytics ticket landed in a sensible spot instead of getting skipped over
      in the order calculation.
- [x] **Filter collapse button is visible and works.** Confirm the `filters`
      control near the top reads as an actual button (bordered, not just faint
      text), toggles the filter chips open and closed, and the chevron flips
      direction accordingly.

## Live sync

- [x] **External edits show up without a refresh.** With the board open, edit
      any ticket's `status:` directly in your editor and save. The card should
      move columns in the browser within about a second, with no manual refresh.
- [x] **Editing `docs/.kanban.yml` updates the columns live.** Temporarily add a
      fifth column name to the file and save. A new empty column should appear
      on the board without restarting the server. Remove it again afterward.

## Wrapping up

If everything above checks out, this ticket is done and the branch is ready to
merge. If something doesn't check out, don't fix it quietly — file it as its own
ticket in the `QA` epic (or reopen this one) so there's a record of what broke
and what the repro was, the same way the `mk-dynamic-columns-and-card-order`
work item has been tracking every review finding so far.

Once you're done with a pass, rerun `scripts/reset-smoke-fixtures.sh` so the
next person starts from clean fixtures instead of whatever state you left them
in — don't hand-edit `docs/`'s copies back into shape. If a fixture needs to
change on purpose (a new scenario, a wording fix), edit it in `test-docs/`, not
`docs/` — `docs/` is disposable output, not the source.
