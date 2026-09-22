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
echo "36a790c4a83f40d2355c354d29db2bb26940e1cbd67d08e85235b3b64e311c68  $HIDDEN/plumbing/format/diff/unidiff_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/diff/unidiff_bb_test.go"
install_hidden "plumbing/format/diff/unidiff_bb_test.go"
echo "36a790c4a83f40d2355c354d29db2bb26940e1cbd67d08e85235b3b64e311c68  /app/plumbing/format/diff/unidiff_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/diff/unidiff_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./plumbing/format/diff/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
