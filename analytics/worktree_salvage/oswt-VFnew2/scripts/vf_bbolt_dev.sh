#!/bin/bash
# Dev loop for the bbolt hidden suites: run a unit's tests/hidden files inside
# ladder-base:bbolt against a staged tree.
#   usage: vf_bbolt_dev.sh <unit> [gold|excised|cheat] [seed] [extra go test args]
set -uo pipefail
unit="$1"; kind="${2:-gold}"; seed="${3:-20260919}"; shift $(( $# < 3 ? $# : 3 ))
WORK=/home/evan/Documents/open_swe_traces_research/experiments/pipeline/work
case "$unit" in
  bucket|cursor|tx|db|node|txcheck|compact) pkg="." ;;
  meta|page|inode|inbucket|loadutil|verifyenv) pkg="./internal/common" ;;
  flarray|flhashmap|flshared) pkg="./internal/freelist" ;;
  surgeon|xray) pkg="./internal/surgeon" ;;
  gutscli) pkg="./internal/guts_cli" ;;
  cmdutils|cmdget|cmdpage|cmddump|cmdpages|cmdsurgerymeta) pkg="./cmd/bbolt/command" ;;
  *) echo "unknown unit $unit" >&2; exit 2 ;;
esac
tree="$WORK/vf_bbolt_${kind}/$unit"
hidden="$WORK/vf_bbolt_hidden/$unit"
docker run --rm --network=none \
  -v "$tree:/app" -v "$hidden:/hidden" -w /app \
  -e "HIDDEN_SEED=$seed" \
  ladder-base:bbolt bash -c "
    cp -r /hidden/. /app/ && \
    go test -v -count=1 -timeout 10m -run '^TestDetail' $pkg $*
  "
