#!/bin/bash
# Fast in-image check for one go-github unit's hidden suite.
# usage: vf_check.sh <unit> [bare|gold|cheat]
# Mounts the materialized excised tree + hidden tests + author patches into
# ladder-base:go-github, runs `go test -run 'TestDetail' ./github` offline.
set -uo pipefail
unit="${1:?unit}"; mode="${2:-bare}"
W=/home/evan/Documents/open_swe_traces_research/experiments/pipeline/work/go-github/units/$unit
AU=/home/evan/Documents/oswt-AUgogithub/experiments/pipeline/authored_batch2/go-github/$unit/_author
docker run --rm --network=none -e HIDDEN_SEED \
  -v "$W/tree:/src:ro" \
  -v "$W/_verifier/tests/hidden:/hidden:ro" \
  -v "$AU/gold.patch:/patches/gold.patch:ro" \
  -v "$AU/cheat.patch:/patches/cheat.patch:ro" \
  ladder-base:go-github bash -c '
    set -uo pipefail
    rm -rf /work && cp -a /src /work && cd /work
    cp /hidden/github/*_test.go github/
    patch_arg="'"$mode"'"
    if [ "$patch_arg" = gold ] || [ "$patch_arg" = cheat ]; then
      patch -p1 --forward --batch -i /patches/$patch_arg.patch || { echo PATCH_FAILED; exit 70; }
    fi
    go test -count=1 -timeout 5m -run "TestDetail" ./github 2>&1 | tail -30
    echo "EXIT_MARKER:${PIPESTATUS[0]}"
  '
