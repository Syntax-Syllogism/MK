#!/usr/bin/env bash
# Keyboard smoke test for `mk tui`, driven through tmux.
#
#   scripts/tui-smoke.sh [--bin PATH] [--only REGEX] [--keep] [--verbose]
#
# Safety:
#   * Works on a scratch copy of test-work-items/ (plus two dependency fixtures);
#     the pristine fixtures and docs/ are never touched.
#   * Uses a private tmux server (`tmux -L <socket>`), so it never touches your own
#     tmux sessions, even when run from inside tmux. It never calls a bare
#     `tmux kill-server`.
#   * UI preferences go to a scratch XDG_STATE_HOME.
#
# Requires: bash, tmux, git, go (unless --bin is given).
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN=""; ONLY=""; KEEP=0; VERBOSE=0
while [ $# -gt 0 ]; do
  case "$1" in
    --bin) BIN="$2"; shift 2 ;;
    --only) ONLY="$2"; shift 2 ;;
    --keep) KEEP=1; shift ;;
    --verbose|-v) VERBOSE=1; shift ;;
    -h|--help) sed -n '2,15p' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

for dep in tmux git; do command -v "$dep" >/dev/null || { echo "missing: $dep" >&2; exit 2; }; done

SCRATCH="$(mktemp -d "${TMPDIR:-/tmp}/mk-tui-smoke.XXXXXX")"
WORK="$SCRATCH/work"          # process cwd (the .kanban/board.json cache lands here)
TASKS="$WORK/tasks"           # scratch task directory
STATE="$SCRATCH/state"        # XDG_STATE_HOME for preferences
SOCK="mk-smoke-$$"
SESS="mk"
T=(tmux -L "$SOCK")
COLS=120; ROWS=40
PASS=0; FAIL=0; GROUP=""; FAILED=()

cleanup() {
  "${T[@]}" kill-server 2>/dev/null   # private server only (-L), never the user's
  rm -f "${TMUX_TMPDIR:-/tmp}/tmux-$(id -u)/$SOCK"
  if [ "$KEEP" = 1 ]; then echo "scratch kept: $SCRATCH"; else rm -rf "$SCRATCH"; fi
}
trap cleanup EXIT

if [ -z "$BIN" ]; then
  BIN="$SCRATCH/mk"
  (cd "$ROOT" && go build -o "$BIN" .) || { echo "go build failed" >&2; exit 2; }
fi

# ---------------------------------------------------------------- fixtures --
make_fixtures() {
  rm -rf "$WORK"; mkdir -p "$WORK"
  cp -r "$ROOT/test-work-items" "$TASKS"
  cat > "$TASKS/dep_a.md" <<'MD'
---
status: TODO
project: MK
epic: QA
order: 20
tags: [alpha]
depends_on: [dep_b, nonexistent_slug]
---
# Dep A

Depends on B (a cycle) and a missing slug.
MD
  cat > "$TASKS/dep_b.md" <<'MD'
---
status: IN PROGRESS
project: MK
epic: QA
order: 20
depends_on: [dep_a]
---
# Dep B

Cycle with A.
MD
  # Fake $EDITOR: appends a marker line to the file it is given.
  cat > "$SCRATCH/fake-editor.sh" <<'SH'
#!/bin/sh
for last; do :; done
printf '\nEDITED-BY-SMOKE\n' >> "$last"
SH
  chmod +x "$SCRATCH/fake-editor.sh"
  cp "$SCRATCH/fake-editor.sh" "$SCRATCH/fake editor.sh"   # path containing a space
  git -C "$TASKS" init -q
  git -C "$TASKS" add -A
  git -C "$TASKS" -c user.email=smoke@example.com -c user.name=smoke commit -qm base
}
reset_fixtures() { git -C "$TASKS" checkout -q -- . && git -C "$TASKS" clean -fdq; }

# --------------------------------------------------------------- tmux glue --
screen() { "${T[@]}" capture-pane -t "$SESS" -p 2>/dev/null; }
# settle: wait until the screen stops changing.
settle() {
  local prev="" now i
  for i in $(seq 1 25); do
    now="$(screen)"; [ "$now" = "$prev" ] && return 0; prev="$now"; sleep 0.08
  done
}
# k KEY...: send tmux key names (Enter, Escape, Space, Tab, BTab, BSpace, C-c, ...) or single chars.
k() { local key; for key in "$@"; do "${T[@]}" send-keys -t "$SESS" "$key"; sleep 0.12; done; settle; }
# typ TEXT: send literal text (as ordinary characters).
typ() { "${T[@]}" send-keys -t "$SESS" -l "$1"; sleep 0.12; settle; }
# waitfor REGEX [SECONDS]: poll the screen for an ERE.
waitfor() {
  local re="$1" end=$((SECONDS + ${2:-4}))
  while [ $SECONDS -le $end ]; do screen | grep -Eq -- "$re" && return 0; sleep 0.1; done
  return 1
}
alive() { [ "$("${T[@]}" display-message -t "$SESS" -p '#{pane_dead}' 2>/dev/null)" = "0" ]; }

