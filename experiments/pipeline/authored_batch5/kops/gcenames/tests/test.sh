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
echo "17eaa93bef9952ebfc91443403572d744cf66be4f97e0d686361ed951b3f0981  $HIDDEN/upup/pkg/fi/cloudup/gce/gcenames_bb_test.go" | sha256sum -c --status || checksum_fail "hidden/upup/pkg/fi/cloudup/gce/gcenames_bb_test.go"
install_hidden "upup/pkg/fi/cloudup/gce/gcenames_bb_test.go"
echo "17eaa93bef9952ebfc91443403572d744cf66be4f97e0d686361ed951b3f0981  /app/upup/pkg/fi/cloudup/gce/gcenames_bb_test.go" | sha256sum -c --status || checksum_fail "upup/pkg/fi/cloudup/gce/gcenames_bb_test.go"
if go test -count=1 -timeout 15m -run '^(TestDetail01|TestDetail02|TestDetail03|TestDetail04|TestDetail05|TestDetail06|TestDetail07|TestDetail08)$' ./upup/pkg/fi/cloudup/gce; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
