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
echo "eb578b2a35725c97a9c4055f0b646640758617dbf9064781e0cda2911790370a  $HIDDEN/expr/httperrexpr_bb_test.go" | sha256sum -c --status || checksum_fail "hidden/expr/httperrexpr_bb_test.go"
install_hidden "expr/httperrexpr_bb_test.go"
echo "eb578b2a35725c97a9c4055f0b646640758617dbf9064781e0cda2911790370a  /app/expr/httperrexpr_bb_test.go" | sha256sum -c --status || checksum_fail "expr/httperrexpr_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./expr/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