start_tui() { # [args...]; env: KEEP_STATE=1 keeps saved preferences, EDITOR_CMD overrides $EDITOR
  "${T[@]}" kill-session -t "$SESS" 2>/dev/null
  reset_fixtures
  [ "${KEEP_STATE:-0}" = 1 ] || rm -rf "$STATE"
  mkdir -p "$STATE"
  {
    echo '#!/bin/sh'
    printf 'export TERM=xterm-256color XDG_STATE_HOME=%q VISUAL=\n' "$STATE"
    printf 'export EDITOR=%q\n' "${EDITOR_CMD:-$SCRATCH/fake-editor.sh}"
    printf 'exec %q tui -dir %q "$@"\n' "$BIN" "$TASKS"
  } > "$SCRATCH/launch.sh"
  chmod +x "$SCRATCH/launch.sh"
  env -u TMUX "${T[@]}" new-session -d -s "$SESS" -x "$COLS" -y "$ROWS" -c "$WORK" "'$SCRATCH/launch.sh' $*"
  "${T[@]}" set-option -t "$SESS" remain-on-exit on >/dev/null
  "${T[@]}" resize-window -t "$SESS" -x "$COLS" -y "$ROWS" >/dev/null 2>&1
  waitfor 'TODO' 5 || true
  settle
}
stop_tui() { "${T[@]}" kill-session -t "$SESS" 2>/dev/null; }

# ------------------------------------------------------------- assertions --
ok()   { PASS=$((PASS+1)); echo "  ok    $1"; }
bad()  { FAIL=$((FAIL+1)); FAILED+=("[$GROUP] $1"); echo "  FAIL  $1"
         if [ "$VERBOSE" = 1 ]; then screen | sed 's/^/        | /'; fi; }
check() { local desc="$1"; shift; if "$@"; then ok "$desc"; else bad "$desc"; fi; }

has()      { screen | grep -Eq -- "$1"; }
hasnt()    { ! screen | grep -Eq -- "$1"; }
see()      { local d="${2:-screen shows /$1/}"; if waitfor "$1" 3; then ok "$d"; else bad "$d"; fi; }
unsee()    { local d="${2:-screen no longer shows /$1/}"; settle; if hasnt "$1"; then ok "$d"; else bad "$d"; fi; }
eq()       { if [ "$2" = "$3" ]; then ok "$1"; else bad "$1 (got '$2', want '$3')"; fi; }
fm()       { sed -n '/^---$/,/^---$/p' "$TASKS/$1" | sed -n "s/^$2: *//p" | head -1; }
status_of(){ fm "$1" status; }
clean_tree() { [ -z "$(git -C "$TASKS" status --porcelain)" ]; }
changed_files() { git -C "$TASKS" status --porcelain | awk '{print $2}' | sort | paste -sd, -; }
alive_check() { alive; }
order_of() { fm "$1" order; }
pane() { "${T[@]}" display-message -t "$SESS" -p "$1"; }
exited() { sleep 0.3; [ "$(pane '#{pane_dead}')" = 1 ]; }
tags_of() { sed -n '/^---$/,/^---$/p' "$TASKS/$1" | sed -n '/^tags:/,/^[a-z_]*:/p' | grep -E '^\s*- ' | sed 's/^ *- *//' | paste -sd, -; }
cmp_lt() { awk -v a="$1" -v b="$2" 'BEGIN{exit !(a<b)}'; }
cmp_gt() { awk -v a="$1" -v b="$2" 'BEGIN{exit !(a>b)}'; }

# selected: open the preview of the current selection, read its filename, leave with q.
selected() {
  k Enter
  local f; f="$(screen | head -10 | grep -Eo '^[A-Za-z0-9_.-]+\.md$' | head -1)"
  k q
  echo "$f"
}
# Order of filenames visible in a board column / table, top to bottom, from the screen.
col_files() { screen | grep -Eo '[a-z0-9_]+\.md' | awk '!s[$0]++'; }

# goto SECTION CARD: jump on the board to section index (CONFLICT=0 TODO=1 IN PROGRESS=2
# UNDER REVIEW=3 DONE=4 OTHER=5) and zero-based card index. Board view, no overlay open.
goto() { k h h h h h h; local i; for ((i=0;i<$1;i++)); do k l; done; k g; for ((i=0;i<$2;i++)); do k j; done; }

group() { GROUP="$1"; if [ -n "$ONLY" ] && ! [[ "$1" =~ $ONLY ]]; then return 1; fi; echo; echo "== $1"; return 0; }

# ================================================================== tests ==
make_fixtures

if group startup; then
  start_tui
  see 'CONFLICT 1' 'CONFLICT section shown first with 1 card'
  see 'TODO 5' 'TODO has 5 cards'
  see 'IN PROGRESS 2' 'IN PROGRESS has 2 cards'
  see 'UNDER REVIEW 2' 'UNDER REVIEW has 2 cards'
  check 'top bar shows board view and filters (0)' has 'board +filters \(0\)'
  check 'short help line visible at 120 cols' has 'help$'
  eq 'initial selection is the CONFLICT card' "$(selected)" resolve_merge_conflict_in_onboarding.md
  check 'working tree untouched by browsing' clean_tree
