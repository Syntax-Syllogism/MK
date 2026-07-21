---
epic: QA
order: 0
project: MK
status: UNDER REVIEW
tags:
    - process
---
# Verify Ticket Status After the Board Rename

We renamed a couple of columns last sprint (`REVIEW` became `UNDER REVIEW`, and we
dropped `BACKLOG` in favor of just starting things in `TODO`). A handful of tickets
still have their old status value sitting in frontmatter and never got migrated — this
is one of them.

Go through the board, find any ticket sitting outside the real columns, and either
update its `status:` to match the current list or file a follow-up if it's genuinely
stale and should be closed instead.