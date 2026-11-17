# Preview-find terminal smoke results — review round 1

Verified on 2026-10-14 against implementation commit `82427a9`, addressing the
missing terminal acceptance checks reported at `docs/terminal-ui.md:260`.
No implementation changes were required.

The built MK binary ran in an actual pseudo-terminal with `TERM=xterm-256color`.
Keys were sent through the terminal input stream, screen text and ANSI cell
styles were inspected with pyte, and resize used `TIOCSWINSZ` plus `SIGWINCH`.
This was agent-driven terminal verification, without a human terminal observer.
All task changes were confined to a fresh copy of `test-work-items/` at
`/tmp/mk-preview-find-h7kx8evv/tasks`. Process working directory and preferences
were also isolated under that scratch directory.

The scratch long Markdown task had three mixed-case occurrences on one line,
a fourth occurrence after 40 paragraphs, Unicode text, Markdown emphasis,
and a dependency target. The board query was kept while opening its preview.

| Check | Observed result |
| --- | --- |
| Both openers | `/` and Ctrl+F opened preview-local find. |
| Literal phrase across soft wrap | `words needle phrase,` showed `1/1`; active highlighting covered the end of one rendered line and the start of the next, including punctuation. |
| Case-insensitive matches and styling | `needle phrase` showed `1/4`. The three visible occurrences used one yellow/brown underlined active highlight and two gray inactive highlights; the surrounding Unicode and emphasis remained rendered. |
| Forward navigation and wrap | Enter advanced through `2/4`, `3/4`, `4/4`, then `1/4`. The fourth match was revealed at the bottom of the long body. |
| Backward wrap | Shift+Enter (CSI `13;2u`) moved from `1/4` to `4/4` and revealed the last match. |
| Help | `?` displayed preview-find bindings without the prompt; Esc restored the query, focus, highlight, and `4/4` selection. |
| Resize | Resized with find active from 80×24 to 120×40 and back. The active last occurrence stayed visible with query and `4/4` intact; help also opened and restored at the larger size. |
| External refresh | Appending another occurrence changed `4/4` to `4/5` without losing selection. Removing all occurrences changed feedback to `no matches` and refreshed the body. |
| No-match navigation | Enter and Shift+Enter safely retained `no matches`. |
| Esc and dependency navigation | Esc closed find; Tab and Enter followed the dependency. Esc returned to the source preview with find cleared. The kept board query remained present. |
| Unicode and removed file | `Café` showed `1/1`. Deleting the open scratch task displayed the removed-file banner and retained searchable content and the match. |
| Read-only find | SHA-256 checks confirmed all scratch task bytes were unchanged before intentional external edits; other task files remained unchanged afterward. |
| Quit and terminal restoration | Ctrl+C while find was focused exited 0. Terminal attributes matched their pre-launch values exactly, and output released the alternate screen and restored cursor visibility. |

Validation commands, run through `rtk proxy` to preserve unfiltered output:

```sh
go build -o /tmp/mk-preview-find-smoke .
go test -count=1 ./internal/tui -run 'Test(PreviewFind|HelpBinding)'
go test ./...
go vet ./...
```

All exited 0. The documentation build passed during implementation; this round
does not change documentation site inputs. Terminal driver, screen captures,
cell styles, raw terminal output, and restoration summary are retained in the
run's `smoke-round-1/` evidence directory.