fi

if group navigation; then
  start_tui
  k l;      eq 'l -> first TODO card'              "$(selected)" test_frontend.md
  k j;      eq 'j -> next card'                    "$(selected)" read_prd.md
  k Down;   eq 'Down -> next card'                 "$(selected)" wire_up_analytics_events.md
  k k;      eq 'k -> previous card'                "$(selected)" read_prd.md
  k Up;     eq 'Up -> previous card'               "$(selected)" test_frontend.md
  k G;      eq 'G -> last card of section'         "$(selected)" dep_a.md
  k g;      eq 'g -> first card of section'        "$(selected)" test_frontend.md
  k C-d;    s="$(selected)"; check "ctrl+d moves down (now $s)" test "$s" != test_frontend.md
  k C-u;    eq 'ctrl+u returns to the top'         "$(selected)" test_frontend.md
  k Right;  eq 'Right -> IN PROGRESS'              "$(selected)" setup_backend.md
  k Left;   eq 'Left -> TODO'                      "$(selected)" test_frontend.md
  k h;      eq 'h -> CONFLICT'                     "$(selected)" resolve_merge_conflict_in_onboarding.md
  k h;      eq 'h at the first section stays put'  "$(selected)" resolve_merge_conflict_in_onboarding.md
  k l l l l l
  eq 'l x5 reaches OTHER (first card)'             "$(selected)" verify_status_after_board_rename.md
  see 'OTHER' 'OTHER header scrolled into view'
  k l;      eq 'l at the last section stays put'   "$(selected)" verify_status_after_board_rename.md
  k j;      eq 'j within OTHER'                    "$(selected)" capture_customer_feedback_notes.md
  k G;      eq 'G within OTHER'                    "$(selected)" spike_notification_service.md
  check 'navigation never writes files' clean_tree
fi

if group help; then
  start_tui
  for closer in Escape q '?' '!'; do
    k '?'; see 'Global' "? opens help (to be closed by $closer)"
    see 'quit \(leave preview\)' 'help lists global keys'
    k "$closer"; unsee 'ctrl\+c +quit; cancel grab' "$closer closes help"
  done
  k '?'; k j j; k Space; k b; k C-d; k C-u; k d; k u; k l; k h
  check 'help scrolls without crashing' alive_check
  k Escape
  k Enter; k '?'; see '^Preview' 'help in preview mode is the Preview group'; k Escape; k q
  k f;     k '?'; see 'Filter panel' 'help in filter mode is the Filter panel group'; k Escape; k Escape
  k v;     k '?'; see '^Table' 'help in table mode is the Table group'; k Escape; k v
  k l;     k Space; k '?'; see '^Grab' 'help in grab mode is the Grab group'; k Escape; k Escape
  k t;     k '?'; see 'Tag popup' 'help in tag popup is the Tag popup group'; k Escape; k Escape
  k /;     k '?'; see '^Search' 'help in search is the Search group'; k Escape; k Escape
  check 'help never writes files' clean_tree
fi

if group warnings; then
  start_tui
  k '!'; see 'arning|parse|scan|migrate_legacy|\(none\)|No ' '! opens the warnings overlay'
  k j j k; k Escape; unsee 'arnings' 'Esc closes warnings'
  k '!'; k q; k '!'; k '!'
  check 'warnings overlay closes with q and !' alive_check
fi

if group view-and-density; then
  start_tui
  k v; see 'FILE +PROJECT +EPIC +STATUS +TAGS +DEPS' 'v switches to table'
  check 'table has OTHER rows with "-" metadata' has 'spike_notification_service.md +- +- +-'
  check 'table shows dep warning marker' has 'dep_a.md .*2!'
  k j;  eq 'table j moves to next row'  "$(selected)" test_frontend.md
  k G;  eq 'table G -> last row'        "$(selected)" spike_notification_service.md
  k g;  eq 'table g -> first row'       "$(selected)" resolve_merge_conflict_in_onboarding.md
  k C-d; s="$(selected)"; check "table ctrl+d moves down ($s)" test "$s" != resolve_merge_conflict_in_onboarding.md
  k C-u
  k v; see 'board +filters' 'v returns to board'
  before="$(screen)"; k z
  check 'z toggles to full cards (screen changes)' test "$(screen)" != "$before"
  see 'resolve_merge_conflict_in_on\s*$|!!! GIT CONFLICT' 'full cards show conflict diagnostic'
  k z; eq 'z toggles back to compact' "$(screen)" "$before"
  check 'view/density never write files' clean_tree
fi

if group preferences; then
  start_tui
  k v; k z; k C-f
  k q; sleep 0.5
  check 'q exits the TUI' bash -c "[ \"\$(tmux -L $SOCK display-message -t $SESS -p '#{pane_dead}')\" = 1 ]"
  KEEP_STATE=1 start_tui
  see 'FILE +PROJECT' 'table view persisted across restart'
  check 'filter panel collapse persisted' hasnt 'PROJECTS:'
  k C-f; see 'PROJECTS:' 'ctrl+f expands the filter panel'
  k v; k z
  unset KEEP_STATE
