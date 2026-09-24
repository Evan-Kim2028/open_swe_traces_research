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
echo "150b95f08523c967af25a6c4a8d5e924fdcde16d01000961a50137c547deec43  $HIDDEN/internal/pathutil/pathutil_bb_test.go" | sha256sum -c --status || checksum_fail "internal/pathutil/pathutil_bb_test.go"
install_hidden "internal/pathutil/pathutil_bb_test.go"
echo "150b95f08523c967af25a6c4a8d5e924fdcde16d01000961a50137c547deec43  /app/internal/pathutil/pathutil_bb_test.go" | sha256sum -c --status || checksum_fail "internal/pathutil/pathutil_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./internal/pathutil/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
