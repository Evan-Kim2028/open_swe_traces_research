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
echo "b53c34204dfe642cc9f2742ea67b819d64e0bf887cb5fe8b9679ef00e7eb2d49  $HIDDEN/txnkv/transaction/doactionbatches_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/txnkv/transaction/doactionbatches_bb_prop_test.go"
install_hidden "txnkv/transaction/doactionbatches_bb_prop_test.go"
echo "b53c34204dfe642cc9f2742ea67b819d64e0bf887cb5fe8b9679ef00e7eb2d49  /app/txnkv/transaction/doactionbatches_bb_prop_test.go" | sha256sum -c --status || checksum_fail "txnkv/transaction/doactionbatches_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestPrewriteBatchSizeProperty|TestMultiRegionDispatchProperty|TestPrimaryFirstPrewriteProperty|TestDoActionBatchesContractExamples|TestDoActionBatchesUnmentionedRandom)$' ./txnkv/transaction/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
