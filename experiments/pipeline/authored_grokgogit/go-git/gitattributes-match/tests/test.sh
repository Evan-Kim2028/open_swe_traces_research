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
echo "3a6fb252722e0953319fa0b2863bfa0881bb26c9b1401994e8c885b81247a95b  $HIDDEN/plumbing/format/gitattributes/gitattributes_match_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/gitattributes/gitattributes_match_bb_test.go"
install_hidden "plumbing/format/gitattributes/gitattributes_match_bb_test.go"
echo "3a6fb252722e0953319fa0b2863bfa0881bb26c9b1401994e8c885b81247a95b  /app/plumbing/format/gitattributes/gitattributes_match_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/gitattributes/gitattributes_match_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./plumbing/format/gitattributes/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
