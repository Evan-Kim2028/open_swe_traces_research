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
echo "d4bc99277a4c14568226ed718b5b59b34cd2b8f90ab2a9e4291a70896a4aadd9  $HIDDEN/expr/grpcend_bb_test.go" | sha256sum -c --status || checksum_fail "hidden/expr/grpcend_bb_test.go"
install_hidden "expr/grpcend_bb_test.go"
echo "d4bc99277a4c14568226ed718b5b59b34cd2b8f90ab2a9e4291a70896a4aadd9  /app/expr/grpcend_bb_test.go" | sha256sum -c --status || checksum_fail "expr/grpcend_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./expr/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
