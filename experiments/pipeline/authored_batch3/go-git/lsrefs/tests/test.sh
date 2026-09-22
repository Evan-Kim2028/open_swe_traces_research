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
echo "ffbf0ac85d8cdcf06c0506b3e8ef9e12a7208ff910b0d6a2e2cb8eedbb341182  $HIDDEN/plumbing/protocol/packp/lsrefs_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/protocol/packp/lsrefs_bb_test.go"
install_hidden "plumbing/protocol/packp/lsrefs_bb_test.go"
echo "ffbf0ac85d8cdcf06c0506b3e8ef9e12a7208ff910b0d6a2e2cb8eedbb341182  /app/plumbing/protocol/packp/lsrefs_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/protocol/packp/lsrefs_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./plumbing/protocol/packp/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
