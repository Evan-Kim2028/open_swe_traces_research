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
echo "fda70b13eca54d8fa9825eb7760761ada5ea93b0be3c836df492172d386f6b2a  $HIDDEN/github/createfork_bb_test.go" | sha256sum -c --status || checksum_fail "hidden/github/createfork_bb_test.go"
install_hidden "github/createfork_bb_test.go"
echo "fda70b13eca54d8fa9825eb7760761ada5ea93b0be3c836df492172d386f6b2a  /app/github/createfork_bb_test.go" | sha256sum -c --status || checksum_fail "github/createfork_bb_test.go"
if go test -count=1 -timeout 15m -run '^(TestDetail01|TestDetail02|TestDetail03)$' ./github; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