fi

if group filters; then
  start_tui
  k f; see 'enter toggle chip' 'f focuses the filter panel'
  k Space; see 'filters \(1\)' 'Space toggles the first chip (project MK)'
  k Space; see 'filters \(0\)' 'Space toggles it back off'
  k j; k Enter; see 'filters \(1\)' 'j moves to EPICS group; Enter toggles a chip'
  k F; see 'filters \(0\)' 'F clears chips while in the panel'
  # Select tag "alpha" (first tag chip): epics -> tags group
  k j; k Space
  see 'filters \(1\)' 'tag chip toggled'
  k Escape; see 'board|table' 'Esc leaves the panel'
  see 'TODO 1/5' 'visible/total counts while filtering'
  check 'dep_a still visible' has 'dep_a.md'
  check 'read_prd hidden by the tag filter' hasnt 'read_prd.md'
  k C-f; unsee 'PROJECTS:' 'ctrl+f collapses the panel'
  check 'filter still applied while collapsed' has 'filters \(1\)'
  k C-f; see 'PROJECTS:' 'ctrl+f expands again'
  k F; see 'filters \(0\)' 'F on the board clears chips'
  see 'TODO 5' 'counts restored'
  k f; k l; k l; k Space; see 'filters \(1\)' 'l moves between chips'
  k f; k F
  check 'filters never write files' clean_tree
fi

if group search; then
  start_tui
  k /; typ 'qr!'; see 'qr!' 'q, r and ! are ordinary text in the search prompt'
  check 'TUI still alive after typing q' alive_check
  k Escape; unsee 'qr!' 'Esc clears and closes the prompt'
  k /; typ 'prd'; see 'read_prd' 'live filtering shows the match'
  unsee 'test_frontend' 'non-matching cards are hidden live'
  k Enter; see 'search: "prd"' 'Enter keeps the query in the top bar'
  k Escape; unsee 'search: "prd"' 'Esc on the board clears the kept query'
  see 'test_frontend' 'cards return'
  k /; typ 'backend'; k Enter
  k n; a="$(selected)"; k n; b="$(selected)"
  check "n advances between matches ($a -> $b)" test -n "$a" -a "$a" != "$b"
  k N; eq 'N goes back' "$(selected)" "$a"
  k Escape
  k /; typ 'Northwind'; k Enter; see 'capture_customer_feedback' 'body text matches (substring)'
  check 'body match hides non-matching cards' hasnt 'test_frontend'
  k F; unsee 'search:' 'F clears the query'
  k /; typ 'zzzznomatch'; k Enter; see 'search:' 'no-match query is kept'
  k Escape
  k v; k /; typ 'pcpy'; k Enter
  see 'draft_pricing_page_copy' 'fuzzy title match in table'
  k Escape; k v
  k /; typ 'alpha'; k C-c;
  sleep 0.3
  check 'ctrl+c in search quits the TUI (per keymap)' bash -c "[ \"\$(tmux -L $SOCK display-message -t $SESS -p '#{pane_dead}')\" = 1 ]"
  check 'search never writes files' clean_tree
fi

if group preview; then
  start_tui
  goto 1 4   # dep_a
  eq 'positioned on dep_a' "$(selected)" dep_a.md
  k Enter; see 'depends|deps: ' 'preview header shows deps'
  check 'deps listed with cycle/missing markers' has 'dep_b.*nonexistent_slug\?|nonexistent_slug\?.*dep_b'
  k Tab; k Enter; see '^dep_b.md' 'Tab selects first dependency, Enter follows it'
  k BSpace; see '^dep_a.md' 'Backspace goes back through dependency history'
  k Tab; k Tab; k Enter; see 'unknown card: nonexistent_slug' 'following a missing dep shows an error and stays'
  check 'still on dep_a' has '^dep_a.md'
  k BTab; k Enter; see '^dep_b.md' 'Shift+Tab selects previous dependency'
  k Escape; see '^dep_a.md' 'Esc steps back through history'
  k Escape; see 'board +filters' 'Esc with empty history leaves preview'
  eq 'returned to dep_a' "$(selected)" dep_a.md
  goto 1 4; k Enter; k Tab; k Enter; k q; see 'board +filters' 'q leaves the preview directly from a followed dep'
  # scrolling on a long document
  goto 4 2
   eq 'on document_smoke_test_plan' "$(selected)" document_smoke_test_plan.md
  k Enter; see 'Smoke Test Plan' 'long preview opens'
  top="$(screen)"
  k j j j; check 'j scrolls down' test "$(screen)" != "$top"
  k k k k; eq 'k scrolls back to the top' "$(screen)" "$top"
  k Down; k Up; eq 'Down/Up scroll symmetric' "$(screen)" "$top"
  k G; check 'G -> bottom' test "$(screen)" != "$top"; see 'tmp/mk-smoke' 'bottom text visible'
  k g; eq 'g -> top' "$(screen)" "$top"
  k C-d; mid="$(screen)"; check 'ctrl+d half page down' test "$mid" != "$top"
  k C-u; eq 'ctrl+u half page up' "$(screen)" "$top"
  k d; check 'd half page down' test "$(screen)" != "$top"; k u; eq 'u half page up' "$(screen)" "$top"
  k Space; check 'Space page down' test "$(screen)" != "$top"; k b; eq 'b page up' "$(screen)" "$top"
  k PageDown; check 'PageDown' test "$(screen)" != "$top"; k PageUp; eq 'PageUp' "$(screen)" "$top"
  k f; check 'f page down' test "$(screen)" != "$top"; k b
  k l l l; k h h h; check 'h/l horizontal scroll survives' alive_check
  k q
  check 'preview scrolling never writes files' clean_tree
