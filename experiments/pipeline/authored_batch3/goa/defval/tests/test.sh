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
echo "8c8a8e44ea47c01daa928d2227419463cd181044d0bff8b1ed774684f9b4152c  $HIDDEN/expr/defval_bb_test.go" | sha256sum -c --status || checksum_fail "hidden/expr/defval_bb_test.go"
install_hidden "expr/defval_bb_test.go"
echo "8c8a8e44ea47c01daa928d2227419463cd181044d0bff8b1ed774684f9b4152c  /app/expr/defval_bb_test.go" | sha256sum -c --status || checksum_fail "expr/defval_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./expr/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
