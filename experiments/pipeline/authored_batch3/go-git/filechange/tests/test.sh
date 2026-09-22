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
echo "bd1f0252c2adc0e5d221cd986180672adf772cdf7ac5614555c2a2049ccde838  $HIDDEN/plumbing/object/filechange_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/object/filechange_bb_test.go"
install_hidden "plumbing/object/filechange_bb_test.go"
echo "bd1f0252c2adc0e5d221cd986180672adf772cdf7ac5614555c2a2049ccde838  /app/plumbing/object/filechange_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/object/filechange_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./plumbing/object/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