fi


if group grab; then
  start_tui
  goto 1 0
  k Space; check 'Space shows the double-border ghost' has '╔'
  k l;     see '╔' 'l retargets to the next column'
  k Enter
  eq 'Enter drops: status updated' "$(status_of test_frontend.md)" "IN PROGRESS"
  eq 'only the moved file changed' "$(changed_files)" test_frontend.md
  eq 'selection follows the card' "$(selected)" test_frontend.md
  k u;  check 'u restores byte-identical files' clean_tree
  # cancel with Esc
  goto 1 0; k Space; k l; k Escape
  check 'Esc cancels grab (ghost gone)' hasnt '╔'
  check 'cancelled grab writes nothing' clean_tree
  # cancel with ctrl+c, second ctrl+c quits
  k Space; k l; k C-c
  check 'ctrl+c cancels grab without quitting' alive_check
  check 'ghost gone after ctrl+c' hasnt '╔'
  check 'ctrl+c cancel writes nothing' clean_tree
  # other keys ignored in grab
  k Space; k x v t e y u z f F '/' n r
  check 'grab ignores non-grab keys (still grabbing)' has '╔'
  check 'TUI alive after ignored keys' alive_check
  k q; check 'q does not quit during grab' alive_check
  check 'still grabbing after q' has '╔'
  k Escape
  check 'ignored keys wrote nothing' clean_tree
  # bottom / top / arrows
  goto 1 0; k Space; k G; k Enter
  check 'G+Enter drops at the bottom (order after dep_a)' cmp_gt "$(order_of test_frontend.md)" "$(order_of dep_a.md)"
  k u; check 'undo after bottom drop is byte-identical' clean_tree
  goto 1 3; k Space; k g; k Enter
  check 'g+Enter drops at the top (order before test_frontend)' cmp_lt "$(order_of polish_mobile_nav_transitions.md)" "$(order_of test_frontend.md)"
  k u; check 'undo after top drop is byte-identical' clean_tree
  goto 1 0; k Space; k Right Right Right; k Enter
  eq 'Right x3 + Enter moves to DONE' "$(status_of test_frontend.md)" DONE
  k u
  goto 1 0; k Space; k Right; k Left; k Down; k Up; k Enter
  check 'Right/Left/Down/Up in grab are accepted' alive_check
  k u
  check 'all grab operations undone cleanly' clean_tree
  # CONFLICT / OTHER are not configured targets
  goto 1 0; k Space; k h h h; k Enter
  eq 'h cannot target CONFLICT; stays TODO' "$(status_of test_frontend.md)" TODO
  k u
  # a second ctrl+c after cancelling quits
  goto 1 0; k Space; k C-c; k C-c
  check 'second ctrl+c quits' exited
fi

if group quick-moves; then
  start_tui
  goto 1 1   # read_prd
  k L
  eq 'L sends card to next column (IN PROGRESS)' "$(status_of read_prd.md)" "IN PROGRESS"
  check 'L lands at the real bottom' cmp_gt "$(order_of read_prd.md)" "$(order_of dep_b.md)"
  k H
  eq 'H sends card back to previous column (TODO)' "$(status_of read_prd.md)" TODO
  check 'H lands at the real bottom' cmp_gt "$(order_of read_prd.md)" "$(order_of dep_a.md)"
  k u u; check 'two undos restore byte-identical files' clean_tree
  goto 1 0   # test_frontend (order 1), next visible neighbour read_prd (1.5)
  k H;  check 'H from the first configured column is a no-op' clean_tree
  k J
  check 'J reorders below the neighbour' cmp_gt "$(order_of test_frontend.md)" "$(order_of read_prd.md)"
  check 'J stays above the following card' cmp_lt "$(order_of test_frontend.md)" "$(order_of wire_up_analytics_events.md)"
  eq 'selection follows the reordered card' "$(selected)" test_frontend.md
  k K
  check 'K reorders back above the neighbour' cmp_lt "$(order_of test_frontend.md)" "$(order_of read_prd.md)"
  k u u; check 'undo restores' clean_tree
  k K;  check 'K at the top is a no-op' clean_tree
  goto 1 4; k J; check 'J at the bottom is a no-op' clean_tree
  goto 4 0; k L; check 'L from the last configured column is a no-op' clean_tree
  # guarded cards
  goto 0 0
  k Space; check 'CONFLICT card cannot be grabbed' hasnt '╔'
  k L H J K; check 'CONFLICT card cannot be quick-moved' clean_tree
  k t; check 'CONFLICT card cannot be tagged (no popup)' hasnt 'Tab|tag input'
  k Escape
  check 'conflict guards wrote nothing' clean_tree
  goto 5 3   # migrate_legacy_priority_field.md: parse error
  eq 'positioned on the parse-error card' "$(selected)" migrate_legacy_priority_field.md
  k Space; check 'parse-error card cannot be grabbed' hasnt '╔'
  k L H; k t; k Escape
  check 'parse-error card cannot be moved/tagged' clean_tree
  k Enter; see 'press e|fix the YAML|external editor' 'parse-error preview shows the fix-in-editor banner'
  k q
  # valid OTHER cards can move
  goto 5 0   # verify_status_after_board_rename.md (status: REVIEW)
  eq 'positioned on the REVIEW card' "$(selected)" verify_status_after_board_rename.md
  k Space; k Right; k Enter
  s="$(status_of verify_status_after_board_rename.md)"
  check "valid OTHER card moves to a configured column (status now '$s')" test "$s" != REVIEW
  k u; check 'undo restores the OTHER card' clean_tree
  # frontmatter-less card: gets frontmatter, body kept, undo restores exact bytes
  goto 5 1
  eq 'positioned on the frontmatter-less card' "$(selected)" capture_customer_feedback_notes.md
  k Space; k Left; k Enter
  check 'moved card gained a status' test -n "$(status_of capture_customer_feedback_notes.md)"
  check 'body preserved after the move' grep -q 'Northwind' "$TASKS/capture_customer_feedback_notes.md"
  k u; check 'undo restores the frontmatter-less file byte-for-byte' clean_tree
