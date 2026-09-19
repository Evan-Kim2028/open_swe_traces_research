#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"
HIDDEN="$TESTS_DIR/hidden"
install_hidden() {
  rel="$1"
  dest="/app/$rel"
  mkdir -p "$(dirname "$dest")"
  cp "$HIDDEN/$rel" "$dest"
}
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "f8313b437727151d3eeb15cc34b3b69eb71fc3909e088b71d79df8a3b9b503e1  $HIDDEN/pkg/chart/common/util/coalesce_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/pkg/chart/common/util/coalesce_bb_prop_test.go"
install_hidden "pkg/chart/common/util/coalesce_bb_prop_test.go"
echo "f8313b437727151d3eeb15cc34b3b69eb71fc3909e088b71d79df8a3b9b503e1  /app/pkg/chart/common/util/coalesce_bb_prop_test.go" | sha256sum -c --status || checksum_fail "pkg/chart/common/util/coalesce_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestCoalesceContractTableProperty|TestCoalesceTablesProperty|TestMergeTablesProperty|TestCoalesceValuesProperty|TestMergeValuesProperty|TestCoalesceAdversarialProperty|TestCoalesceUnseenRandomProperty)$' ./pkg/chart/common/util/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
