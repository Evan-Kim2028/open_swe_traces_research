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
echo "12b211f33a6acccb0a7a2f03b6c472db5a05b3b720ac57e6b7b94038a13e4163  $HIDDEN/rawkv/batch_delete_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/rawkv/batch_delete_bb_prop_test.go"
install_hidden "rawkv/batch_delete_bb_prop_test.go"
echo "12b211f33a6acccb0a7a2f03b6c472db5a05b3b720ac57e6b7b94038a13e4163  /app/rawkv/batch_delete_bb_prop_test.go" | sha256sum -c --status || checksum_fail "rawkv/batch_delete_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestBatchDeleteRemovesListedKeys|TestBatchDeleteContractExamples|TestBatchDeleteUnmentionedRandom)$' ./rawkv/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
