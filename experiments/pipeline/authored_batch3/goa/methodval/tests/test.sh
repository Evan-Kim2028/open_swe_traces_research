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
echo "07cd931ac71c9eb269a66345bb20ec784f9e2bc2861df82676f63deecb8919fd  $HIDDEN/expr/methodval_bb_test.go" | sha256sum -c --status || checksum_fail "hidden/expr/methodval_bb_test.go"
install_hidden "expr/methodval_bb_test.go"
echo "07cd931ac71c9eb269a66345bb20ec784f9e2bc2861df82676f63deecb8919fd  /app/expr/methodval_bb_test.go" | sha256sum -c --status || checksum_fail "expr/methodval_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./expr/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
