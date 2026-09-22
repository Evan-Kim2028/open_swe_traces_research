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
echo "0ec44c0b287858a2aafd3f34275f62208e62a42cca41c78fff6d7f5ea1413f25  $HIDDEN/plumbing/format/commitgraph/cgenc_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/commitgraph/cgenc_bb_test.go"
install_hidden "plumbing/format/commitgraph/cgenc_bb_test.go"
echo "0ec44c0b287858a2aafd3f34275f62208e62a42cca41c78fff6d7f5ea1413f25  /app/plumbing/format/commitgraph/cgenc_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/commitgraph/cgenc_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./plumbing/format/commitgraph/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
