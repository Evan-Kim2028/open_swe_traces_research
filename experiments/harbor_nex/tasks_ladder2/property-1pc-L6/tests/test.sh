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
echo "82e146c28a91dd513ebf870e182a4b50b18e1eac0ec19f1f9ceb4eac009a74d8  $HIDDEN/txnkv/transaction/onepc_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/txnkv/transaction/onepc_prop_test.go"
install_hidden "txnkv/transaction/onepc_prop_test.go"
echo "82e146c28a91dd513ebf870e182a4b50b18e1eac0ec19f1f9ceb4eac009a74d8  /app/txnkv/transaction/onepc_prop_test.go" | sha256sum -c --status || checksum_fail "txnkv/transaction/onepc_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestOnePCAsyncDecisionProperty|TestOnePCContractExamples|TestOnePCUnmentionedRandom)$' ./txnkv/transaction/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
