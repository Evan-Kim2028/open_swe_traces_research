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
echo "26d4bf94b62e4c5fa259244c6f8b4bac5760a113e6ad82d3abb9065e7deafc92  $HIDDEN/plumbing/protocol/packp/ulreq_encode_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/protocol/packp/ulreq_encode_bb_test.go"
install_hidden "plumbing/protocol/packp/ulreq_encode_bb_test.go"
echo "26d4bf94b62e4c5fa259244c6f8b4bac5760a113e6ad82d3abb9065e7deafc92  /app/plumbing/protocol/packp/ulreq_encode_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/protocol/packp/ulreq_encode_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./plumbing/protocol/packp/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
