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
echo "0dc004912ac66dd2f86ee3b01dd767b61b419965cd4420188260b8949104e8b1  $HIDDEN/txnkv/rangetask/rangetask_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/txnkv/rangetask/rangetask_bb_prop_test.go"
install_hidden "txnkv/rangetask/rangetask_bb_prop_test.go"
echo "0dc004912ac66dd2f86ee3b01dd767b61b419965cd4420188260b8949104e8b1  /app/txnkv/rangetask/rangetask_bb_prop_test.go" | sha256sum -c --status || checksum_fail "txnkv/rangetask/rangetask_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestRangeTaskCoverageProperty|TestRangeTaskContractExamples|TestRangeTaskErrorProperty|TestRangeTaskUnmentionedRandom)$' ./txnkv/rangetask/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
