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
echo "dce6c405ee10d93692da16b8debb3ee43c3efb0a90dcd8d1b7dce67dea90f7c4  $HIDDEN/plumbing/format/reflog/reflog_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/reflog/reflog_bb_test.go"
install_hidden "plumbing/format/reflog/reflog_bb_test.go"
echo "dce6c405ee10d93692da16b8debb3ee43c3efb0a90dcd8d1b7dce67dea90f7c4  /app/plumbing/format/reflog/reflog_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/reflog/reflog_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./plumbing/format/reflog/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
