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
echo "67de6eafb4583d78ca96a4351d1533804494b4ea9f93b9acc31b56a8d2f225e4  $HIDDEN/expr/secschemes_bb_test.go" | sha256sum -c --status || checksum_fail "hidden/expr/secschemes_bb_test.go"
install_hidden "expr/secschemes_bb_test.go"
echo "67de6eafb4583d78ca96a4351d1533804494b4ea9f93b9acc31b56a8d2f225e4  /app/expr/secschemes_bb_test.go" | sha256sum -c --status || checksum_fail "expr/secschemes_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./expr/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