fi

if group tags; then
  start_tui
  goto 1 0
  before_tags="$(tags_of test_frontend.md)"
  k t; see 'Tab|tag' 't opens the tag popup'
  typ 'Foo, bar ,'
  k Enter
  k Escape
  check 'popup closed on Esc' hasnt 'add comma'
  t="$(tags_of test_frontend.md)"
  case ",$t," in *,foo,*,bar,*|*,bar,*,foo,*|*,foo,*) r=0 ;; *) r=1 ;; esac
  check "Esc saves added tags, lower-cased ('$t')" test $r = 0
  check 'tags kept existing ones' bash -c "[[ ',$t,' == *,frontend,* ]]"
  check 'no uppercase tag written' bash -c "! [[ '$t' == *Foo* ]]"
  eq 'only the tagged file changed' "$(changed_files)" test_frontend.md
  # duplicate
  k t; typ 'foo'; k Enter; see 'already tagged' 'duplicate tag is skipped with a notice'
  k Escape
  # remove via Tab, j, d / x / Delete
  k t; k Tab
  n0="$(tags_of test_frontend.md | tr ',' '\n' | wc -l)"
  k d; k Escape
  n1="$(tags_of test_frontend.md | tr ',' '\n' | wc -l)"
  check "d removes the selected tag ($n0 -> $n1)" test "$n1" -lt "$n0"
  k t; k Tab; k j; k x; k Escape
  n2="$(tags_of test_frontend.md | tr ',' '\n' | wc -l)"
  check "x removes a tag ($n1 -> $n2)" test "$n2" -lt "$n1"
  k t; k Tab; k DC; k Escape
  n3="$(tags_of test_frontend.md | tr ',' '\n' | wc -l)"
  check "Delete removes a tag ($n2 -> $n3)" test "$n3" -lt "$n2"
  # Tab/BTab switch focus
  k t; k Tab; k BTab; typ 'viaBTab'; k Enter; k Escape
  check 'Shift+Tab returns to the input' bash -c "[[ ',$(tags_of test_frontend.md),' == *,viabtab,* ]]"
  # ctrl+c discards
  k t; typ 'discard-me'; k Enter; k C-c
  check 'ctrl+c in popup does not quit' alive_check
  check 'ctrl+c discards staged tags' bash -c "! grep -q discard-me '$TASKS/test_frontend.md'"
  # empty input is a no-op
  cp "$TASKS/test_frontend.md" "$SCRATCH/snap.md"
  k t; k Enter; k Escape
  check 'empty Enter is a no-op' cmp -s "$TASKS/test_frontend.md" "$SCRATCH/snap.md"
  # j/k in input are ordinary text
  k t; typ 'jk q r !'; k Enter; k Escape
  check 'q/r/j/k/! are ordinary text in the tag input' bash -c "[[ ',$(tags_of test_frontend.md),' == *,'jk q r !',* ]]"
  # external change while the popup is open cancels the save
  k t; typ 'ext-tag'; k Enter
  printf '\nEXTERNAL\n' >> "$TASKS/test_frontend.md"; sleep 0.6
  k Escape
  check 'external edit during popup: tag not written' bash -c "! grep -q ext-tag '$TASKS/test_frontend.md'"
  check 'external edit during popup: external change kept' grep -q EXTERNAL "$TASKS/test_frontend.md"
  # tag from preview and from table
  git -C "$TASKS" checkout -q -- .
  k r; sleep 0.5
  goto 1 1; k Enter; k t; typ 'frompreview'; k Enter; k Escape; k q
  check 't works from preview' bash -c "[[ ',$(tags_of read_prd.md),' == *,frompreview,* ]]"
  k v; k j; k j; k t; typ 'fromtable'; k Enter; k Escape
  check 't works from the table' bash -c "grep -rqs fromtable '$TASKS'"
  k v
  # undo walks back every tag action
  for i in 1 2 3 4 5 6 7 8 9 10; do k u; done
  git -C "$TASKS" checkout -q -- . 2>/dev/null
