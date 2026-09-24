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
echo "8eb3fc565ea34c3f5eb9e4fc2ba9c8980fa74b86bb82cdaa598a6c8dfc5cb524  $HIDDEN/plumbing/protocol/capability/capability_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/protocol/capability/capability_bb_test.go"
install_hidden "plumbing/protocol/capability/capability_bb_test.go"
echo "8eb3fc565ea34c3f5eb9e4fc2ba9c8980fa74b86bb82cdaa598a6c8dfc5cb524  /app/plumbing/protocol/capability/capability_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/protocol/capability/capability_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./plumbing/protocol/capability/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
