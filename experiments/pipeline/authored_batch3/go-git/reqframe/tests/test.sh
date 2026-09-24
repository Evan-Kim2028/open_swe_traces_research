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
echo "9b213f66cfc467f0cdb5574e0d3a94029a709d95b5a5977856cf21b25168625e  $HIDDEN/plumbing/protocol/packp/reqframe_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/protocol/packp/reqframe_bb_test.go"
install_hidden "plumbing/protocol/packp/reqframe_bb_test.go"
echo "9b213f66cfc467f0cdb5574e0d3a94029a709d95b5a5977856cf21b25168625e  /app/plumbing/protocol/packp/reqframe_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/protocol/packp/reqframe_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./plumbing/protocol/packp/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