fi

if group undo; then
  start_tui
  goto 1 0
  k L;  k t; typ 'undo-tag'; k Enter; k Escape; goto 1 1; k J
  check 'three actions applied' test -n "$(changed_files)"
  # the toast replaces the status line for a moment; wait for the "u undo (N)" hint
  if waitfor 'u undo \([0-9]+\)' 5; then ok 'status bar shows undo depth'; else bad 'status bar shows undo depth'; fi
  k u u u
  check 'three undos -> byte-identical tree' clean_tree
  k u; check 'u on empty stack does not crash' alive_check
  # guarded: refuses after an external edit
  goto 1 1; k L
  printf '\nEXTERNAL-EDIT\n' >> "$TASKS/read_prd.md"; sleep 0.6
  k u
  check 'undo refuses after external edit (edit retained)' grep -q EXTERNAL-EDIT "$TASKS/read_prd.md"
  eq 'refused undo keeps the moved status' "$(status_of read_prd.md)" "IN PROGRESS"
  if waitfor "can't undo: read_prd.md changed since" 2; then ok 'refusal reported on screen'; else bad 'refusal reported on screen'; fi
  k u; check 'discarded entry: second u is a no-op' alive_check
  # undo from the table
  git -C "$TASKS" checkout -q -- .; k r; sleep 0.5
  goto 1 0; k L; k v; k u; k v
  check 'u works from the table view' clean_tree
  # u in preview is not an undo (keymap: board/table only)
  goto 1 0; k L; k Enter; k u; k q
  check 'u in preview does not undo' test -n "$(changed_files)"
  k u
  check 'undo from board afterwards restores' clean_tree
fi

if group copy-path; then
  start_tui
  goto 1 0; k y
  check 'y on board does not crash' alive_check
  if waitfor '[Cc]op|[Cc]lipboard|path' 3; then ok 'y reports something (copied / OSC 52)'; else bad 'y reports something (copied / OSC 52)'; fi
  k Enter; k y; check 'y in preview does not crash' alive_check; k q
  k v; k y; check 'y in table does not crash' alive_check; k v
  check 'y never writes files' clean_tree
  # tmux buffer receives OSC 52 only if set-clipboard is on; report, do not fail.
  buf="$("${T[@]}" show-buffer 2>/dev/null | head -1)"
  echo "  info  tmux buffer after y: '${buf:-<empty>}'"
fi

if group editor; then
  start_tui
  goto 1 0; k e
  waitfor 'board +filters' 3
  check 'editor ran (file modified)' grep -q EDITED-BY-SMOKE "$TASKS/test_frontend.md"
  check 'TUI is back on the board and alive' alive_check
  check 'terminal restored: alternate screen active again' test "$(pane '#{alternate_on}')" = 1
  eq 'only the edited file changed' "$(changed_files)" test_frontend.md
  k Enter; see 'EDITED-BY-SMOKE' 'preview shows the editor change (rescan on return)'
  k e; sleep 0.8
  check 'e works from preview' test "$(grep -c EDITED-BY-SMOKE "$TASKS/test_frontend.md")" = 2
  see 'EDITED-BY-SMOKE' 'preview refreshed'
  k q
  k u; check 'editor changes are not undoable (file keeps edits)' grep -q EDITED-BY-SMOKE "$TASKS/test_frontend.md"
  k v; k e; sleep 0.8; check 'e works from the table' test "$(grep -c EDITED-BY-SMOKE "$TASKS/test_frontend.md")" = 3
  k v
  # parse-error card can be fixed with e
  goto 5 3
  k e; sleep 0.6; check 'e is allowed on a parse-error card' grep -q EDITED-BY-SMOKE "$TASKS/migrate_legacy_priority_field.md"
  # quoted path containing a space + argument
  EDITOR_CMD="\"$SCRATCH/fake editor.sh\" --wait" start_tui
  goto 1 0; k e; sleep 0.8
  check 'quoted editor path with spaces and args works' grep -q EDITED-BY-SMOKE "$TASKS/test_frontend.md"
  # failing editor: TUI survives and rescans
  EDITOR_CMD=false start_tui
  goto 1 0; k e; sleep 0.8
  check 'failing editor: TUI stays alive' alive_check
  check 'failing editor writes nothing' clean_tree
  see 'board +filters' 'failing editor: back on the board'
  unset EDITOR_CMD
fi

