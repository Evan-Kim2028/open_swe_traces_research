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
echo "6c38e62e6ea46b7781ca884935dba1bd03529a87001a0a09642810cabb36725a  $HIDDEN/rawkv/delete_range_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/rawkv/delete_range_bb_prop_test.go"
install_hidden "rawkv/delete_range_bb_prop_test.go"
echo "6c38e62e6ea46b7781ca884935dba1bd03529a87001a0a09642810cabb36725a  /app/rawkv/delete_range_bb_prop_test.go" | sha256sum -c --status || checksum_fail "rawkv/delete_range_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestDeleteRangeHalfOpenProperty|TestDeleteRangeContractExamples|TestDeleteRangeUnmentionedRandom)$' ./rawkv/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
