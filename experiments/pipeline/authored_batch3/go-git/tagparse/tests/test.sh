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
echo "04366d0c4f38ba4fb2f06dfa3c1f674c7e873b7956281b8cba20809da41cad00  $HIDDEN/plumbing/object/tagparse_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/object/tagparse_bb_test.go"
install_hidden "plumbing/object/tagparse_bb_test.go"
echo "04366d0c4f38ba4fb2f06dfa3c1f674c7e873b7956281b8cba20809da41cad00  /app/plumbing/object/tagparse_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/object/tagparse_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./plumbing/object/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
