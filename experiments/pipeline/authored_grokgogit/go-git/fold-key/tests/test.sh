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
echo "725f1f6b84bf582b0ba8a6c13643b642c8a2ab7d2a84c4e7e7fb5cd29354a848  $HIDDEN/plumbing/format/config/fold_key_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/config/fold_key_bb_test.go"
install_hidden "plumbing/format/config/fold_key_bb_test.go"
echo "725f1f6b84bf582b0ba8a6c13643b642c8a2ab7d2a84c4e7e7fb5cd29354a848  /app/plumbing/format/config/fold_key_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/config/fold_key_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./plumbing/format/config/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
