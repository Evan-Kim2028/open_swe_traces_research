#!/usr/bin/env bash
# queue-brief.sh <brief.md> <c|m>  — queue a brief with _constraints.md PREPENDED.
#
# The runners feed the brief file to the agent verbatim. Briefs used to merely
# *reference* _constraints.md ("read it before you start"), which agents
# routinely did not do — so standing rules (ignore CI billing and --admin
# merge; never self-merge a storage-format change) reached them only when a
# brief happened to restate them. Prepending makes that unconditional.
set -eu
D=/home/evan/devin-tasks
B="$1"; LANE="${2:-c}"
[ -s "$B" ] || { echo "brief missing or empty: $B" >&2; exit 1; }
case "$LANE" in
  c) Q="$D/cqueue" ;;
  m) Q="$D/mqueue" ;;
  *) echo "lane must be c (cursor) or m (deepseek)" >&2; exit 1 ;;
esac
name="$(basename "$B")"
{
  echo "# Standing constraints (read these first — they override brief wording)"
  echo
  cat "$D/_constraints.md"
  echo
  echo "---"
  echo
  cat "$B"
} > "$Q/$name"
echo "queued $name -> $Q ($(wc -l < "$Q/$name") lines, constraints prepended)"
