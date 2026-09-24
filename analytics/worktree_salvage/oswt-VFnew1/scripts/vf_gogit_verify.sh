#!/usr/bin/env bash
# vf_gogit_verify.sh UNIT [UNIT...] — host-side verify of hidden suites.
# For each unit: copy hidden tests into excised/gold/cheat trees at their
# relpaths, then run `go test -run TestDetail`:
#   excised must FAIL, cheat must FAIL, gold must PASS under 5 seeds.
set -uo pipefail
WORK=/home/evan/Documents/open_swe_traces_research/experiments/pipeline/work
AU=/home/evan/Documents/oswt-AUnew1/experiments/pipeline/authored_batch2/go-git
export GOTOOLCHAIN=auto

install_hidden() { # $1 = tree, $2 = unit
  find "$WORK/vf_hidden_gogit/$2" -name '*_test.go' | while read -r f; do
    rel="${f#$WORK/vf_hidden_gogit/$2/}"
    mkdir -p "$1/$(dirname "$rel")"
    cp "$f" "$1/$rel"
  done
}

pkg_args() { # $1 = unit -> unique parent dirs of hidden tests as ./pkg/ args
  find "$WORK/vf_hidden_gogit/$1" -name '*_test.go' -printf '%h\n' \
    | sed "s|$WORK/vf_hidden_gogit/$1||" | sort -u | sed 's|^|.|;s|$|/|'
}

for unit in "$@"; do
  hid="$WORK/vf_hidden_gogit/$unit"
  if [ ! -d "$hid" ]; then echo "!! $unit: no hidden dir"; continue; fi
  for kind in excised gold cheat; do
    dest="$WORK/vf_${kind}_gogit/$unit"
    if [ "$kind" != excised ] && [ ! -d "$dest" ]; then
      rm -rf "$dest"; cp -a "$WORK/vf_excised_gogit/$unit" "$dest"
      patch -p1 --forward --batch -s -i "$AU/$unit/_author/$kind.patch" -d "$dest" \
        || { echo "!! $unit: $kind.patch failed"; continue; }
    fi
    install_hidden "$dest" "$unit"
  done
  pkgs=$(pkg_args "$unit")
  cd "$WORK/vf_excised_gogit/$unit"
  bare=$(go test -count=1 -run 'TestDetail' $pkgs 2>&1 | tail -3)
  echo "$bare" | grep -qE "panic:|FAIL" && barev=FAIL || barev="?PASS/INFRA"
  cd "$WORK/vf_cheat_gogit/$unit"
  cheat=$(go test -count=1 -run 'TestDetail' $pkgs 2>&1 | tail -3)
  echo "$cheat" | grep -qE "panic:|FAIL" && cheatv=FAIL || cheatv="?PASS"
  cd "$WORK/vf_gold_gogit/$unit"
  goldv=PASS
  for seed in "" 20260920 7301989 1 42; do
    if [ -z "$seed" ]; then
      go test -count=1 -run 'TestDetail' $pkgs >/dev/null 2>&1 || { goldv="FAIL(seed=$seed)"; break; }
    else
      HIDDEN_SEED=$seed go test -count=1 -run 'TestDetail' $pkgs >/dev/null 2>&1 || { goldv="FAIL(seed=$seed)"; break; }
    fi
  done
  echo "$unit: bare=$barev gold=$goldv cheat=$cheatv"
done
