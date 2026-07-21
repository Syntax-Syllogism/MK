#!/usr/bin/env bash
# Reset the disposable QA fixture copies in docs/ from the pristine fixtures in
# test-docs/. Do not edit test-docs/ unless changing or adding a test scenario.
#
# Usage: scripts/reset-smoke-fixtures.sh [docs-dir]

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
src_dir="$repo_root/test-docs"
dest_dir="${1:-$repo_root/docs}"

if [ ! -d "$src_dir" ]; then
  echo "error: $src_dir does not exist" >&2
  exit 1
fi

mkdir -p "$dest_dir"

count=0
for fixture in "$src_dir"/*.md; do
  [ -e "$fixture" ] || continue
  cp "$fixture" "$dest_dir/"
  echo "reset: $(basename "$fixture")"
  count=$((count + 1))
done

echo
echo "Reset $count fixture(s) in $dest_dir from $src_dir."
echo "test-docs/ remains the pristine source of truth."
