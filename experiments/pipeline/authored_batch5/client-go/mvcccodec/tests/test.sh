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
echo "1486100fcc9470ed0f86c07880be189d0578f4bf86aa1dcc9bfd151c143c17df  $HIDDEN/internal/mockstore/mockkv/mvcccodec_bb_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/mockstore/mockkv/mvcccodec_bb_test.go"
install_hidden "internal/mockstore/mockkv/mvcccodec_bb_test.go"
echo "1486100fcc9470ed0f86c07880be189d0578f4bf86aa1dcc9bfd151c143c17df  /app/internal/mockstore/mockkv/mvcccodec_bb_test.go" | sha256sum -c --status || checksum_fail "internal/mockstore/mockkv/mvcccodec_bb_test.go"
if go test -count=1 -timeout 15m -run '^(TestDetail01|TestDetail02|TestDetail03|TestDetail04|TestDetail05|TestDetail06)$' ./internal/mockstore/mockkv; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