if group reload; then
  start_tui
  goto 1 1   # read_prd
  # new file appears
  printf -- '---\nstatus: TODO\n---\n# Brand new\n' > "$TASKS/brand_new.md"
  see 'TODO 6' 'new external file appears (TODO 5 -> 6)'
  eq 'selection kept by filename on reload' "$(selected)" read_prd.md
  # selected task moves away: stays in section, falls back to position
  sed -i 's/^status: TODO/status: DONE/' "$TASKS/read_prd.md"
  see 'TODO 5' 'externally moved card leaves TODO'
  s="$(selected)"; check "selection falls back within section ($s)" test -n "$s" -a "$s" != read_prd.md
  # remove a file
  rm "$TASKS/brand_new.md"; see 'TODO 4' 'removed file disappears'
  # column config
  printf '  - BLOCKED\n' >> "$TASKS/.kanban.yml"
  sleep 1; k l l l l l l l l   # the board shows a window of sections; walk to the far right
  see 'BLOCKED' 'new column in .kanban.yml appears live'
  git -C "$TASKS" checkout -q -- .kanban.yml
  unsee 'BLOCKED' 'removing the column removes it live'
  # manual rescan
  k r; check 'r rescans without crashing' alive_check
  # malformed frontmatter -> OTHER + warning
  printf -- '---\nstatus: [oops\n---\nbroken\n' > "$TASKS/broken_new.md"
  sleep 0.8; k '!'
  see 'broken_new' '! lists the new parse warning'
  k Escape
  rm "$TASKS/broken_new.md"
  # open preview, then change, then remove its file
  start_tui
  goto 1 1; k Enter
  printf '\nLIVE-APPENDED-LINE\n' >> "$TASKS/read_prd.md"
  see 'LIVE-APPENDED-LINE' 'open preview refreshes on external change'
  rm "$TASKS/read_prd.md"
  see 'file removed' 'removed open task shows the file removed banner'
  see 'LIVE-APPENDED-LINE' 'last content retained under the banner'
  k q; see 'board +filters' 'q leaves the stale preview'
fi

if group resize; then
  start_tui
  "${T[@]}" resize-window -t "$SESS" -x 80 -y 24 >/dev/null; settle; sleep 0.3
  check '80x24 renders section headers' has 'TODO'
  check '80x24 shows an edge marker for hidden sections' has '[‹›]'
  if waitfor '\? help' 5; then ok '80x24 keeps the short help line (once the warning toast clears)'; else bad '80x24 keeps the short help line'; fi
  k l l l l; see 'DONE' 'window follows the focused section at 80x24'
  eq 'navigation works at 80x24' "$(selected)" "$(selected)"
  "${T[@]}" resize-window -t "$SESS" -x 50 -y 20 >/dev/null; settle; sleep 0.3
  check '50x20 survives' alive_check
  check '50x20 drops the help line (<60 cols)' hasnt 'help$'
  k v; k z; k Enter; k q; k v
  "${T[@]}" resize-window -t "$SESS" -x 200 -y 60 >/dev/null; settle; sleep 0.3
  check '200x60 survives and shows all columns' has 'UNDER REVIEW'
  "${T[@]}" resize-window -t "$SESS" -x "$COLS" -y "$ROWS" >/dev/null
  check 'resizing never writes files' clean_tree
fi

if group quit; then
  start_tui; k q
  check 'q on the board quits' exited
  eq 'exit status is 0' "$(pane '#{pane_dead_status}')" 0
  eq 'alternate screen left (terminal restored)' "$(pane '#{alternate_on}')" 0
  start_tui; k C-c
  check 'ctrl+c on the board quits' exited
  eq 'ctrl+c exit status is 0' "$(pane '#{pane_dead_status}')" 0
  start_tui; goto 1 0; k Enter; k q
  check 'q in preview only leaves the preview' alive_check
  k Enter; k Escape
  check 'Esc in preview does not quit' alive_check
  start_tui; goto 1 0; k t; k C-c
  check 'ctrl+c in the tag popup cancels without quitting' alive_check
  k C-c; check 'second ctrl+c quits' exited
  start_tui; k '?'; k q; check 'q in help only closes help' alive_check
  start_tui; k f; k q; check 'q in filter panel quits' exited
fi

if group cli; then
  out="$("$BIN" tui -version 2>&1)"; rc=$?
  check "tui -version prints build info ($out)" bash -c "[[ '$out' == mk\ * ]] && [ $rc = 0 ]"
  out="$("$BIN" tui bogus 2>&1)"; rc=$?
  check 'tui with a stray argument fails' test "$rc" != 0
  out="$("$BIN" tui -port 1 2>&1)"; rc=$?
  check 'tui rejects -port' test "$rc" != 0
  start_tui
  eq 'mouse capture is on by default' "$(pane '#{mouse_any_flag}')" 1
  stop_tui
  start_tui -no-mouse
  eq '-no-mouse disables mouse capture' "$(pane '#{mouse_any_flag}')" 0
  k l j; eq 'keyboard works with -no-mouse' "$(selected)" read_prd.md
fi

# ================================================================ summary ==
echo
echo "passed: $PASS  failed: $FAIL"
if [ "$FAIL" -gt 0 ]; then printf 'FAILED: %s\n' "${FAILED[@]}"; exit 1; fi
exit 0
