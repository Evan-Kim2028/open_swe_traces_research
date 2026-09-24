#!/bin/bash
# make_cheat.sh <unit_dir> — build cheat.patch from _author/cheat/<relpath> files.
# Stages an excised tree, drops each cheat file in, compiles the touched
# packages, then emits a unified diff excised->cheat into _author/cheat.patch.
set -uo pipefail
UNIT="$1"
AU="$UNIT/_author"
GOLD_SRC=/home/evan/Documents/oswt-AU4bbolt2/outputs/scratch/bbolt_gold
export GOFLAGS=-mod=mod GOPROXY=off PATH=/home/evan/go/bin:$PATH
CHEAT_DIR="$AU/cheat"
[ -d "$CHEAT_DIR" ] || { echo "no cheat dir"; exit 1; }

t=$(mktemp -d); ref=$(mktemp -d)
cp -a "$GOLD_SRC/." "$t/"
patch -p1 -s -d "$t" < "$AU/excised/excision.patch" || { echo "excision apply FAIL"; exit 1; }

files=$(cd "$CHEAT_DIR" && find . -type f -name "*.go" | sed 's|^\./||')
pkgs=""
patch_out="$AU/cheat.patch"
: > "$patch_out"
for f in $files; do
  mkdir -p "$ref/$(dirname "$f")"
  cp "$t/$f" "$ref/$f"                      # excised baseline
  cp "$CHEAT_DIR/$f" "$t/$f"                # cheat in place
  pkgs="$pkgs ./$(dirname "$f")"
  git diff --no-index "$ref/$f" "$t/$f" \
    | sed -e "s|a/.*/$f|a/$f|" -e "s|b/.*/$f|b/$f|" \
    | sed -e "s|a$ref/$f|a/$f|" -e "s|b$t/$f|b/$f|" \
    >> "$patch_out"
done
# normalize header paths (git --no-index emits absolute)
sed -i -e "s|--- $ref/|--- a/|g" -e "s|+++ $t/|+++ b/|g" "$patch_out"
pkgs=$(echo "$pkgs" | tr ' ' '\n' | sort -u)
(cd "$t" && go build $pkgs) 2>/tmp/mc_build.err
rc=$?
if [ $rc -eq 0 ]; then echo "cheat build OK"; else echo "cheat build FAIL"; head -8 /tmp/mc_build.err; fi
grep -c "panic(\"excised" $t/$(echo "$files" | head -1) 2>/dev/null | sed 's/^/stubs left in first cheat file: /'
rm -rf "$t" "$ref"
exit $rc
