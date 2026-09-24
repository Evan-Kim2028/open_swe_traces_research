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
echo "231fbe317e642098f7b72269c6ab60c3387ca5bff490a98f8068bd5520508fd3  $HIDDEN/expr/rootval_bb_test.go" | sha256sum -c --status || checksum_fail "hidden/expr/rootval_bb_test.go"
install_hidden "expr/rootval_bb_test.go"
echo "231fbe317e642098f7b72269c6ab60c3387ca5bff490a98f8068bd5520508fd3  /app/expr/rootval_bb_test.go" | sha256sum -c --status || checksum_fail "expr/rootval_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./expr/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
