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
echo "e30cdc3a8775deb09717c5c848e27a01ae3730e4b07d0b9e40a34ae106047aad  $HIDDEN/internal/client/connarray_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/client/connarray_bb_prop_test.go"
install_hidden "internal/client/connarray_bb_prop_test.go"
echo "e30cdc3a8775deb09717c5c848e27a01ae3730e4b07d0b9e40a34ae106047aad  /app/internal/client/connarray_bb_prop_test.go" | sha256sum -c --status || checksum_fail "internal/client/connarray_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestConnArraySendPoolProperty|TestConnArrayCancelTimeoutProperty|TestConnArrayCloseLifecycleProperty|TestConnArrayConcurrentSendProperty|TestConnArrayServerRestartProperty)$' ./internal/client/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
