#!/usr/bin/env bash
# author_unit.sh — mechanical helper for authoring pipeline units.
#
# Operates on a repo checkout pair:
#   $WORK/orig     pristine tree (never modified)
#   $WORK/scratch  git repo whose index mirrors orig; worktree is edited
#
# Subcommands:
#   reset <file>...            restore files from orig and reindex scratch
#   excision <unit>            write worktree-vs-index diff to <unit>/_author/excised/excision.patch
#   stagestub <file>...        git add files so the index records the stub state
#   impl <unit> <name> <file>  write diff(stub-state, worktree) for <file> to
#                              <unit>/_author/<name>.patch  (name: gold|cheat)
#
# Usage: WORK=experiments/pipeline/work/go-github scripts/author_unit.sh reset github/x.go
set -euo pipefail

WORK="${WORK:?set WORK to the work dir containing orig/ and scratch/}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ORIG="$WORK/orig"
SCRATCH="$WORK/scratch"
UNITS="${UNITS:-experiments/pipeline/authored_batch4/go-github}"

cmd="${1:?subcommand required}"; shift

case "$cmd" in
  reset)
    for f in "$@"; do cp "$ORIG/$f" "$SCRATCH/$f"; done
    git -C "$SCRATCH" add -A
    ;;
  excision)
    unit="$1"; mkdir -p "$ROOT/$UNITS/$unit/_author/excised"
    git -C "$SCRATCH" diff > "$ROOT/$UNITS/$unit/_author/excised/excision.patch"
    ;;
  stagestub)
    for f in "$@"; do git -C "$SCRATCH" add -- "$f"; done
    ;;
  impl)
    unit="$1"; name="$2"; file="$3"
    mkdir -p "$ROOT/$UNITS/$unit/_author"
    git -C "$SCRATCH" diff -- "$file" > "$ROOT/$UNITS/$unit/_author/$name.patch"
    ;;
  modified)
    git -C "$SCRATCH" status --porcelain | awk '{print $2}'
    ;;
  *) echo "unknown subcommand: $cmd" >&2; exit 1 ;;
esac
