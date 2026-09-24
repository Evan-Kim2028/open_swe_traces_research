#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
mkdir -p /logs/artifacts
if [ -d /pristine ]; then
  ( cd / && if command -v git >/dev/null 2>&1; then git diff --no-index --no-color pristine app; else diff -ruN pristine app; fi )     | sed -e 's|a/pristine/|a/|g' -e 's|b/app/|b/|g' -e 's|a/app/|a/|g' -e 's|b/pristine/|b/|g' -e 's|^--- pristine/|--- a/|' -e 's|^+++ app/|+++ b/|'     > /logs/artifacts/agent.patch || true
fi
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
echo "b686750a56737a0aa0f78e448b6500fe354220dfa0ad5c0afda87371f2afabbe  $HIDDEN/util/pkg/hashing/hashparse_bb_test.go" | sha256sum -c --status || checksum_fail "hidden/util/pkg/hashing/hashparse_bb_test.go"
install_hidden "util/pkg/hashing/hashparse_bb_test.go"
echo "b686750a56737a0aa0f78e448b6500fe354220dfa0ad5c0afda87371f2afabbe  /app/util/pkg/hashing/hashparse_bb_test.go" | sha256sum -c --status || checksum_fail "util/pkg/hashing/hashparse_bb_test.go"
if go test -count=1 -timeout 15m -run '^(TestDetail01|TestDetail02|TestDetail03|TestDetail04|TestDetail05|TestDetail06|TestDetail07)$' ./util/pkg/hashing; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
