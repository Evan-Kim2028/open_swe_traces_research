#!/usr/bin/env bash
# grok-batch-review.sh [hours] -- ONE grok-4.6 pass over all PRs in a window.
#
# Payload discipline: grok's -p takes the prompt as an argv value, so dumping
# full diffs overflows ARG_MAX (14 PRs x 400 lines did). That limit forced a
# better design anyway -- we compute the file-overlap matrix here and send raw
# diff ONLY for files more than one PR touched. That shared surface is where
# cross-PR interactions live; the rest is noise a per-PR review already covers.
set -u
HOURS="${1:-6}"; REPO="${2:-Evan-Kim2028/lake-of-rage}"
D=/home/evan/devin-tasks
OUT="$D/grok-batch-$(date +%Y%m%dT%H%M%S).md"
SINCE=$(( $(date +%s) - HOURS*3600 ))

nums=""
while read -r n at; do
  [ -z "$n" ] && continue
  e=$(date -d "$at" +%s 2>/dev/null) || continue
  [ "$e" -ge "$SINCE" ] && nums="$nums $n"
done < <(gh pr list -R "$REPO" --state merged --limit 40 --json number,mergedAt -q '.[]|"\(.number) \(.mergedAt)"')
[ -z "$nums" ] && { echo "no PRs in last ${HOURS}h"; exit 0; }
echo "batch:$nums"

MAP=$(mktemp); : > "$MAP"
for n in $nums; do
  gh pr view "$n" -R "$REPO" --json files -q '.files[].path' 2>/dev/null | while read -r f; do echo "$f $n"; done >> "$MAP"
done
SHARED=$(awk '{a[$1]=a[$1]" "$2} END{for(f in a){c=split(a[f],x," "); if(c>1) print f}}' "$MAP" | sort)

P=$(mktemp)
{
  echo "BATCH REVIEW: ${HOURS}h of merged PRs in $REPO. Checkout: /home/evan/lake-of-rage"
  echo
  echo "This is a CLUSTER review. Per-PR review already happened. Your value is"
  echo "ONLY in what a single-PR reviewer cannot see. Rank findings by severity;"
  echo "skip praise; cite file:line."
  echo
  echo "## PRs"
  for n in $nums; do
    echo "- #$n $(gh pr view "$n" -R "$REPO" --json title -q .title 2>/dev/null)"
  done
  echo
  echo "## Files touched by MORE THAN ONE PR (the interaction surface)"
  if [ -n "$SHARED" ]; then
    for f in $SHARED; do echo "- $f  <-  $(grep -E "^$f " "$MAP" | awk '{printf "#%s ", $2}')"; done
  else
    echo "(none — but interactions still travel via shared macros, config,"
    echo " a model whose output another PR asserts on, or a guard one PR adds"
    echo " that another PR's change would now trip)"
  fi
  echo
  if [ -n "$SHARED" ]; then
    echo "## Diffs for shared files only"
    for f in $SHARED; do
      for n in $(grep -E "^$f " "$MAP" | awk '{print $2}'); do
        echo "### #$n :: $f"
        gh pr diff "$n" -R "$REPO" 2>/dev/null | awk -v want="$f" '
          /^diff --git /{cur=$0; on=(index(cur, want)>0)} on' | head -120
      done
    done
  fi
  echo
  echo "## Answer these"
  echo "1. For each shared file: can the changes combine into a broken state?"
  echo "   State the matrix verdict even where it is clean."
  echo "2. Systemic: is one bug being fixed repeatedly in different places"
  echo "   (a missing shared abstraction)? Do any two PRs disagree about a"
  echo "   contract? Is one PR's guard weaker than another's for the same"
  echo "   invariant?"
  echo "3. Guard quality: would each new assertion actually fail if its bug"
  echo "   returned? Flag self-confirming tests. Flag any assertion on an"
  echo "   aggregate whose meaning shifts when the population beneath it"
  echo "   changes — this repo has shipped several (row-count floors, pooled"
  echo "   p95, global null-share bands)."
  echo "4. Environment-dependence: tests here have silently read production"
  echo "   data from \$HOME/data and silently skipped on missing venv/import."
  echo "   Flag anything that can pass by not running."
  echo "5. The single highest-value follow-up."
  echo
  echo "Standing rules: never raise memory/CPU/timeout limits as a fix;"
  echo "bounded partial progress exits 0; CI is billing-blocked so local gates"
  echo "are authoritative."
} > "$P"

SZ=$(wc -c < "$P")
echo "prompt bytes: $SZ"
if [ "$SZ" -gt 120000 ]; then head -c 120000 "$P" > "$P.cut"; mv "$P.cut" "$P"; echo "truncated to 120k"; fi
grok -m grok-4.6 --effort high -p "$(cat "$P")" > "$OUT" 2>&1
rm -f "$P" "$MAP"
echo "written: $OUT"
