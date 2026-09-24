#!/bin/bash
# verify_unit.sh <unit_dir> — apply excision+gold+cheat on fresh pristine copies,
# compile each, and check gold restores byte-exactly.
set -uo pipefail
UNIT="$1"
AU="$UNIT/_author"
GOLD_SRC=/home/evan/Documents/oswt-AU4bbolt2/outputs/scratch/bbolt_gold
export GOFLAGS=-mod=mod GOPROXY=off PATH=/home/evan/go/bin:$PATH
pkgs=$(grep -E "^\+\+\+ b/(.*\.go)" "$AU/gold.patch" | sed -E 's|^\+\+\+ b/||; s|/[^/]+\.go$||; s|^[^.]+\.go$|.|' | sort -u | sed 's|^[^.]|./&|')
fail=0
for mode in excised gold cheat; do
  t=$(mktemp -d)
  cp -a "$GOLD_SRC/." "$t/"
  patch -p1 -s -d "$t" < "$AU/excised/excision.patch" || { echo "excision.patch FAILS to apply"; fail=1; }
  case $mode in
    gold)  patch -p1 -s -d "$t" < "$AU/gold.patch" ;;
    cheat) patch -p1 -s -d "$t" < "$AU/cheat.patch" ;;
  esac
  if (cd "$t" && go build $pkgs 2>/tmp/vu_build.err); then
    echo "$mode build OK"
  else
    echo "$mode build FAIL:"; head -8 /tmp/vu_build.err; fail=1
  fi
  if [ "$mode" = gold ]; then
    for f in $(grep -oE "^\+\+\+ b/\S+" "$AU/gold.patch" | sed 's|+++ b/||'); do
      diff -q "$t/$f" "$GOLD_SRC/$f" >/dev/null || { echo "gold mismatch: $f"; fail=1; }
    done
  fi
  rm -rf "$t"
done
exit $fail
